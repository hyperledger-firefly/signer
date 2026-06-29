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

package abi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/ffapi"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleABI1 = `[
    {
      "name": "foo",
      "type": "function",
      "inputs": [
        {
            "name": "a",
            "type": "tuple",
            "internalType": "struct AType",
            "components": [
                {
                    "name": "b",
                    "type": "uint"
                },
                {
                    "name": "c",
                    "type": "string[2]"
                },
                {
                    "name": "d",
                    "type": "bytes"
                }
            ]
        }
      ],
      "outputs": [
            {
                "name": "e",
                "type": "uint256"
            },
            {
                "name": "f",
                "type": "string"
            }
      ]
    }
  ]`

const sampleABI2 = `[
    {
      "name": "foo",
      "type": "function",
      "inputs": [
        {
            "name": "a",
            "type": "uint8"
        },
        {
            "name": "b",
            "type": "int"
        },
        {
            "name": "c",
            "type": "address"
        },
        {
            "name": "d",
            "type": "bool"
        },
        {
            "name": "e",
            "type": "fixed64x10"
        },
        {
            "name": "f",
            "type": "ufixed"
        },
        {
            "name": "g",
            "type": "bytes10"
        },
        {
            "name": "h",
            "type": "bytes"
        },
        {
            "name": "i",
            "type": "function"
        },
        {
            "name": "j",
            "type": "string"
        }
      ],
      "outputs": []
    }
  ]`

const sampleABI3 = `[
    {
        "type": "constructor",
        "inputs": [
          {
              "name": "a",
              "type": "tuple",
              "components": [
                  {
                      "name": "b",
                      "type": "uint"
                  },
                  {
                      "name": "c",
                      "type": "string[2]"
                  },
                  {
                      "name": "d",
                      "type": "bytes"
                  }
              ]
          }
        ],
        "outputs": []
    },
    {
      "name": "foo",
      "type": "function",
      "inputs": [
        {
            "name": "a",
            "type": "tuple",
            "components": [
                {
                    "name": "b",
                    "type": "uint"
                },
                {
                    "name": "c",
                    "type": "string[2]"
                },
                {
                    "name": "d",
                    "type": "bytes"
                }
            ]
        }
      ],
      "outputs": []
    }
  ]`

const sampleABI4 = `[
    {
        "name": "simple",
        "type": "function",
        "inputs": [
          {
              "name": "a",
              "type": "string"
          }
        ],
        "outputs": []
    }
  ]`

const sampleABI5 = `[
    {
      "anonymous": false,
      "inputs": [
        {
          "components": [
            {
              "internalType": "address",
              "name": "owner",
              "type": "address"
            },
            {
              "internalType": "bytes32",
              "name": "locator",
              "type": "bytes32"
            }
          ],
          "indexed": false,
          "internalType": "struct AribtraryWidgets.Customer",
          "name": "customer",
          "type": "tuple"
        },
        {
          "components": [
            {
              "internalType": "string",
              "name": "description",
              "type": "string"
            },
            {
              "internalType": "uint256",
              "name": "price",
              "type": "uint256"
            },
            {
              "internalType": "string[]",
              "name": "attributes",
              "type": "string[]"
            }
          ],
          "indexed": false,
          "internalType": "struct AribtraryWidgets.Widget[]",
          "name": "widgets",
          "type": "tuple[]"
        },
        {
          "name": "account",
          "type": "address",
          "indexed": true
        }
      ],
      "name": "Invoiced",
      "type": "event"
    },
    {
      "inputs": [
        {
          "components": [
            {
              "components": [
                {
                  "internalType": "address",
                  "name": "owner",
                  "type": "address"
                },
                {
                  "internalType": "bytes32",
                  "name": "locator",
                  "type": "bytes32"
                }
              ],
              "internalType": "struct AribtraryWidgets.Customer",
              "name": "customer",
              "type": "tuple"
            },
            {
              "components": [
                {
                  "internalType": "string",
                  "name": "description",
                  "type": "string"
                },
                {
                  "internalType": "uint256",
                  "name": "price",
                  "type": "uint256"
                },
                {
                  "internalType": "string[]",
                  "name": "attributes",
                  "type": "string[]"
                }
              ],
              "internalType": "struct AribtraryWidgets.Widget[]",
              "name": "widgets",
              "type": "tuple[]"
            }
          ],
          "internalType": "struct AribtraryWidgets.Invoice",
          "name": "_invoice",
          "type": "tuple"
        }
      ],
      "name": "invoice",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    }
  ]`

func testABI(t *testing.T, abiJSON string) (abi ABI) {
	err := json.Unmarshal([]byte(abiJSON), &abi)
	assert.NoError(t, err)
	return abi
}

func TestDocsFunctionCallExample(t *testing.T) {

	transferABI := `[
        {
            "inputs": [
                {
                    "internalType": "address",
                    "name": "recipient",
                    "type": "address"
                },
                {
                    "internalType": "uint256",
                    "name": "amount",
                    "type": "uint256"
                }
            ],
            "name": "transfer",
            "outputs": [
                {
                    "internalType": "bool",
                    "name": "",
                    "type": "bool"
                }
            ],
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ]`

	// Parse the ABI definition
	var abi ABI
	_ = json.Unmarshal([]byte(transferABI), &abi)
	f := abi.Functions()["transfer"]

	// Parse some JSON input data conforming to the ABI
	encodedValueTree, _ := f.Inputs.ParseJSON([]byte(`{
        "recipient": "0x03706Ff580119B130E7D26C5e816913123C24d89",
        "amount": "1000000000000000000"
    }`))

	// We can serialize this directly to abi bytes
	abiData, _ := encodedValueTree.EncodeABIData()
	fmt.Println(hex.EncodeToString(abiData))
	// 00000000000000000000000003706ff580119b130e7d26c5e816913123c24d890000000000000000000000000000000000000000000000000de0b6b3a7640000

	// We can also serialize that to function call data, with the function selector prefix
	abiCallData, _ := f.EncodeCallData(encodedValueTree)

	// Decode those ABI bytes back again, verifying the function selector
	decodedValueTree, _ := f.DecodeCallData(abiCallData)

	// Serialize back to JSON with default formatting - note the keys are alphabetically ordered
	jsonData, _ := decodedValueTree.JSON()
	fmt.Println(string(jsonData))
	// {"amount":"1000000000000000000","recipient":"03706ff580119b130e7d26c5e816913123c24d89"}

	// Use a custom serializer to get ordered array output, hex integers, and 0x prefixes
	// - Check out FormatAsSelfDescribingArrays for a format with embedded type information
	jsonData2, _ := NewSerializer().
		SetFormattingMode(FormatAsFlatArrays).
		SetIntSerializer(HexIntSerializer0xPrefix).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SerializeJSON(decodedValueTree)
	fmt.Println(string(jsonData2))
	// ["0x03706ff580119b130e7d26c5e816913123c24d89","0xde0b6b3a7640000"]

	// Test that signature gets hashed correctly via Keccak-256
	sigHash, _ := f.SignatureHash()

	// Test validation - not for copy/paste to docs
	assert.Equal(t, `00000000000000000000000003706ff580119b130e7d26c5e816913123c24d890000000000000000000000000000000000000000000000000de0b6b3a7640000`, hex.EncodeToString(abiData))
	assert.Equal(t, `{"amount":"1000000000000000000","recipient":"03706ff580119b130e7d26c5e816913123c24d89"}`, string(jsonData))
	assert.Equal(t, `["0x03706ff580119b130e7d26c5e816913123c24d89","0xde0b6b3a7640000"]`, string(jsonData2))
	assert.Equal(t, "0xa9059cbb2ab09eb219583f4a59a5d0623ade346d962bcd4e46b11da047c9049b", sigHash.String())

	// Check the solidity def
	solDef, childStructs, err := f.SolidityDef()
	assert.NoError(t, err)
	assert.Equal(t, "function transfer(address recipient, uint256 amount) external returns (bool) { }", solDef)
	assert.Empty(t, childStructs)
}

func TestTLdr(t *testing.T) {

	sampleABI, _ := ParseABI([]byte(`[
        {
            "name": "transfer",
            "inputs": [
                {"name": "recipient", "internalType": "address", "type": "address" },
                {"name": "amount", "internalType": "uint256", "type": "uint256"}
            ],
            "outputs": [{"internalType": "bool", "type": "bool"}],
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ]`))
	transferABIFn := sampleABI.Functions()["transfer"]
	sampleABICallBytes, _ := transferABIFn.EncodeCallDataJSON([]byte(
		`{"recipient":"0x4a0d852ebb58fc88cb260bb270ae240f72edc45b","amount":"100000000000000000"}`,
	))
	fmt.Printf("ABI Call Bytes: %s\n", hex.EncodeToString(sampleABICallBytes))
	values, _ := transferABIFn.DecodeCallData(sampleABICallBytes)
	outputJSON, _ := NewSerializer().
		SetFormattingMode(FormatAsObjects).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SerializeJSON(values)
	fmt.Printf("Back to JSON:   %s\n", outputJSON)

	assert.JSONEq(t, `{"recipient":"0x4a0d852ebb58fc88cb260bb270ae240f72edc45b","amount":"100000000000000000"}`, string(outputJSON))

}

func TestABIGetTupleTypeTree(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI1), &abi)
	assert.NoError(t, err)

	assert.Equal(t, "foo((uint256,string[2],bytes))", abi[0].String())
	tc, err := abi[0].Inputs[0].TypeComponentTree()
	assert.NoError(t, err)

	solDef, childStructs, err := abi[0].SolidityDef()
	assert.NoError(t, err)
	assert.Equal(t, "function foo(AType memory a) external returns (uint256 e, string memory f) { }", solDef)
	assert.Equal(t, []string{"struct AType { uint256 b; string[2] c; bytes d; }"}, childStructs)

	assert.Equal(t, TupleComponent, tc.ComponentType())
	assert.Len(t, tc.TupleChildren(), 3)
	assert.Equal(t, "(uint256,string[2],bytes)", tc.String())
	assert.False(t, tc.ElementaryFixed()) // not fixed, as not elementary

	assert.Equal(t, ElementaryComponent, tc.TupleChildren()[0].ComponentType())
	assert.Equal(t, ElementaryTypeUint, tc.TupleChildren()[0].ElementaryType())
	assert.Equal(t, "256", tc.TupleChildren()[0].ElementarySuffix()) // alias resolved
	assert.True(t, tc.TupleChildren()[0].ElementaryFixed())

	assert.Equal(t, FixedArrayComponent, tc.TupleChildren()[1].ComponentType())
	assert.Equal(t, 2, tc.TupleChildren()[1].FixedArrayLen())
	assert.Equal(t, BaseTypeString, tc.TupleChildren()[1].ArrayChild().ElementaryType().BaseType())
	assert.Equal(t, ElementaryComponent, tc.TupleChildren()[1].ArrayChild().ComponentType())
	assert.Equal(t, ElementaryTypeString, tc.TupleChildren()[1].ArrayChild().ElementaryType())

}

func TestABIModifyReParse(t *testing.T) {

	abiString := `[
        {
          "name": "foo",
          "type": "function",
          "inputs": [
            {
                "name": "a",
                "type": "uint256"
            }
          ],
          "outputs": []
        }
      ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	assert.Equal(t, "foo(uint256)", abi[0].String())

	// Just changing doesn't help, as it's cached
	abi[0].Inputs[0].Type = "uint128"
	assert.Equal(t, "foo(uint256)", abi[0].String())

	// Re-parse sorts it
	abi.Validate()
	assert.Equal(t, "foo(uint128)", abi[0].String())
	assert.Equal(t, "0xc56cb6b0", abi[0].FunctionSelectorBytes().String())
	assert.Equal(t, ethtypes.HexBytes0xPrefix{0xc5, 0x6c, 0xb6, 0xb0}, abi[0].FunctionSelectorBytes())

}

func TestABIModifyBadInputs(t *testing.T) {

	abiString := `[
        {
          "name": "foo",
          "type": "function",
          "inputs": [
            {
                "name": "a",
                "type": "uint-1"
            }
          ],
          "outputs": []
        }
      ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	assert.Empty(t, abi[0].String())

	err = abi.Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Inputs[0].Validate()
	assert.Regexp(t, "FF22028", err)

	assert.Empty(t, abi[0].Inputs[0].String())
	assert.Equal(t, ethtypes.HexBytes0xPrefix{0x00, 0x00, 0x00, 0x00}, abi[0].FunctionSelectorBytes())

}

func TestABIModifyBadOutputs(t *testing.T) {

	abiString := `[
        {
          "name": "foo",
          "type": "function",
          "inputs": [],
          "outputs": [
              {
                "name": "a",
                "type": "uint-1"
              }
          ]
        }
      ]`
	var abi ABI
	err := json.Unmarshal([]byte(abiString), &abi)
	assert.NoError(t, err)

	err = abi.Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Validate()
	assert.Regexp(t, "FF22028", err)

	err = abi[0].Outputs[0].Validate()
	assert.Regexp(t, "FF22028", err)

	assert.Empty(t, abi[0].Outputs[0].String())

}

func TestParseJSONObjectModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `{
        "a": {
            "b": 12345,
            "c": ["string1", "string2"],
            "d": "0xfeedbeef"
        }
    }`
	var jv interface{}
	err := json.Unmarshal([]byte(values), &jv)
	assert.NoError(t, err)

	cv, err := inputs.ParseExternalData(jv)
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseJSONArrayModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `[
        [
            12345,
            ["string1", "string2"],
            "0xfeedbeef"
        ]
    ]`

	cv, err := inputs.ParseJSON([]byte(values))
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseJSONMixedModeOk(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `[
        {
            "b": 12345,
            "c": ["string1", "string2"],
            "d": "feedbeef"
        }
    ]`

	cv, err := inputs.ParseJSON([]byte(values))
	assert.NoError(t, err)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestABIParseCoerceGoTypes(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := map[interface{}]interface{}{
		"a": map[interface{}]interface{}{
			TestStringCustomType("b"): TestInt32CustomType(12345),
			&TestStringable{"c"}: []*TestStringable{
				{"string1"},
				{"string2"},
			},
			"d": TestByteArrayCustomType{0xfe, 0xed, 0xbe, 0xef},
		},
	}

	cv, err := inputs.ParseExternalData(values)
	assert.NoError(t, err)

	assert.Equal(t, "12345", cv.Children[0].Children[0].Value.(*big.Int).String())
	assert.Equal(t, "string1", cv.Children[0].Children[1].Children[0].Value)
	assert.Equal(t, "string2", cv.Children[0].Children[1].Children[1].Value)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, cv.Children[0].Children[2].Value)

}

func TestParseJSONArrayLotsOfTypes(t *testing.T) {

	inputs := testABI(t, sampleABI2)[0].Inputs

	values := `[
        "-12345",
        "0x12345",
        "0x4a0d852eBb58FC88Cb260Bb270AE240f72EdC45B",
        true,
        "-1.2345",
        1.2345,
        "0xfeedbeef",
        "00010203040506070809",
        "00",
        "test string"
    ]`

	cv, err := inputs.ParseJSON([]byte(values))
	assert.NoError(t, err)
	assert.NotNil(t, cv)

	assert.Equal(t, int64(-12345), cv.Children[0].Value.(*big.Int).Int64())
	assert.Equal(t, int64(0x12345), cv.Children[1].Value.(*big.Int).Int64())
	addrBytes, err := hex.DecodeString("4a0d852ebb58fc88cb260bb270ae240f72edc45b")
	assert.NoError(t, err)
	addrUint := new(big.Int).SetBytes(addrBytes)
	assert.Equal(t, addrUint.String(), cv.Children[2].Value.(*big.Int).String())
	assert.Equal(t, "1", cv.Children[3].Value.(*big.Int).String())
	assert.Equal(t, "-1.2345", cv.Children[4].Value.(*big.Float).String())
	assert.Equal(t, "1.2345", cv.Children[5].Value.(*big.Float).String())
	assert.Equal(t, "0xfeedbeef", ethtypes.HexBytes0xPrefix(cv.Children[6].Value.([]byte)).String())
	assert.Equal(t, "0x00010203040506070809", ethtypes.HexBytes0xPrefix(cv.Children[7].Value.([]byte)).String())
	assert.Equal(t, "0x00", ethtypes.HexBytes0xPrefix(cv.Children[8].Value.([]byte)).String())
	assert.Equal(t, "test string", cv.Children[9].Value)

	solDef, childStructs, err := testABI(t, sampleABI2)[0].SolidityDef()
	assert.NoError(t, err)
	assert.Equal(t, "function foo(uint8 a, int256 b, address c, bool d, fixed64x10 e, ufixed128x18 f, bytes10 g, bytes memory h, function i, string memory j) external { }", solDef)
	assert.Empty(t, childStructs)

}

func TestParseJSONBadData(t *testing.T) {
	inputs := testABI(t, sampleABI1)[0].Inputs
	_, err := inputs.ParseJSON([]byte(`{`))
	assert.Regexp(t, "unexpected EOF", err)

}

func TestParseJSONBadABI(t *testing.T) {
	inputs := testABI(t, `[
        {
          "name": "foo",
          "type": "function",
          "inputs": [
            {
                "name": "a",
                "type": "wrong"
            }
          ],
          "outputs": []
        }
      ]`)[0].Inputs
	_, err := inputs.ParseJSON([]byte(`{}`))
	assert.Regexp(t, "FF22025", err)

}

func TestEncodeABIDataCtxBadABI(t *testing.T) {
	f := testABI(t, `[
        {
          "name": "foo",
          "type": "function",
          "inputs": [
            {
                "name": "a",
                "type": "wrong"
            }
          ],
          "outputs": []
        }
      ]`)[0]
	_, err := f.EncodeCallData(nil)
	assert.Regexp(t, "FF22025", err)
}

func TestEncodeABIDataCtxBadInputs(t *testing.T) {
	f := testABI(t, sampleABI1)[0]
	_, err := f.EncodeCallData(nil)
	assert.Regexp(t, "FF22041", err)
}

func TestSignatureHashInvalid(t *testing.T) {
	e := &Entry{
		Inputs: ParameterArray{
			{
				Type: "foobar",
			},
		},
	}
	_, err := e.SignatureHash()
	assert.Regexp(t, "FF22025", err)

	_, _, err = e.SolidityDef()
	assert.Regexp(t, "FF22025", err)

	assert.Empty(t, e.SolString())

	assert.Equal(t, make(ethtypes.HexBytes0xPrefix, 32), e.SignatureHashBytes())

	e = &Entry{
		Outputs: ParameterArray{
			{
				Type: "foobar",
			},
		},
	}
	_, _, err = e.SolidityDef()
	assert.Regexp(t, "FF22025", err)

}

func TestDecodeEventIndexedOnly(t *testing.T) {
	e := &Entry{
		Anonymous: true,
		Type:      Event,
		Inputs: ParameterArray{
			{
				Name:    "from",
				Type:    "address",
				Indexed: true,
			},
			{
				Name:    "to",
				Type:    "address",
				Indexed: true,
			},
			{
				Name:    "tokenId",
				Type:    "uint256",
				Indexed: true,
			},
		},
	}
	v, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{
		ethtypes.MustNewHexBytes0xPrefix("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ethtypes.MustNewHexBytes0xPrefix("0x000000000000000000000000fb075bb99f2aa4c49955bf703509a227d7a12248"),
		ethtypes.MustNewHexBytes0xPrefix("0x000000000000000000000000000000000000000000000000000000000000091d"),
	}, ethtypes.HexBytes0xPrefix{})
	assert.NoError(t, err)

	j, err := v.JSON()
	assert.NoError(t, err)

	assert.JSONEq(t, `{
        "from": "0000000000000000000000000000000000000000",
        "to": "fb075bb99f2aa4c49955bf703509a227d7a12248",
        "tokenId": "2333"
    }`, string(j))
}

func TestDecodeEventMixed(t *testing.T) {
	e := &Entry{
		Type: Event,
		Name: "MyEvent",
		Inputs: ParameterArray{
			{
				Name:    "indexed1",
				Type:    "uint256",
				Indexed: true,
			},
			{
				Name:    "indexed2",
				Type:    "address",
				Indexed: true,
			},
			{
				Name: "unindexed1",
				Type: "uint256",
			},
			{
				Name: "unindexed2",
				Type: "bool",
			},
			{
				Name:    "indexed3",
				Type:    "string",
				Indexed: true,
			},
			{
				Name: "unindexed3",
				Type: "string",
			},
		},
	}
	v, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{
		ethtypes.MustNewHexBytes0xPrefix("0x27f22555fe6499d07163873ce3237d90091053cdeb2280c652466f4e1be378e5"),
		ethtypes.MustNewHexBytes0xPrefix("0x0000000000000000000000000000000000000000000000000000000000002b67"),
		ethtypes.MustNewHexBytes0xPrefix("0x0000000000000000000000003968ef051b422d3d1cdc182a88bba8dd922e6fa4"),
		ethtypes.MustNewHexBytes0xPrefix("0x592fa743889fc7f92ac2a37bb1f5ba1daf2a5c84741ca0e0061d243a2e6707ba"),
	}, ethtypes.MustNewHexBytes0xPrefix("0x00000000000000000000000000000000000000000000000000000000000056ce00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000b48656c6c6f20576f726c64000000000000000000000000000000000000000000"))
	assert.NoError(t, err)

	j, err := v.JSON()
	assert.NoError(t, err)

	assert.JSONEq(t, `{
        "indexed1": "11111",
        "indexed2": "3968ef051b422d3d1cdc182a88bba8dd922e6fa4",
        "unindexed1": "22222",
        "unindexed2": true,
        "indexed3": "592fa743889fc7f92ac2a37bb1f5ba1daf2a5c84741ca0e0061d243a2e6707ba",
        "unindexed3": "Hello World"
    }`, string(j))
}

func TestDecodeEventBadABI(t *testing.T) {
	e := &Entry{
		Inputs: ParameterArray{
			{
				Type: "wrong",
			},
		},
	}
	_, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{}, ethtypes.HexBytes0xPrefix{})
	assert.Regexp(t, "FF22025", err)
}

func TestDecodeEventBadSignature(t *testing.T) {
	e := &Entry{
		Name: "MyEvent",
		Type: Event,
		Inputs: ParameterArray{
			{
				Name:    "addr1",
				Type:    "address",
				Indexed: true,
			},
		},
	}
	_, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{
		ethtypes.MustNewHexBytes0xPrefix("0x0000000000000000000000000000000000000000000000000000000000000000"),
	}, ethtypes.HexBytes0xPrefix{})
	assert.Regexp(t, "FF22054", err)
}

func TestDecodeEventInsufficientTopics(t *testing.T) {
	e := &Entry{
		Name:      "MyEvent",
		Type:      Event,
		Anonymous: true,
		Inputs: ParameterArray{
			{
				Name:    "addr1",
				Type:    "address",
				Indexed: true,
			},
		},
	}
	_, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{}, ethtypes.HexBytes0xPrefix{})
	assert.Regexp(t, "FF22053", err)
}

func TestDecodeEventBadValue(t *testing.T) {
	e := &Entry{
		Name:      "MyEvent",
		Type:      Event,
		Anonymous: true,
		Inputs: ParameterArray{
			{
				Name:    "addr1",
				Type:    "address",
				Indexed: true,
			},
		},
	}
	_, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{
		ethtypes.MustNewHexBytes0xPrefix("0x"),
	}, ethtypes.HexBytes0xPrefix{})
	assert.Regexp(t, "FF22047", err)
}

func TestDecodeEventBadData(t *testing.T) {
	e := &Entry{
		Name:      "MyEvent",
		Type:      Event,
		Anonymous: true,
		Inputs: ParameterArray{
			{
				Name: "addr1",
				Type: "address",
			},
		},
	}
	_, err := e.DecodeEventData([]ethtypes.HexBytes0xPrefix{}, ethtypes.MustNewHexBytes0xPrefix("0x"))
	assert.Regexp(t, "FF22047", err)
}

func TestGetConstructor(t *testing.T) {
	a := testABI(t, sampleABI1)
	c := a.Constructor()
	assert.Nil(t, c)

	a = testABI(t, sampleABI3)
	c = a.Constructor()
	assert.Equal(t, Constructor, c.Type)
	assert.Equal(t, 1, len(c.Inputs))
}

func TestEncodeABIDataJSONHelper(t *testing.T) {

	a, _ := ParseABI([]byte(sampleABI4))
	_, err := a[0].Inputs.EncodeABIDataJSON([]byte(`[]`))
	assert.Regexp(t, "FF22037", err)

	b, err := a[0].Inputs.EncodeABIDataJSON([]byte(`["test"]`))
	assert.NoError(t, err)
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b))

}

func TestEncodeABIDataValuesHelper(t *testing.T) {

	a, _ := ParseABI([]byte(sampleABI4))
	_, err := a[0].Inputs.EncodeABIDataValues([]string{})
	assert.Regexp(t, "FF22037", err)

	b, err := a[0].Inputs.EncodeABIDataValues([]string{"test"})
	assert.NoError(t, err)
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b))

}

func TestEncodeCallDataJSONHelper(t *testing.T) {

	a, _ := ParseABI([]byte(sampleABI4))
	_, err := a[0].EncodeCallDataJSON([]byte(`[]`))
	assert.Regexp(t, "FF22037", err)

	b, err := a[0].EncodeCallDataJSON([]byte(`["test"]`))
	assert.NoError(t, err)
	assert.Equal(t, "113bc475000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b))

}

func TestEncodeCallDataValuesHelper(t *testing.T) {

	a, _ := ParseABI([]byte(sampleABI4))
	_, err := a[0].EncodeCallDataValues([]string{})
	assert.Regexp(t, "FF22037", err)

	b, err := a[0].EncodeCallDataValues([]string{"test"})
	assert.NoError(t, err)
	assert.Equal(t, "113bc475000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b))

}

func TestABIDocumented(t *testing.T) {
	ffapi.CheckObjectDocumented(&ABI{})
}

func TestComplexStructSolidityDef(t *testing.T) {

	var abi ABI
	err := json.Unmarshal([]byte(sampleABI5), &abi)
	assert.NoError(t, err)

	solDef, childStructs, err := abi.Functions()["invoice"].SolidityDef()
	assert.NoError(t, err)
	assert.Equal(t, "function invoice(Invoice memory _invoice) external payable { }", solDef)
	assert.Equal(t, []string{
		"struct Customer { address owner; bytes32 locator; }",
		"struct Widget { string description; uint256 price; string[] attributes; }",
		"struct Invoice { Customer customer; Widget[] widgets; }",
	}, childStructs)

	assert.Equal(t, "function invoice(Invoice memory _invoice) external payable { }; struct Customer { address owner; bytes32 locator; }; struct Widget { string description; uint256 price; string[] attributes; }; struct Invoice { Customer customer; Widget[] widgets; }",
		abi.Functions()["invoice"].SolString())

	solDef, childStructs, err = abi.Events()["Invoiced"].SolidityDef()
	assert.NoError(t, err)
	assert.Equal(t, "event Invoiced(Customer customer, Widget[] widgets, address indexed account)", solDef)
	assert.Equal(t, []string{
		"struct Customer { address owner; bytes32 locator; }",
		"struct Widget { string description; uint256 price; string[] attributes; }",
	}, childStructs)

}

func TestErrorString(t *testing.T) {

	customErrABI := ABI{
		{
			Type: Error,
			Name: "ExampleError",
			Inputs: ParameterArray{
				{
					Name: "param1",
					Type: "string",
				},
				{
					Name: "param2",
					Type: "uint256",
				},
			},
		},
	}

	revertReason, err := customErrABI[0].EncodeCallDataJSON([]byte(`{"param1":"test1","param2":12345}`))
	assert.NoError(t, err)

	errString, ok := customErrABI.ErrorString(revertReason)
	assert.True(t, ok)
	assert.Equal(t, `ExampleError("test1","12345")`, errString)

	e, cv, ok := customErrABI.ParseError(revertReason)
	assert.True(t, ok)
	assert.NotNil(t, e)
	assert.NotNil(t, cv)

	exampleDefaultError := ethtypes.MustNewHexBytes0xPrefix(`0x08c379a0` +
		`0000000000000000000000000000000000000000000000000000000000000020` +
		`000000000000000000000000000000000000000000000000000000000000001a` +
		`4e6f7420656e6f7567682045746865722070726f76696465642e000000000000`)
	errString, ok = customErrABI.ErrorString(exampleDefaultError)
	assert.True(t, ok)
	assert.Equal(t, `Error("Not enough Ether provided.")`, errString)

	mismatchError := ethtypes.MustNewHexBytes0xPrefix(`0x11223344`)
	_, ok = customErrABI.ErrorString(mismatchError)
	assert.False(t, ok)

}

func TestFilterType(t *testing.T) {
	abi := ABI{
		{Type: Function, Name: "transfer"},
		{Type: Error, Name: "MyError", Inputs: ParameterArray{{Type: "string"}}},
		{Type: Event, Name: "Transfer"},
		{Type: Error, Name: "AnotherError", Inputs: ParameterArray{{Type: "uint256"}}},
	}

	errors := abi.FilterType(Error)
	require.Len(t, errors, 2)
	assert.Equal(t, "MyError", errors[0].Name)
	assert.Equal(t, "AnotherError", errors[1].Name)

	functions := abi.FilterType(Function)
	require.Len(t, functions, 1)
	assert.Equal(t, "transfer", functions[0].Name)

	// No constructors present — result should be nil
	assert.Nil(t, abi.FilterType(Constructor))

	// Empty ABI
	assert.Nil(t, ABI{}.FilterType(Error))
}

func TestSelectorMap(t *testing.T) {
	errorEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	panicEntry := &Entry{Type: Error, Name: "Panic", Inputs: ParameterArray{{Name: "code", Type: "uint256"}}}
	abi := ABI{errorEntry, panicEntry}

	m := abi.SelectorMap()
	assert.Len(t, m, 2)

	var errorKey [4]byte
	copy(errorKey[:], errorEntry.FunctionSelectorBytes())
	assert.Equal(t, errorEntry, m[errorKey])

	var panicKey [4]byte
	copy(panicKey[:], panicEntry.FunctionSelectorBytes())
	assert.Equal(t, panicEntry, m[panicKey])
}

func TestSelectorMapFirstWins(t *testing.T) {
	// Two entries with identical signatures produce the same 4-byte selector;
	// the first one should win as a deterministic tiebreaker.
	first := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	second := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	m := ABI{first, second}.SelectorMap()

	var key [4]byte
	copy(key[:], first.FunctionSelectorBytes())
	assert.Equal(t, first, m[key], "first entry should win on selector collision")
}

func TestSelectorMapEmpty(t *testing.T) {
	assert.Empty(t, ABI{}.SelectorMap())
}

func TestSelectorMapAllEntryTypes(t *testing.T) {
	// SelectorMap works on any entry type, not just errors.
	// Callers can use FilterType first to restrict the scope.
	fnEntry := &Entry{Type: Function, Name: "transfer", Inputs: ParameterArray{{Type: "address"}, {Type: "uint256"}}}
	errEntry := &Entry{Type: Error, Name: "InsufficientBalance", Inputs: ParameterArray{{Type: "uint256"}}}
	abi := ABI{fnEntry, errEntry}

	m := abi.SelectorMap()
	assert.Len(t, m, 2)

	// Restricting to errors only via FilterType gives a smaller map.
	errOnly := abi.FilterType(Error).SelectorMap()
	assert.Len(t, errOnly, 1)
	var key [4]byte
	copy(key[:], errEntry.FunctionSelectorBytes())
	assert.Equal(t, errEntry, errOnly[key])
}

// buildErrorStringABI encodes msgBytes as the reason argument of Error(string)
// using the ABI pipeline, producing call data prefixed with the 4-byte selector.
// msgBytes may contain arbitrary binary content (e.g. embedded ABI-encoded errors).
func buildErrorStringABI(msgBytes []byte) []byte {
	entry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	data, err := entry.EncodeCallDataValues([]interface{}{string(msgBytes)})
	if err != nil {
		panic(fmt.Sprintf("buildErrorStringABI: %s", err))
	}
	return data
}

func TestErrorStringBinaryWrappedPlainError(t *testing.T) {
	revertData := ethtypes.MustNewHexBytes0xPrefix(
		"0x08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000000000000000000000000000000000000000001a" +
			"4e6f7420656e6f7567682045746865722070726f76696465642e000000000000")

	result, ok := ABI{}.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `Error("Not enough Ether provided.")`, result)
}

func TestErrorStringBinaryWrappedSingleNested(t *testing.T) {
	revertData := ethtypes.MustNewHexBytes0xPrefix(
		"0x08c379a00000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000000000000000000000000000000000000000006b" +
			"6f757465723a20" +
			"08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000013" +
			"696e6e6572206572726f72206d65737361676500000000000000000000000000" +
			"000000000000000000000000000000000000000000")

	result, ok := ABI{}.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `outer: Error("inner error message")`, result)
}

func TestErrorStringBinaryWrappedDoubleNested(t *testing.T) {
	revertData := ethtypes.MustNewHexBytes0xPrefix(
		"0x08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"00000000000000000000000000000000000000000000000000000000000000cc" +
			"6c6576656c313a20" +
			"08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000000000000000000000000000000000000000006c" +
			"6c6576656c323a20" +
			"08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000000000000000000000000000000000000000000d" +
			"64656570657374206572726f720000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	result, ok := ABI{}.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `level1: level2: Error("deepest error")`, result)
}

func TestErrorStringBinaryWrappedNestedCustomError(t *testing.T) {
	customABI := ABI{
		{Type: Error, Name: "MyCustomError", Inputs: ParameterArray{{Type: "bytes"}}},
	}
	customSelector := hex.EncodeToString(customABI[0].FunctionSelectorBytes())

	revertData := ethtypes.MustNewHexBytes0xPrefix(
		"0x08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000000000000000000000000000000000000000007c" +
			"5b3430345d303164202d206361756768742062797465733a" +
			customSelector +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000004" +
			"deadbeef00000000000000000000000000000000000000000000000000000000" +
			"00000000")

	// Without the custom ABI the inner error can't be decoded — the
	// outer Error(string) is formatted directly (binary content included)
	result, ok := ABI{}.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(result, `Error("[404]01d`))

	// With the custom ABI the inner error is decoded
	result, ok = customABI.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `[404]01d - caught bytes:MyCustomError("0xdeadbeef")`, result)
}

func TestErrorStringBinaryWrappedUnknownSelector(t *testing.T) {
	// Unknown top-level selector
	_, ok := ABI{}.ErrorString([]byte{0x11, 0x22, 0x33, 0x44}, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.False(t, ok)
}

func TestErrorStringBinaryWrappedMalformedInner(t *testing.T) {
	defaultErr := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	sel := defaultErr.FunctionSelectorBytes()

	badData := "prefix:" + string(sel) + "truncated"
	outerABI := buildErrorStringABI([]byte(badData))

	// Malformed inner data can't be decoded, so the outer Error(string)
	// is formatted directly with the raw string content
	result, ok := ABI{}.ErrorString(outerABI, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(result, "Error("))
}

func TestErrorStringBinaryWrappedDepthLimit(t *testing.T) {
	// Build a chain deeper than maxRevertErrorDepth (10)
	data := []byte("leaf")
	for i := 0; i < maxRevertErrorDepth+2; i++ {
		data = buildErrorStringABI(append([]byte("L:"), data...))
	}

	result, ok := ABI{}.ErrorString(data, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)

	// The chain should be capped — the leaf should not be fully unwrapped
	// through all levels, so the result won't contain a cleanly decoded "leaf"
	assert.NotEmpty(t, result)
}

func TestErrorStringBinaryWrappedCustomBeforeDefault(t *testing.T) {
	customABI := ABI{
		{Type: Error, Name: "EarlyErr", Inputs: ParameterArray{{Type: "uint256"}}},
	}
	customEncoded, err := customABI[0].EncodeCallDataValues([]interface{}{42})
	require.NoError(t, err)

	innerErrorABI := buildErrorStringABI([]byte("late-error"))
	// Custom selector appears before the Error(string) selector
	s := "head:" + string(customEncoded) + "middle:" + string(innerErrorABI)
	outerABI := buildErrorStringABI([]byte(s))

	result, ok := customABI.ErrorString(outerABI, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `head:EarlyErr("42")`, result)
}

// TestErrorStringAssemblyBubbleUp demonstrates that the assembly bubble-up
// pattern produces legible output from ErrorString — with or without SearchForWrappedBinaryErrors.
//
// The assembly bubble-up pattern:
//
//	(bool success, bytes memory result) = target.call(data);
//	if (!success) { assembly { revert(add(32, result), mload(result)) } }
//
// passes the raw error bytes through unchanged (no outer wrapper), so the
// error is decoded directly and is legible either way. SearchForWrappedBinaryErrors
// is not required but is harmless.
// TestAssemblyBubbleUpRealPayloads uses revert bytes captured from a live
// Solidity deployment on Kaleido (contract AssemblyRevertTest) where each
// bubbleXxx() function catches its inner revert via `catch (bytes memory data)`
// and re-reverts with:
//
//	assembly { revert(add(32, data), mload(data)) }
//
// These payloads confirm that the assembly pattern passes bytes through
// unchanged and that DecodeRevertError handles each without a new option.
func TestAssemblyBubbleUpRealPayloads(t *testing.T) {
	noParamsEntry := &Entry{Type: Error, Name: "NoParams", Inputs: ParameterArray{}}
	insufficientEntry := &Entry{Type: Error, Name: "InsufficientBalance", Inputs: ParameterArray{
		{Name: "available", Type: "uint256"},
		{Name: "required", Type: "uint256"},
	}}
	unauthorizedEntry := &Entry{Type: Error, Name: "Unauthorized", Inputs: ParameterArray{
		{Name: "caller", Type: "address"},
	}}
	withStringEntry := &Entry{Type: Error, Name: "WithString", Inputs: ParameterArray{
		{Name: "message", Type: "string"},
	}}
	customABI := ABI{noParamsEntry, insufficientEntry, unauthorizedEntry, withStringEntry}

	tests := []struct {
		name     string
		hex      string
		abi      ABI
		wantStr  string
		wantSig  string
	}{
		{
			name:    "Error(string)",
			hex:     "0x08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001a4e6f7420656e6f7567682045746865722070726f76696465642e000000000000",
			abi:     ABI{},
			wantStr: `Error("Not enough Ether provided.")`,
			wantSig: "Error(string)",
		},
		{
			name:    "Panic(uint256)",
			hex:     "0x4e487b710000000000000000000000000000000000000000000000000000000000000001",
			abi:     ABI{},
			wantStr: `Panic("1")`,
			wantSig: "Panic(uint256)",
		},
		{
			name:    "NoParams()",
			hex:     "0xa28f5fc7",
			abi:     customABI,
			wantStr: `NoParams()`,
			wantSig: "NoParams()",
		},
		{
			name:    "InsufficientBalance(uint256,uint256)",
			hex:     "0xcf479181000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000000c8",
			abi:     customABI,
			wantStr: `InsufficientBalance("100","200")`,
			wantSig: "InsufficientBalance(uint256,uint256)",
		},
		{
			name:    "WithString(string)",
			hex:     "0x64ed940e0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000962616420696e7075740000000000000000000000000000000000000000000000",
			abi:     customABI,
			wantStr: `WithString("bad input")`,
			wantSig: "WithString(string)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revertData := ethtypes.MustNewHexBytes0xPrefix(tt.hex)

			r := tt.abi.DecodeRevertError(revertData)
			require.NotNil(t, r)
			assert.Nil(t, r.GetInnerError(), "assembly bubble-up should produce no nesting")
			assert.Equal(t, tt.wantStr, r.String())

			sig, err := r.Signature()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSig, sig)

			// SearchForWrappedBinaryErrors must not change the result for non-nested payloads
			result, ok := tt.abi.ErrorString(revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
			assert.True(t, ok)
			assert.Equal(t, tt.wantStr, result)
		})
	}
}

func TestAssemblyBubbleUpUnauthorizedAddress(t *testing.T) {
	// Separate test: address formatting needs its own assertion since the
	// exact hex representation depends on the serializer's address format.
	unauthorizedEntry := &Entry{Type: Error, Name: "Unauthorized", Inputs: ParameterArray{
		{Name: "caller", Type: "address"},
	}}
	revertData := ethtypes.MustNewHexBytes0xPrefix(
		"0x8e4a23d6000000000000000000000000000000000000000000000000000000000000dead")

	r := ABI{unauthorizedEntry}.DecodeRevertError(revertData)
	require.NotNil(t, r)
	assert.Nil(t, r.GetInnerError())
	assert.Equal(t, "Unauthorized", r.ErrorEntry.Name)
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized(address)", sig)
	// Confirm the address value is present in the formatted string
	assert.Contains(t, r.String(), "dead")
}

func TestErrorStringAssemblyBubbleUp(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "InsufficientBalance", Inputs: ParameterArray{
		{Name: "available", Type: "uint256"},
		{Name: "required", Type: "uint256"},
	}}
	customABI := ABI{customEntry}

	// Real bytes from eth_call on AssemblyRevertTest.bubbleCustomWithUints()
	// deployed on Kaleido — identical to a direct InsufficientBalance(100,200) revert.
	rawErrorBytes := ethtypes.MustNewHexBytes0xPrefix(
		"0xcf479181" +
			"0000000000000000000000000000000000000000000000000000000000000064" +
			"00000000000000000000000000000000000000000000000000000000000000c8")

	// Without Unwrap: legible because the raw bytes ARE the error.
	result, ok := customABI.ErrorString(rawErrorBytes)
	assert.True(t, ok)
	assert.Equal(t, `InsufficientBalance("100","200")`, result)

	// With SearchForWrappedBinaryErrors: same output — no nesting, so result is unchanged.
	resultUnwrap, ok := customABI.ErrorString(rawErrorBytes, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, result, resultUnwrap)

	// Contrast: wrapping the same error inside Error(string) embeds binary in the string.
	// Without SearchForWrappedBinaryErrors the string contains raw binary and is not legible.
	wrappedBytes := buildErrorStringABI(append([]byte("outer: "), rawErrorBytes...))
	resultWrappedNoUnwrap, ok := customABI.ErrorString(wrappedBytes)
	assert.True(t, ok)
	assert.False(t, strings.HasPrefix(resultWrappedNoUnwrap, "InsufficientBalance"),
		"binary-wrapped error without SearchForWrappedBinaryErrors is not legible")

	// With SearchForWrappedBinaryErrors the inner error is decoded and the result is legible.
	resultWrappedUnwrap, ok := customABI.ErrorString(wrappedBytes, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
	assert.True(t, ok)
	assert.Equal(t, `outer: InsufficientBalance("100","200")`, resultWrappedUnwrap)
}

func TestSanitizeBinaryString(t *testing.T) {
	assert.Equal(t, "", SanitizeBinaryString(nil))
	assert.Equal(t, "", SanitizeBinaryString([]byte{}))
	assert.Equal(t, "hello world", SanitizeBinaryString([]byte("hello world")))
	assert.Equal(t, "0xdeadbeef", SanitizeBinaryString([]byte{0xde, 0xad, 0xbe, 0xef}))
	assert.Equal(t, "0x000000", SanitizeBinaryString([]byte{0x00, 0x00, 0x00}))
	assert.Equal(t, "0x736f6d65206572726f72000000", SanitizeBinaryString([]byte("some error\x00\x00\x00")))
	assert.Equal(t, "0x0168656c6c6f", SanitizeBinaryString([]byte{0x01, 'h', 'e', 'l', 'l', 'o'}))
}

func TestUnnamedInputOutput(t *testing.T) {

	sampleABI := ABI{
		{Type: Function, Name: "set", Inputs: ParameterArray{
			{Type: "uint256"},
			{Type: "string"},
		}},
	}

	cv, err := sampleABI[0].Inputs.ParseJSON([]byte(`{"0":12345,"1":"test"}`))
	require.NoError(t, err)
	res, err := NewSerializer().SetFormattingMode(FormatAsFlatArrays).SerializeJSON(cv)
	require.NoError(t, err)
	require.JSONEq(t, `["12345","test"]`, string(res))

	cv, err = sampleABI[0].Inputs.ParseJSON([]byte(`[12345,"test"]`))
	require.NoError(t, err)
	res, err = NewSerializer().SetFormattingMode(FormatAsObjects).SerializeJSON(cv)
	require.NoError(t, err)
	require.JSONEq(t, `{"0":"12345","1":"test"}`, string(res))

}
