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
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

type TestStringCustomType string
type TestStringPtrCustomType *string
type TestInt64CustomType int64
type TestInt64PtrCustomType *int64
type TestInt32CustomType int32
type TestInt32PtrCustomType *int32
type TestFloat64CustomType float64
type TestFloat64PtrCustomType *float64
type TestFloat32CustomType float32
type TestFloat32PtrCustomType *float32
type TestByteArrayCustomType []byte
type TestByteArrayPtrCustomType *[]byte

type TestStringable struct {
	s string
}

func (ts *TestStringable) String() string {
	return ts.s
}

const sampleABI2NestedTupleArray = `[
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
					"type": "tuple[][]",
					"components": [
						{
							"name": "c",
							"type": "string"
						}		
					]
				}
			]
		}
	  ],
	  "outputs": []
	}
  ]`

func TestGetIntegerFromInterface(t *testing.T) {

	ctx := context.Background()

	i, err := getIntegerFromInterface(ctx, "ut", "-12345")
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "4276993775", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", "-0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "-4276993775", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", int(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", int64(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", int32(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", int16(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", int8(-32))
	assert.NoError(t, err)
	assert.Equal(t, "-32", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", uint(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", uint64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", uint32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", uint16(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", uint8(32))
	assert.NoError(t, err)
	assert.Equal(t, "32", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", float64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", float32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	iB := big.NewInt(-12345)
	i, err = getIntegerFromInterface(ctx, "ut", iB)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	iF := big.NewFloat(-12345)
	i, err = getIntegerFromInterface(ctx, "ut", iF)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	str := "-12345"
	i, err = getIntegerFromInterface(ctx, "ut", &str)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	strPtr := &str
	i, err = getIntegerFromInterface(ctx, "ut", &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", &TestStringable{s: "-12345"})
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var is TestStringCustomType = "-12345"
	i, err = getIntegerFromInterface(ctx, "ut", &is)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var ips TestStringPtrCustomType = strPtr
	i, err = getIntegerFromInterface(ctx, "ut", ips)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI64 TestInt64CustomType = -12345
	i, err = getIntegerFromInterface(ctx, "ut", &iI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i64 := int64(-12345)
	var iPI64 TestInt64PtrCustomType = &i64
	i, err = getIntegerFromInterface(ctx, "ut", iPI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI32 TestInt32CustomType = -12345
	i, err = getIntegerFromInterface(ctx, "ut", &iI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var scientificNumber fftypes.JSONAny
	err = json.Unmarshal([]byte("1.0000000000000000000000001e+25"), &scientificNumber)
	assert.NoError(t, err)

	i, err = getIntegerFromInterface(ctx, "ut", scientificNumber)
	assert.NoError(t, err)
	assert.Equal(t, "10000000000000000000000001", i.String())

	var hugeNumber fftypes.JSONAny
	err = json.Unmarshal([]byte("10000000000000000000000000000001"), &hugeNumber)
	assert.NoError(t, err)

	i, err = getIntegerFromInterface(ctx, "ut", hugeNumber)
	assert.NoError(t, err)
	assert.Equal(t, "10000000000000000000000000000001", i.String())

	var jsonNumber json.Number
	err = json.Unmarshal([]byte("20000000000000000000000000000002"), &jsonNumber)
	assert.NoError(t, err)

	i, err = getIntegerFromInterface(ctx, "ut", jsonNumber)
	assert.NoError(t, err)
	assert.Equal(t, "20000000000000000000000000000002", i.String())

	i32 := int32(-12345)
	var iPI32 TestInt32PtrCustomType = &i32
	i, err = getIntegerFromInterface(ctx, "ut", iPI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getIntegerFromInterface(ctx, "ut", "wrong")
	assert.Regexp(t, "FF22030", err)
	assert.Nil(t, i)

	i, err = getIntegerFromInterface(ctx, "ut", []string{"wrong"})
	assert.Regexp(t, "FF22030", err)
	assert.Nil(t, i)

	i, err = getIntegerFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22030", err)
	assert.Nil(t, i)

	i, err = getIntegerFromInterface(ctx, "ut", json.Number("wrong"))
	assert.Regexp(t, "FF22030", err)
	assert.Nil(t, i)

}

func TestGetFloatFromInterface(t *testing.T) {

	ctx := context.Background()

	i, err := getFloatFromInterface(ctx, "ut", "-1.2345")
	assert.NoError(t, err)
	assert.Equal(t, "-1.2345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "4276993775", i.String())

	i, err = getFloatFromInterface(ctx, "ut", "-0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, "-4276993775", i.String())

	i, err = getFloatFromInterface(ctx, "ut", int(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", int64(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", int32(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", int16(-12345))
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", int8(-32))
	assert.NoError(t, err)
	assert.Equal(t, "-32", i.String())

	i, err = getFloatFromInterface(ctx, "ut", uint(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", uint64(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", uint32(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", uint16(12345))
	assert.NoError(t, err)
	assert.Equal(t, "12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", uint8(32))
	assert.NoError(t, err)
	assert.Equal(t, "32", i.String())

	i, err = getFloatFromInterface(ctx, "ut", float64(-1.2345))
	assert.NoError(t, err)
	assert.Equal(t, "-1.2345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", float32(1.2345))
	assert.NoError(t, err)
	assert.Equal(t, "1.234500051", i.String())

	iB := big.NewInt(-12345)
	i, err = getFloatFromInterface(ctx, "ut", iB)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	iF := big.NewFloat(-12345)
	i, err = getFloatFromInterface(ctx, "ut", iF)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	str := "-12345"
	i, err = getFloatFromInterface(ctx, "ut", &str)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	strPtr := &str
	i, err = getFloatFromInterface(ctx, "ut", &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var is TestStringCustomType = "-12345"
	i, err = getFloatFromInterface(ctx, "ut", &is)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var ips TestStringPtrCustomType = strPtr
	i, err = getFloatFromInterface(ctx, "ut", ips)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI64 TestInt64CustomType = -12345
	i, err = getFloatFromInterface(ctx, "ut", &iI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i64 := int64(-12345)
	var iPI64 TestInt64PtrCustomType = &i64
	i, err = getFloatFromInterface(ctx, "ut", iPI64)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	var iI32 TestInt32CustomType = -12345
	i, err = getFloatFromInterface(ctx, "ut", &iI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i32 := int32(-12345)
	var iPI32 TestInt32PtrCustomType = &i32
	i, err = getFloatFromInterface(ctx, "ut", iPI32)
	assert.NoError(t, err)
	assert.Equal(t, "-12345", i.String())

	i, err = getFloatFromInterface(ctx, "ut", "wrong")
	assert.Regexp(t, "FF22031", err)
	assert.Nil(t, i)

	i, err = getFloatFromInterface(ctx, "ut", []string{"wrong"})
	assert.Regexp(t, "FF22031", err)
	assert.Nil(t, i)

	i, err = getFloatFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22031", err)
	assert.Nil(t, i)
}

func TestGetBoolFromInterface(t *testing.T) {

	ctx := context.Background()

	v, err := getBoolAsUnsignedIntegerFromInterface(ctx, "ut", "true")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v.Int64())

	v, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", "false")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), v.Int64())

	v, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v.Int64())

	v, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), v.Int64())

	vTrue := true
	v, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", &vTrue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v.Int64())

	var is TestStringCustomType = "true"
	v, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", &is)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v.Int64())

	_, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", int(-12345))
	assert.Regexp(t, "FF22033", err)

	_, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", []bool{true})
	assert.Regexp(t, "FF22033", err)

	_, err = getBoolAsUnsignedIntegerFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22033", err)

}

func TestGetStringFromInterface(t *testing.T) {

	ctx := context.Background()

	s, err := getStringFromInterface(ctx, "ut", "test data")
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	s, err = getStringFromInterface(ctx, "ut", "test data")
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	s, err = getStringFromInterface(ctx, "ut", []byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	str := "test data"
	s, err = getStringFromInterface(ctx, "ut", &str)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	strPtr := &str
	s, err = getStringFromInterface(ctx, "ut", &strPtr)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	var is TestStringCustomType = "test data"
	s, err = getStringFromInterface(ctx, "ut", &is)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	var ips TestStringPtrCustomType = strPtr
	s, err = getStringFromInterface(ctx, "ut", ips)
	assert.NoError(t, err)
	assert.Equal(t, "test data", s)

	_, err = getStringFromInterface(ctx, "ut", int(-12345))
	assert.Regexp(t, "FF22032", err)

	_, err = getStringFromInterface(ctx, "ut", []string{"wrong"})
	assert.Regexp(t, "FF22032", err)

	_, err = getStringFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22032", err)

}

func TestGetBytesFromInterface(t *testing.T) {

	ctx := context.Background()

	s, err := getBytesFromInterface(ctx, "ut", "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, "ut", "feedbeef")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, "ut", []byte{0xfe, 0xed, 0xbe, 0xef})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	s, err = getBytesFromInterface(ctx, "ut", []byte{0xfe, 0xed, 0xbe, 0xef})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	bv := []byte{0xfe, 0xed, 0xbe, 0xef}
	s, err = getBytesFromInterface(ctx, "ut", &bv)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var bt TestByteArrayCustomType = []byte{0xfe, 0xed, 0xbe, 0xef}
	s, err = getBytesFromInterface(ctx, "ut", bt)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var bpt TestByteArrayPtrCustomType = &bv
	s, err = getBytesFromInterface(ctx, "ut", bpt)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	var is TestStringCustomType = "0xfeedbeef"
	s, err = getBytesFromInterface(ctx, "ut", &is)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xfe, 0xed, 0xbe, 0xef}, s)

	_, err = getBytesFromInterface(ctx, "ut", int(-12345))
	assert.Regexp(t, "FF22034", err)

	_, err = getBytesFromInterface(ctx, "ut", []string{"wrong"})
	assert.Regexp(t, "FF22034", err)

	_, err = getBytesFromInterface(ctx, "ut", "wrong")
	assert.Regexp(t, "FF22034", err)

	_, err = getBytesFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22034", err)

}

func TestGetUintBytesFromInterface(t *testing.T) {

	ctx := context.Background()

	i, err := getUintBytesFromInterface(ctx, "ut", "0xfeedbeef")
	assert.NoError(t, err)
	assert.Equal(t, int64(0xfeedbeef), i.Int64())

	_, err = getUintBytesFromInterface(ctx, "ut", nil)
	assert.Regexp(t, "FF22034", err)

}

func TestABIParseMissingRoot(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `{}`

	_, err := inputs.ParseJSON([]byte(values))
	assert.Regexp(t, "FF22040", err)

}

func TestABIParseCoerceGoMapFail(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := map[interface{}]interface{}{
		false: true,
	}

	_, err := inputs.ParseExternalData(values)
	assert.Regexp(t, "FF22032", err)

}

func TestABIParseBadElementalType(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := map[string]interface{}{
		"a": []interface{}{
			nil, nil, nil,
		},
	}

	_, err := inputs.ParseExternalData(values)
	assert.Regexp(t, "FF22030", err)

}

func TestWalkInputBadType(t *testing.T) {

	cv, err := walkInput(context.Background(), "", nil, &typeComponent{
		cType: ComponentType(99),
	})
	assert.Regexp(t, "FF22041", err)
	assert.Nil(t, cv)

}

func TestWalkArrayInputBadType(t *testing.T) {

	cv, err := walkArrayInput(context.Background(), "", nil, &typeComponent{
		cType: FixedArrayComponent,
	})
	assert.Regexp(t, "FF22035", err)
	assert.Nil(t, cv)

}

func TestWrongLength(t *testing.T) {

	inputs := testABI(t, sampleABI1)[0].Inputs

	values := `[
		{
			"b": 12345,
			"c": ["string1"],
			"d": "feedbeef"
		}
	]`

	_, err := inputs.ParseJSON([]byte(values))
	assert.Regexp(t, "FF22036", err)

}

func TestNestedTuplesOk(t *testing.T) {

	inputs := testABI(t, sampleABI2NestedTupleArray)[0].Inputs

	values := `{
		"a": {
			"b": [
				[
					{
						"c": "test1"
					}
				]
			]
		}
	}`

	cv, err := inputs.ParseJSON([]byte(values))
	assert.NoError(t, err)

	assert.Equal(t, "test1", cv.Children[0].Children[0].Children[0].Children[0].Children[0].Value)

}

func TestNestedTuplesBadLeaf(t *testing.T) {

	inputs := testABI(t, sampleABI2NestedTupleArray)[0].Inputs

	values := `{
		"a": {
			"b": [
				[
					{
						"c": false
					}
				]
			]
		}
	}`

	_, err := inputs.ParseJSON([]byte(values))
	assert.Regexp(t, "FF22032", err)

}

func TestNestedTuplesMissingTupleArrayEntry(t *testing.T) {

	inputs := testABI(t, sampleABI2NestedTupleArray)[0].Inputs

	values := `{
		"a": {
			"b": [
				[
					[]
				]
			]
		}
	}`

	_, err := inputs.ParseJSON([]byte(values))
	assert.Regexp(t, "FF22037", err)

}

func TestTuplesWrongType(t *testing.T) {

	inputs := testABI(t, sampleABI2NestedTupleArray)[0].Inputs

	values := `{
		"a": false
	}`

	_, err := inputs.ParseJSON([]byte(values))
	assert.Regexp(t, "FF22038", err)

}

func TestTupleEncodeIndividualFixedParam(t *testing.T) {
	const sample = `[
		{
		"name": "foo",
		"type": "function",
		"inputs": [
			{
				"name": "a",
				"type": "int"
			}
		],
		"outputs": []
		}
	]`

	inputs := testABI(t, sample)[0].Inputs

	// Fine if you use the array syntax
	values := `{ "a": 12345 }`
	cv, err := inputs.ParseJSON([]byte(values))
	assert.NoError(t, err)
	assert.Equal(t, TupleComponent, cv.Component.ComponentType())
	assert.Len(t, cv.Children, 1)
	_, _, err = cv.ElementaryABIData()
	assert.Regexp(t, "FF22069", err)

	intComp := cv.Children[0]
	assert.Equal(t, ElementaryComponent, intComp.Component.ComponentType())
	b, dynamic, err := intComp.ElementaryABIData()
	assert.NoError(t, err)
	assert.False(t, dynamic)
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000003039", ethtypes.HexBytes0xPrefix(b).String())

}
