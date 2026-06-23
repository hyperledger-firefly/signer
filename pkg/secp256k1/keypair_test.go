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
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratedKeyRoundTrip(t *testing.T) {

	keypair, err := GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	b := keypair.PrivateKeyBytes()
	keypair2, err := NewSecp256k1KeyPair(b)
	assert.NoError(t, err)

	assert.Equal(t, keypair.PrivateKeyBytes(), keypair2.PrivateKeyBytes())
	assert.True(t, keypair.PublicKey.IsEqual(keypair2.PublicKey))

	data := []byte("hello world")
	sig, err := keypair.Sign(data)
	assert.NoError(t, err)

	// Legacy 27/28 - pre EIP-155
	addr, err := sig.Recover(data, 0)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address, *addr)

	err = sig.UpdateOriginal() // this should be a no-op for 27/28
	assert.NoError(t, err)
	addr, err = sig.Recover(data, 0)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address, *addr)

	// Latest 0/1 - EIP-1559 / EIP-2930
	err = sig.UpdateEIP2930()
	assert.NoError(t, err)
	addr, err = sig.Recover(data, 0)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address, *addr)
	sig.V.SetInt64(sig.V.Int64() + 27)

	// Chain ID encoded in V value - EIP-155
	err = sig.UpdateEIP155(1001)
	assert.NoError(t, err)
	addr, err = sig.Recover(data, 1001)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address, *addr)

	sigRSV := sig.CompactRSV()
	sig2, err := DecodeCompactRSV(context.Background(), sigRSV)
	assert.NoError(t, err)
	addr, err = sig2.Recover(data, 1001)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address, *addr)

	_, err = DecodeCompactRSV(context.Background(), []byte("wrong"))
	assert.Regexp(t, "FF22087", err)

	_, err = sig.Recover(data, 42)
	assert.Regexp(t, "invalid V value in signature", err)

	sigBad := &SignatureData{
		V: big.NewInt(27),
		R: new(big.Int),
		S: new(big.Int),
	}
	_, err = sigBad.Recover(data, 0)
	assert.Error(t, err)

}
