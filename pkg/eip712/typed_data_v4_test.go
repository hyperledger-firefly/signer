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

	"github.com/hyperledger-firefly/common/pkg/ffapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const PersonType = `[{"name": "name","type": "string"},{"name": "wallet","type": "address"}]`

const MailType = `[{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]`

func TestMessage_ExampleFromEIP712Spec(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
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
			],
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
		"domain": {
			"name": "Ether Mail",
			"version": "V4",
			"chainId": 1,
			"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
		},
		"message": {
			"from": {
				"name": "Cow",
				"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
			},
			"to": {
				"name": "Bob",
				"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
			},
			"contents": "Hello, Bob!"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xde26f53b35dd5ffdc13f8297e5cc7bbcb1a04bf33803bd2bf4a45eb251360cb8", ed.String())

	hs, err := HashStruct(ctx, p.PrimaryType, p.Message, p.Types)
	assert.NoError(t, err)
	assert.Equal(t, "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e", hs.String())
}

func TestMessage_EmptyMessage(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {},
		"primaryType": "EIP712Domain"
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x8d4a3f4082945b7879e2b55f181c31a77c8c0a464b70669458abbaaf99de4c38", ed.String())
}

func TestMessage_EmptyDomain(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": {
				"name": "Cow",
				"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
			},
			"to": {
				"name": "Bob",
				"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
			},
			"contents": "Hello, Bob!"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x25c3d40a39e639a4d0b6e4d2ace5e1281e039c88494d97d8d08f99a6ea75d775", ed.String())
}

func TestMessage_NilReference(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": "Hello, Bob!"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x326faa52849c078e0e04abe863b29fc28d9d2885d2c4b515fcfb7ba1fac30534", ed.String())
}

func TestMessage_BytesString(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "bytes"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": "0x48656C6C6F2C20426F6221"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x3e4282c3bc7b7d6df14ef1c1c90f7bef0516134f4ca08d56eb38b061e5632a6b", ed.String())
}

func TestMessage_Bytes11(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "bytes11"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": "0x48656C6C6F2C20426F6221"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xb13b01acae69dbd0fef3568f1b060a692247aa207609d008f344c8cd7f664220", ed.String())
}

func TestMessage_StringArray(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string[]"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": ["Hello,", "Bob!"]
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xd0ac411802ea14e4e64eeed229227be1bf2909f0a30bda74c79447dfbf2f5431", ed.String())
}

func TestMessage_StringArrayArray(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string[][]"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": [
				["Hello,", "Bob!"],
				["How,", "do"]
			]
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x88454de00616bf6b3697b55281de2e8fb542b3997c397ad70e0c8f8f72d164f0", ed.String())
}

func TestMessage_FixedStringArray(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"Person": [{"name": "name","type": "string"},{"name": "wallet","type": "address"}],
			"Mail": [{"name": "from","type": "Person"},{"name": "to","type": "Person"},{"name": "contents","type": "string[2]"}]
		},
		"primaryType": "Mail",
		"message": {
			"from": null,
			"to": null,
			"contents": ["Hello,", "Bob!"]
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0xb1bf2c8635345d6fc3e86e493180f96043548ac761683a4d069725f08a6ea2bf", ed.String())
}

func TestMessage_StructArray(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"AllTheTypes": [
				{
					"name": "i32",
					"type": "int32"
				},
				{
					"name": "i256",
					"type": "int256"
				},
				{
					"name": "ui32",
					"type": "uint32"
				},
				{
					"name": "ui256",
					"type": "uint256"
				},
				{
					"name": "t",
					"type": "bool"
				},
				{
					"name": "b16",
					"type": "bytes16"
				},
				{
					"name": "b32",
					"type": "bytes32"
				},
				{
					"name": "b",
					"type": "bytes"
				},
				{
					"name": "s",
					"type": "string"
				}
			]
		},
		"primaryType": "AllTheTypes",
		"message": {
			"i32": -12345,
			"i256": "-12345",
			"ui32": "0x3039",
			"ui256": "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"t": true,
			"b16": "0x000102030405060708090a0b0c0f0e0f",
			"b32": "0x000102030405060708090a0b0c0f0e0f000102030405060708090a0b0c0f0e0f",
			"b": "0xfeedbeef",
			"s": "Hello World!"
		}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	ed, err := EncodeTypedDataV4(ctx, &p)
	assert.NoError(t, err)
	assert.Equal(t, "0x651579f58b3a8c79ba668e0f5d83e1c9f6e2715586dc11c62696ec376b595a00", ed.String())
}

func TestMissingPrimaryTypeField(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22080", err)
}

func TestInvalidDomain(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"EIP712Domain": [{"name":"name","type":"string"}]
		},
		"primaryType": "EIP712Domain",
		"domain": {"name": ["not","a","string"]}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22032", err)
}

func TestInvalidPrimaryType(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"name","type":"string"}]
		},
		"primaryType": "MyType",
		"message": {"name": ["not","a","string"]}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22032", err)
}

func TestMissingPrimaryType(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType2": [{"name":"name","type":"string"}]
		},
		"primaryType": "MyType1",
		"message": {"name": "Bob"}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22073", err)
}

func TestSecondaryTypeNotMap(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType1": [{"name":"t2","type":"MyType2"}],
			"MyType2": [{"name":"name","type":"string"}]
		},
		"primaryType": "MyType1",
		"message": {"t2": "this is not a map"}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22076", err)
}

func TestTypeInvalid(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"number","type":"int2560"}]
		},
		"primaryType": "MyType",
		"message": {"number": 12345}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22028", err)
}

func TestIntValueInvalid(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"number","type":"int256"}]
		},
		"primaryType": "MyType",
		"message": {"number": "!!! not a number"}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22030", err)
}

func TestBytesValueInvalid(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"bite","type":"bytes"}]
		},
		"primaryType": "MyType",
		"message": {"bite": -99}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22034", err)
}

func TestFixedNotSupported(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"unmovable","type":"fixed256x18"}]
		},
		"primaryType": "MyType",
		"message": {"unmovable": "1.23"}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22072", err)
}

func TestTupleNotSupported(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"abi_struct","type":"tuple"}]
		},
		"primaryType": "MyType",
		"message": {"abi_struct": {}}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22069", err)
}

func TestAddressInvalid(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"addr","type":"address"}]
		},
		"primaryType": "MyType",
		"message": {"addr": "not here"}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22034", err)
}

func TestArrayBadSuffix(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"wonky","type":"int256]"}]
		},
		"primaryType": "MyType",
		"message": {"wonky": 1}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22077", err)
}

func TestArrayFieldNonArrayValue(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"many","type":"int256[]"}]
		},
		"primaryType": "MyType",
		"message": {"many": 1}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22078", err)
}

func TestArrayFieldBadSize(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"many","type":"int256[n]"}]
		},
		"primaryType": "MyType",
		"message": {"many": [1]}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22077", err)
}

func TestArrayFieldSizeMismatch(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"many","type":"int256[2]"}]
		},
		"primaryType": "MyType",
		"message": {"many": [1]}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22079", err)
}

func TestArrayValueInvalid(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	var p TypedData
	err := json.Unmarshal([]byte(`{
		"types": {
			"MyType": [{"name":"many","type":"int256[]"}]
		},
		"primaryType": "MyType",
		"message": {"many": ["not a number"]}
	}`), &p)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = EncodeTypedDataV4(ctx, &p)
	assert.Regexp(t, "FF22030", err)
}

func TestTypedDataDocumented(t *testing.T) {
	ffapi.CheckObjectDocumented(&TypedData{})
}
