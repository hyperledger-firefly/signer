// Copyright © 2023 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ethsigner

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/hyperledger-firefly/common/pkg/ffapi"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/mocks/secp256k1mocks"
	"github.com/hyperledger-firefly/signer/pkg/eip712"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSignTypedDataV4(t *testing.T) {

	// We use a simple empty message payload
	payload := &eip712.TypedData{
		PrimaryType: eip712.EIP712Domain,
	}
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	ctx := context.Background()
	sig, err := SignTypedDataV4(ctx, keypair, payload)
	assert.NoError(t, err)

	b, err := json.Marshal(sig)
	assert.NoError(t, err)
	log.L(context.Background()).Infof("Signature: %s", b)

	foundSig := &secp256k1.SignatureData{
		V: sig.V.BigInt(),
		R: new(big.Int),
		S: new(big.Int),
	}
	foundSig.R.SetBytes(sig.R)
	foundSig.S.SetBytes(sig.S)

	signaturePayload := ethtypes.HexBytes0xPrefix(sig.Hash)
	addr, err := foundSig.RecoverDirect(signaturePayload, -1 /* chain id is in the domain (not applied EIP-155 style to the V value) */)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address.String(), addr.String())

	encoded, err := eip712.EncodeTypedDataV4(ctx, payload)
	assert.NoError(t, err)

	// Check all is as we expect
	assert.Equal(t, "0x8d4a3f4082945b7879e2b55f181c31a77c8c0a464b70669458abbaaf99de4c38", encoded.String())
	assert.Equal(t, "0x8d4a3f4082945b7879e2b55f181c31a77c8c0a464b70669458abbaaf99de4c38", signaturePayload.String())
}

func TestSignTypedDataV4BadPayload(t *testing.T) {

	payload := &eip712.TypedData{
		PrimaryType: "missing",
	}

	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = SignTypedDataV4(ctx, keypair, payload)
	assert.Regexp(t, "FF22073", err)
}

func TestSignTypedDataV4SignFail(t *testing.T) {

	payload := &eip712.TypedData{
		PrimaryType: eip712.EIP712Domain,
	}

	msn := &secp256k1mocks.SignerDirect{}
	msn.On("SignDirect", mock.Anything).Return(nil, fmt.Errorf("pop"))

	ctx := context.Background()
	_, err := SignTypedDataV4(ctx, msn, payload)
	assert.Regexp(t, "pop", err)
}

func TestMessage_2(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p eip712.TypedData
	err := json.Unmarshal([]byte(`{
		"domain": {
			"name": "test-app",
			"version": "1",
			"chainId": 31337,
			"verifyingContract": "0x9fe46736679d2d9a65f0992f2272de9f3c7fa6e0"
		},
		"types": {
			"Issuance": [
				{
					"name": "amount",
					"type": "uint256"
				},
				{
					"name": "to",
					"type": "address"
				}
			],
			"EIP712Domain": [
				{
					"name": "name",
					"type": "string"
				},
				{
					"name": "version",
					"type": "string"
				},
				{
					"name": "chainId",
					"type": "uint256"
				},
				{
					"name": "verifyingContract",
					"type": "address"
				}
			]
		},
		"primaryType": "Issuance",
		"message": {
			"amount": "1000",
			"to": "0xce3a47d24140cca16f8839357ca5fada44a1baef"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := eip712.EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xb0132202fa81cafac0e405917f86705728ba02912d185065697cc4ba4e61aec3", ed.String())

	keys, err := secp256k1.NewSecp256k1KeyPair([]byte(`8d01666832be7eb2dbd57cd3d4410d0231a91533f895de76d0930c689618aefd`))
	assert.NoError(t, err)
	assert.Equal(t, "0xbcef501facf72ddacdb055acc2716786ff038728", keys.Address.String())

	signed, err := SignTypedDataV4(ctx, keys, &p)
	assert.NoError(t, err)

	assert.Equal(t, "0xb0132202fa81cafac0e405917f86705728ba02912d185065697cc4ba4e61aec3", signed.Hash.String())

	// The golang convention is V, R, S for the compact signature (differing from Ethereum's convention of R, S, V)
	golangCompactSignature := make([]byte, 65)
	golangCompactSignature[0] = signed.SignatureRSV[64]
	copy(golangCompactSignature[1:33], signed.SignatureRSV[0:32])
	copy(golangCompactSignature[33:65], signed.SignatureRSV[32:64])

	fmt.Printf("%s\n", ethtypes.HexBytes0xPrefix(golangCompactSignature))
	fmt.Printf("%s\n", ethtypes.HexBytes0xPrefix(signed.SignatureRSV))

	pubKey, _, err := ecdsa.RecoverCompact(golangCompactSignature, signed.Hash)
	assert.NoError(t, err)
	assert.Equal(t, "0xbcef501facf72ddacdb055acc2716786ff038728", secp256k1.PublicKeyToAddress(pubKey).String())
}

func TestEIP712ResultDocumented(t *testing.T) {
	ffapi.CheckObjectDocumented(&EIP712Result{})
}
