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

package rlp

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

func TestWrapHexFail(t *testing.T) {
	assert.Panics(t, func() {
		MustWrapHex("! not hex")
	})
}

func TestEncodeData(t *testing.T) {

	d := Data{}
	assert.False(t, d.IsList())

	assert.Equal(t, []byte{}, int64ToMinimalBytes(0))

	assert.Equal(t, big.NewInt(0x7FFFFFFFFFFFFFF0).Bytes(), int64ToMinimalBytes(0x7FFFFFFFFFFFFFF0))

	assert.Equal(t, []byte{0x80}, WrapString("").Encode())

	assert.Equal(t, []byte{0x0f}, Data([]byte{0x0f}).Encode())

	assert.Equal(t, []byte{0x83, 'd', 'o', 'g'}, WrapString("dog").Encode())

	assert.Equal(t, []byte{0x00}, Data{0x00}.Encode())

	assert.Equal(t, loremIpsumRLPBytes, WrapString(loremIpsumString).Encode())

	expected := make([]byte, 56)
	expected[0] = 0xb7
	assert.Equal(t, expected, make(Data, 55).Encode())
}

func TestEncodeIntegers(t *testing.T) {

	assert.Equal(t, []byte{0x0f}, WrapInt(big.NewInt(0x0f)).Encode())

	assert.Equal(t, []byte{0x82, 0x04, 0x00}, WrapInt(big.NewInt(0x400)).Encode())

	assert.Equal(t, []byte{0x80}, WrapInt(big.NewInt(0)).Encode())

	assert.Equal(t, int64(0xfeedbeef), Data{0xfe, 0xed, 0xbe, 0xef}.Int().Int64())

	assert.Nil(t, Data(nil).Int())

}

func TestEncodeList(t *testing.T) {

	l := List{}
	assert.True(t, l.IsList())

	assert.Equal(t, []byte{0xc8, 0x83, 'c', 'a', 't', 0x83, 'd', 'o', 'g'},
		List{WrapString("cat"), WrapString("dog")}.Encode())

	assert.Equal(t, []byte{0xc0}, List{}.Encode())

	assert.Equal(t, []byte{
		0xc7,
		0xc0,
		0xc1,
		0xc0,
		0xc3,
		0xc0,
		0xc1,
		0xc0,
	}, List{
		List{},
		List{
			List{},
		},
		List{
			List{},
			List{
				List{},
			},
		},
	}.Encode())

	assert.Equal(t, []byte{
		0xc6,
		0x82,
		0x7a,
		0x77,
		0xc1,
		0x04,
		0x01,
	}, List{
		WrapString("zw"),
		List{
			WrapInt(big.NewInt(4)),
		},
		WrapInt(big.NewInt(1)),
	}.Encode())

}

func TestEncodeNil(t *testing.T) {

	assert.Equal(t, []byte{0x80}, (Data)(nil).Encode())

}

func TestEncodeZero(t *testing.T) {

	assert.Equal(t, []byte{0x80}, WrapInt(big.NewInt(0)).Encode())

}

func TestEncodeAddress(t *testing.T) {

	b, err := hex.DecodeString("497eedc4299dea2f2a364be10025d0ad0f702de3")
	assert.NoError(t, err)
	var a ethtypes.Address0xHex
	copy(a[0:20], b[0:20])

	d := WrapAddress(&a)
	aa, _, err := Decode(d.Encode())
	assert.NoError(t, err)
	assert.Equal(t, Data(b), aa.(Data))

	d1 := WrapAddress((*ethtypes.Address0xHex)(nil))
	assert.Equal(t, Data{}, d1)

}
