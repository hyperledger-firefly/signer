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

package rlp

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
)

// Data is an individual RLP Data element - or an "RLP string"
type Data []byte

// List is a list of RLP elements, which could be either Data or List elements
type List []Element

// Element is an interface implemented by both Data and List elements
type Element interface {
	// When true the Element can safely be cast to List, and when false the Element can safely be cast to Data
	IsList() bool
	// Encode converts the element to a byte array
	Encode() []byte
	// Safe function that will give an entry as data, to use the nil-safe functions on it to get the value (will be treated as nil data for list)
	ToData() Data
}

// WrapString converts a plain string to an RLP Data element for encoding
func WrapString(s string) Data {
	return Data(s)
}

// WrapString converts a positive integer to an RLP Data element for encoding
func WrapInt(i *big.Int) Data {
	return Data(i.Bytes())
}

// WrapAddress wraps an address, or writes empty data if the address is nil
func WrapAddress(a *ethtypes.Address0xHex) Data {
	if a == nil {
		return Data{}
	}
	return Data(a[0:20])
}

// WrapHex converts a hex encoded string (with or without 0x prefix) to an RLP Data element for encoding
func WrapHex(s string) (Data, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}
	return Data(b), nil
}

// MustWrapHex panics if hex decoding fails
func MustWrapHex(s string) Data {
	b, err := WrapHex(s)
	if err != nil {
		panic(err)
	}
	return b
}

// Int is a convenience function to convert the bytes within an RLP Data element to an integer (big endian encoding)
func (r Data) Int() *big.Int {
	if r == nil {
		return nil
	}
	i := new(big.Int)
	return i.SetBytes(r)
}

func (r Data) IntOrZero() *big.Int {
	if r == nil {
		return big.NewInt(0)
	}
	i := new(big.Int)
	return i.SetBytes(r)
}

func (r Data) BytesNotNil() []byte {
	if r == nil {
		return []byte{}
	}
	return r
}

func (r Data) Address() *ethtypes.Address0xHex {
	if r == nil || len(r) != 20 {
		return nil
	}
	return (*ethtypes.Address0xHex)(r)
}

// Encode encodes this individual RLP Data element
func (r Data) Encode() []byte {
	return encodeBytes(r, false)
}

// IsList is false for individual RLP Data elements
func (r Data) IsList() bool {
	return false
}

func (r Data) ToData() Data {
	return r
}

// Encode encodes the RLP List to a byte array, including recursing into child arrays
func (l List) Encode() []byte {
	if len(l) == 0 {
		return encodeBytes([]byte{}, true)
	}
	var concatenation []byte
	for _, entry := range l {
		concatenation = append(concatenation, entry.Encode()...)
	}
	return encodeBytes(concatenation, true)

}

// IsList returns true for list elements
func (l List) IsList() bool {
	return true
}

func (l List) ToData() Data {
	// This allows code to not worry about lots of type checking - a list is treated as nil data
	return nil
}
