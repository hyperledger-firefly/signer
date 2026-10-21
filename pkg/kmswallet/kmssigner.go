// Copyright © 2026 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kmswallet

import (
	"context"

	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

type kmsSigner struct {
	wallet *KMSWallet
}

// Sign implements secp256k1.Signer — hashes the message with Keccak-256 then signs via KMS.
func (k *kmsSigner) Sign(message []byte) (*secp256k1.SignatureData, error) {
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(message)
	hashed := msgHash.Sum(nil)
	return k.SignDirect(hashed)
}

// SignDirect implements secp256k1.SignerDirect — signs a pre-hashed (32-byte) message via KMS.
func (k *kmsSigner) SignDirect(message []byte) (*secp256k1.SignatureData, error) {
	return k.wallet.signDirect(context.Background(), message)
}
