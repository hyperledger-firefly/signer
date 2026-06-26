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

package eip712

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hyperledger-firefly/signer/pkg/abi"
	"github.com/stretchr/testify/assert"
)

func TestABISimpleStruct(t *testing.T) {

	// // Extracted from ABI in this solidity:
	// // solc --combined-json abi eip712_examples.sol > eip712_examples.json
	// pragma solidity ^0.8.0;
	// contract EIP712Examples {
	// 	struct Person {
	// 	  string name;
	// 	  address wallet;
	//  }
	// 	struct Mail {
	// 	  Person from;
	// 	  Person to;
	// 	  string contents;
	//  }
	// 	constructor() {}
	// 	function mail() public pure returns (Mail memory) {
	// 	  return Mail(Person("", address(0)), Person("", address(0)), "");
	// 	}
	// }
	mailABI := []byte(`{
		"components": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "name",
						"type": "string"
					},
					{
						"internalType": "address",
						"name": "wallet",
						"type": "address"
					}
				],
				"internalType": "struct EIP712Examples.Person",
				"name": "from",
				"type": "tuple"
			},
			{
				"components": [
					{
						"internalType": "string",
						"name": "name",
						"type": "string"
					},
					{
						"internalType": "address",
						"name": "wallet",
						"type": "address"
					}
				],
				"internalType": "struct EIP712Examples.Person",
				"name": "to",
				"type": "tuple"
			},
			{
				"internalType": "string",
				"name": "contents",
				"type": "string"
			}
		],
		"internalType": "struct EIP712Examples.Mail",
		"name": "",
		"type": "tuple"
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	pt, ts, err := ABItoTypedDataV4(context.Background(), tc)
	assert.NoError(t, err)
	assert.Equal(t, "Mail", pt)
	assert.Equal(t, TypeSet{
		"Person": Type{
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "wallet",
				Type: "address",
			},
		},
		"Mail": Type{
			{
				Name: "from",
				Type: "Person",
			},
			{
				Name: "to",
				Type: "Person",
			},
			{
				Name: "contents",
				Type: "string",
			},
		},
	}, ts)

}

func TestABIArraysAndTypes(t *testing.T) {

	// // solc --combined-json abi eip712_examples.sol > eip712_examples.json
	// pragma solidity ^0.8.0;
	// contract EIP712Examples {
	//   struct TopLevel {
	//     string[] strings;
	//     int32[5] ints;
	//     bool[][] multidim;
	//     Nested[] nested;
	//   }
	//   struct Nested {
	//     uint[] unaliased;
	//     bytes[5] bytestrings;
	//     bytes1[8] eightbytes;
	//   }
	//   constructor() {}
	//   function mail(TopLevel memory) public pure {}
	// }
	mailABI := []byte(`{
		"components": [
			{
				"internalType": "string[]",
				"name": "strings",
				"type": "string[]"
			},
			{
				"internalType": "int32[5]",
				"name": "ints",
				"type": "int32[5]"
			},
			{
				"internalType": "bool[][]",
				"name": "multidim",
				"type": "bool[][]"
			},
			{
				"components": [
					{
						"internalType": "uint256[]",
						"name": "unaliased",
						"type": "uint256[]"
					},
					{
						"internalType": "bytes[5]",
						"name": "bytestrings",
						"type": "bytes[5]"
					},
					{
						"internalType": "bytes1[8]",
						"name": "eightbytes",
						"type": "bytes1[8]"
					}
				],
				"internalType": "struct EIP712Examples.Nested[]",
				"name": "nested",
				"type": "tuple[]"
			}
		],
		"internalType": "struct EIP712Examples.TopLevel",
		"name": "",
		"type": "tuple"
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	pt, ts, err := ABItoTypedDataV4(context.Background(), tc)
	assert.NoError(t, err)
	assert.Equal(t, "TopLevel", pt)
	assert.Equal(t, TypeSet{
		"Nested": Type{
			{
				Name: "unaliased",
				Type: "uint256[]",
			},
			{
				Name: "bytestrings",
				Type: "bytes[5]",
			},
			{
				Name: "eightbytes",
				Type: "bytes1[8]",
			},
		},
		"TopLevel": Type{
			{
				Name: "strings",
				Type: "string[]",
			},
			{
				Name: "ints",
				Type: "int32[5]",
			},
			{
				Name: "multidim",
				Type: "bool[][]",
			},
			{
				Name: "nested",
				Type: "Nested[]",
			},
		},
	}, ts)

}

func TestABINotTuple(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "uint256"
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, _, err = ABItoTypedDataV4(context.Background(), tc)
	assert.Regexp(t, "FF22074", err)

}

func TestABINoInternalType(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple",
		"components": []
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, _, err = ABItoTypedDataV4(context.Background(), tc)
	assert.Regexp(t, "FF22075", err)

}

func TestABINoInternalTypeChild(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple",
		"internalType": "struct MyType",
		"components": [
			{
				"name": "noInternal",
				"type": "tuple",
				"components": []
			}
		]
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, _, err = ABItoTypedDataV4(context.Background(), tc)
	assert.Regexp(t, "FF22075", err)

}

func TestMapElementaryABITypeNonElementary(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple",
		"internalType": "struct MyType",
		"components": []
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, err = mapElementaryABIType(context.Background(), tc)
	assert.Regexp(t, "FF22069", err)

}

func TestMapABITypeBadArray(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple[]",
		"components": []
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, err = mapABIType(context.Background(), tc)
	assert.Regexp(t, "FF22075", err)

}

func TestAddABITypesFailToExtractStructName(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple"
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	err = addABITypes(context.Background(), tc, TypeSet{})
	assert.Regexp(t, "FF22075", err)

}

func TestABIUnsupportedType(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple",
		"internalType": "struct MyType",
		"components": [
			{
				"name": "fixed",
				"type": "fixed256x18"
			}
		]
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, _, err = ABItoTypedDataV4(context.Background(), tc)
	assert.Regexp(t, "FF22072", err)

}

func TestABINestedError(t *testing.T) {

	mailABI := []byte(`{
		"name": "",
		"type": "tuple",
		"internalType": "struct MyType",
		"components": [
			{
				"name": "nested1",
				"type": "tuple",
				"internalType": "struct Nested1",
				"components": [
					{
						"name": "nested1",
						"type": "tuple",
						"internalType": "",
						"components": []
					}
				]
			}
		]
	}`)
	var abiElem abi.Parameter
	err := json.Unmarshal(mailABI, &abiElem)
	assert.NoError(t, err)

	tc, err := abiElem.TypeComponentTree()
	assert.NoError(t, err)

	_, _, err = ABItoTypedDataV4(context.Background(), tc)
	assert.Regexp(t, "FF22075", err)

}
