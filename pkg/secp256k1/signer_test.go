// Copyright © 2022 Kaleido, Inc.
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

package secp256k1

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

const ethMessagePrefix = "\u0019Ethereum Signed Message:\n"

// Test data directly taken from:
// https://github.com/web3j/web3j/blob/master/crypto/src/test/java/org/web3j/crypto/SignTest.java
var (
	sampleMessage    = "A test message"
	samplePrivateKey = "a392604efc2fad9c0b3da43b5f698a2e3f270f170d859912be0d54742275c5f6"
	samplePublicKey  = "0x506bc1dc099358e5137292f4efdd57e400f29ba5132aa5d12b18dac1c1f6aab" +
		"a645c0b7b58158babbfa6c6cd5a48aa7340a8749176b120e8516216787a13dc76"
	sampleAddress = "0xef678007d18427e6022059dbc264f27507cd1ffc"
)

func addEthMessagePrefix(message []byte) []byte {
	b := new(bytes.Buffer)
	b.Write([]byte(ethMessagePrefix))
	b.Write([]byte(strconv.FormatInt(int64(len(message)), 10)))
	b.Write(message)
	return b.Bytes()
}

func testKeyPair(t *testing.T) *KeyPair {
	keyBytes, err := hex.DecodeString(samplePrivateKey)
	assert.NoError(t, err)
	keypair, err := NewSecp256k1KeyPair(keyBytes)
	assert.NoError(t, err)
	return keypair
}

func TestValidateSampleData(t *testing.T) {
	// Validate the above sample data is consistent in the base secp256k1 key management layer
	keypair := testKeyPair(t)
	assert.Equal(t, samplePrivateKey, ((ethtypes.HexBytesPlain)(keypair.PrivateKeyBytes())).String())
	var pubkey ethtypes.HexBytes0xPrefix = keypair.PublicKeyBytes()
	assert.Equal(t, samplePublicKey, pubkey.String())
	var addr ethtypes.Address0xHex = ethtypes.Address0xHex(keypair.Address)
	assert.Equal(t, sampleAddress, addr.String())
}

func TestSignMessage(t *testing.T) {

	keypair := testKeyPair(t)
	sig, err := keypair.Sign(addEthMessagePrefix([]byte(sampleMessage)))
	assert.NoError(t, err)

	assert.Equal(t, int64(28), sig.V.Int64())
	assert.Equal(t, "0464eee9e2fe1a10ffe48c78b80de1ed8dcf996f3f60955cb2e03cb21903d930", ((ethtypes.HexBytesPlain)(sig.R.Bytes())).String())
	assert.Equal(t, "06624da478b3f862582e85b31c6a21c6cae2eee2bd50f55c93c4faad9d9c8d7f", ((ethtypes.HexBytesPlain)(sig.S.Bytes())).String())

	err = sig.UpdateEIP155(1001)
	assert.NoError(t, err)
	assert.Equal(t, int64(2038), sig.V.Int64())
}

func TestUpdateEIP155Error(t *testing.T) {
	sig := &SignatureData{
		V: big.NewInt(42),
		R: new(big.Int),
		S: new(big.Int),
	}
	err := sig.UpdateEIP155(1001)
	assert.Regexp(t, "cannot recover Y-parity value", err)
}

func TestUpdateEIP2930Error(t *testing.T) {
	sig := &SignatureData{
		V: big.NewInt(42),
		R: new(big.Int),
		S: new(big.Int),
	}
	err := sig.UpdateEIP2930()
	assert.Regexp(t, "cannot recover Y-parity value", err)
}

func TestUpdateOriginalError(t *testing.T) {
	sig := &SignatureData{
		V: big.NewInt(42),
		R: new(big.Int),
		S: new(big.Int),
	}
	err := sig.UpdateOriginal()
	assert.Regexp(t, "invalid V value in signature", err)
}

func TestSignFailNil(t *testing.T) {

	_, err := (*KeyPair)(nil).Sign(addEthMessagePrefix([]byte(sampleMessage)))
	assert.Regexp(t, "nil signer", err)

}
