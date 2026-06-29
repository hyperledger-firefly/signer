// Copyright © 2024 Kaleido, Inc.
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

	"github.com/hyperledger-firefly/signer/pkg/eip712"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
)

type EIP712Result struct {
	Hash         ethtypes.HexBytes0xPrefix `ffstruct:"EIP712Result" json:"hash"`
	SignatureRSV ethtypes.HexBytes0xPrefix `ffstruct:"EIP712Result" json:"signatureRSV"`
	V            ethtypes.HexInteger       `ffstruct:"EIP712Result" json:"v"`
	R            ethtypes.HexBytes0xPrefix `ffstruct:"EIP712Result" json:"r"`
	S            ethtypes.HexBytes0xPrefix `ffstruct:"EIP712Result" json:"s"`
}

func SignTypedDataV4(ctx context.Context, signer secp256k1.SignerDirect, payload *eip712.TypedData) (*EIP712Result, error) {
	encodedData, err := eip712.EncodeTypedDataV4(ctx, payload)
	if err != nil {
		return nil, err
	}
	// Note that signer.Sign performs the hash
	sig, err := signer.SignDirect(encodedData)
	if err != nil {
		return nil, err
	}

	err = sig.UpdateOriginal()
	if err != nil {
		return nil, err
	}

	signatureBytes := make([]byte, 65)
	sig.R.FillBytes(signatureBytes[0:32])
	sig.S.FillBytes(signatureBytes[32:64])
	signatureBytes[64] = byte(sig.V.Int64())

	return &EIP712Result{
		Hash: encodedData,
		// Include the clearly distinguished V, R & S values of the signature
		V: ethtypes.HexInteger(*sig.V),
		R: sig.R.FillBytes(make([]byte, 32)),
		S: sig.S.FillBytes(make([]byte, 32)),
		// the Ethereum convention (which is different to the Golang convention) is to encode compact signatures as
		// 65 bytes - R (32B), S (32B), V (1B)
		// See: https://github.com/OpenZeppelin/openzeppelin-contracts/blob/7294d34c17ca215c201b3772ff67036fa4b1ef12/contracts/utils/cryptography/ECDSA.sol#L56-L73
		SignatureRSV: signatureBytes,
	}, nil
}
