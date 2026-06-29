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

package ffi2abi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/abi"
	"github.com/stretchr/testify/assert"
)

func TestFFIMethodToABI(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"details": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "newValue",
				Type:    "uint256",
				Indexed: false,
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)

	actualABIJSON, err := json.Marshal(abi)
	assert.NoError(t, err)
	expectedABIJSON, err := json.Marshal(expectedABIElement)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIJSON), string(actualABIJSON))
}

func TestFFIMethodToABIFloat(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "number",
					"details": {
						"type": "ufixed128x18"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "newValue",
				Type:    "ufixed128x18",
				Indexed: false,
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)

	actualABIJSON, err := json.Marshal(abi)
	assert.NoError(t, err)
	expectedABIJSON, err := json.Marshal(expectedABIElement)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIJSON), string(actualABIJSON))
}

func TestFFIMethodToABIObject(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "widget",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "object",
					"details": {
						"type": "tuple"
					},
					"properties": {
						"radius": {
							"type": "integer",
							"details": {
								"type": "uint256",
								"index": 0,
								"indexed": true
							}
						},
						"numbers": {
							"type": "array",
							"details": {
								"type": "uint256[]",
								"index": 1
							},
							"items": {
								"type": "integer",
								"details": {
									"type": "uint256"
								}
							}
						}
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "widget",
				Type:    "tuple",
				Indexed: false,
				Components: abi.ParameterArray{
					{
						Name:         "radius",
						Type:         "uint256",
						Indexed:      true,
						InternalType: "",
					},
					{
						Name:         "numbers",
						Type:         "uint256[]",
						Indexed:      false,
						InternalType: "",
					},
				},
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)
	assert.ObjectsAreEqual(expectedABIElement, abi)
}

func TestABIFFIConversionArrayOfObjects(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"components": [
						{
							"internalType": "string",
							"name": "name",
							"type": "string"
						},
						{
							"internalType": "uint256",
							"name": "weight",
							"type": "uint256"
						},
						{
							"internalType": "uint256",
							"name": "volume",
							"type": "uint256"
						},
						{
							"components": [
								{
									"internalType": "string",
									"name": "name",
									"type": "string"
								},
								{
									"internalType": "string",
									"name": "description",
									"type": "string"
								},
								{
									"internalType": "enum ComplexStorage.Alignment",
									"name": "alignment",
									"type": "uint8"
								}
							],
							"internalType": "struct ComplexStorage.BoxContent[]",
							"name": "contents",
							"type": "tuple[]"
						}
					],
					"internalType": "struct ComplexStorage.Box[]",
					"name": "newBox",
					"type": "tuple[]"
				}
			],
			"name": "set",
			"outputs": [],
			"stateMutability": "payable",
			"type": "function",
			"payable": true,
			"constant": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	abiFunction := abi.Functions()["set"]

	ffiMethod, err := convertABIFunctionToFFIMethod(context.Background(), abiFunction)
	assert.NoError(t, err)
	abiFunctionOut, err := ConvertFFIMethodToABI(context.Background(), ffiMethod)
	assert.NoError(t, err)

	expectedABIFunctionJSON, err := json.Marshal(abiFunction)
	assert.NoError(t, err)
	abiFunctionJSON, err := json.Marshal(abiFunctionOut)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedABIFunctionJSON), string(abiFunctionJSON))

}

func TestFFIMethodToABINestedArray(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "widget",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "array",
					"details": {
						"type": "string[][]",
						"internalType": "string[][]"
					},
					"items": {
						"type": "array",
						"items": {
							"type": "string"
						}
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:         "widget",
				Type:         "string[][]",
				InternalType: "string[][]",
				Indexed:      false,
			},
		},
		Outputs: abi.ParameterArray{},
	}

	abi, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.NoError(t, err)
	expectedABIJSON, err := json.Marshal(expectedABIElement)
	assert.NoError(t, err)
	abiJSON, err := json.Marshal(abi)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedABIJSON), string(abiJSON))
}

func TestFFIMethodToABIInvalidJSON(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name:   "newValue",
				Schema: fftypes.JSONAnyPtr(`{#!`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "invalid character", err)
}

func TestFFIMethodToABIBadSchema(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"detailz": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "FF22052", err)
}

func TestFFIMethodToABIBadReturn(t *testing.T) {
	method := &fftypes.FFIMethod{
		Name:   "set",
		Params: []*fftypes.FFIParam{},
		Returns: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"detailz": {
						"type": "uint256"
					}
				}`),
			},
		},
	}

	_, err := ConvertFFIMethodToABI(context.Background(), method)
	assert.Regexp(t, "FF22052", err)
}

func TestConvertABIToFFI(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "uint256",
					InternalType: "uint256",
				},
			},
			Outputs: abi.ParameterArray{},
		},
		{
			Name:   "get",
			Type:   "function",
			Inputs: abi.ParameterArray{},
			Outputs: abi.ParameterArray{
				{
					Name:         "value",
					Type:         "uint256",
					InternalType: "uint256",
				},
			},
		},
		{
			Name: "Updated",
			Type: "event",
			Inputs: abi.ParameterArray{{
				Name:         "value",
				Type:         "uint256",
				InternalType: "uint256",
			}},
			Outputs: abi.ParameterArray{},
		},
		{
			Name: "biggerThanTen",
			Type: "error",
			Inputs: abi.ParameterArray{{
				Name:         "value",
				Type:         "uint256",
				InternalType: "uint256",
			}},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"integer","details":{"type":"uint256","internalType":"uint256"}}`)

	expectedFFI := &fftypes.FFI{
		Name:        "SimpleStorage",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
			},
			{
				Name:   "get",
				Params: fftypes.FFIParams{},
				Returns: fftypes.FFIParams{
					{
						Name:   "value",
						Schema: schema,
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "Updated",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: schema,
						},
					},
				},
			},
		},
		Errors: []*fftypes.FFIError{
			{
				FFIErrorDefinition: fftypes.FFIErrorDefinition{
					Name: "biggerThanTen",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: schema,
						},
					},
				},
			},
		},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "SimpleStorage", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	assert.ObjectsAreEqual(expectedFFI, actualFFI)
}

func TestConvertABIToFFIWithObject(t *testing.T) {
	abi := &abi.ABI{
		&abi.Entry{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "tuple",
					InternalType: "struct WidgetFactory.Widget",
					Components: abi.ParameterArray{
						{
							Name:         "size",
							Type:         "uint256",
							InternalType: "uint256",
						},
						{
							Name:         "description",
							Type:         "string",
							InternalType: "string",
						},
					},
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	bigIntDesc := i18n.Expand(context.Background(), signermsgs.APIIntegerDescription)
	schema := fftypes.JSONAnyPtr(fmt.Sprintf(`{"type":"object","details":{"type":"tuple","internalType":"struct WidgetFactory.Widget"},"properties":{"description":{"type":"string","details":{"type":"string","internalType":"string","index":1}},"size":{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256","internalType":"uint256","index":0},"description":"%s"}}}`, bigIntDesc))

	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)

	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))

}

func TestConvertABIToFFIWithArray(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "string[]",
					InternalType: "string[]",
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"array","details":{"type":"string[]","internalType":"string[]"},"items":{"type":"string"}}`)

	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)

	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIWithNestedArray(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					Name:         "newValue",
					Type:         "uint256[][]",
					InternalType: "uint256[][]",
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	schema := fftypes.JSONAnyPtr(`{"type":"array","details":{"type":"uint256[][]","internalType":"uint256[][]"},"items":{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"integer"}],"description":"An integer. You are recommended to use a JSON string. A JSON number can be used for values up to the safe maximum."}}}`)
	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "newValue",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIWithNestedArrayOfObjects(t *testing.T) {
	abi := &abi.ABI{
		{
			Name: "set",
			Type: "function",
			Inputs: abi.ParameterArray{
				{
					InternalType: "struct WidgetFactory.Widget[][]",
					Name:         "gears",
					Type:         "tuple[][]",
					Components: abi.ParameterArray{
						{
							InternalType: "string",
							Name:         "description",
							Type:         "string",
						},
						{
							InternalType: "uint256",
							Name:         "size",
							Type:         "uint256",
						},
						{
							InternalType: "bool",
							Name:         "inUse",
							Type:         "bool",
						},
						{
							InternalType: "ufixed128x18",
							Name:         "value",
							Type:         "ufixed128x18",
						},
						{
							InternalType: "bytes",
							Name:         "data",
							Type:         "bytes",
						},
					},
				},
			},
			Outputs: abi.ParameterArray{},
		},
	}

	boolDesc := i18n.Expand(context.Background(), signermsgs.APIBoolDescription)
	bigIntDesc := i18n.Expand(context.Background(), signermsgs.APIIntegerDescription)
	floatDesc := i18n.Expand(context.Background(), signermsgs.APIFloatDescription)
	hexDesc := i18n.Expand(context.Background(), signermsgs.APIHexDescription)
	schema := fftypes.JSONAnyPtr(fmt.Sprintf(`{
		"type": "array",
		"details": {
			"type": "tuple[][]",
			"internalType": "struct WidgetFactory.Widget[][]"
		},
		"items": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"data": {
						"type": "string",
						"details": {
							"type": "bytes",
							"internalType": "bytes",
							"index": 4
						},
						"description": "%s"
					},
					"description": {
						"type": "string",
						"details": {
							"type": "string",
							"internalType": "string",
							"index": 0
						}
					},
					"inUse": {
						"oneOf": [
							{
								"type": "string"
							},
							{
								"type": "boolean"
							}
						],
						"details": {
							"type": "bool",
							"internalType": "bool",
							"index": 2
						},
						"description": "%s"
					},
					"size": {
						"oneOf": [
							{
								"type": "string"
							},
							{
								"type": "integer"
							}
						],
						"details": {
							"type": "uint256",
							"internalType": "uint256",
							"index": 1
						},
						"description": "%s"
					},
					"value": {
						"oneOf": [
							{
								"type": "string"
							},
							{
								"type": "number"
							}
						],
						"details": {
							"type": "ufixed128x18",
							"internalType": "ufixed128x18",
							"index": 3
						},
						"description": "%s"
					}
				}
			}
		}
	}`, hexDesc, boolDesc, bigIntDesc, floatDesc))
	expectedFFI := &fftypes.FFI{
		Name:        "WidgetTest",
		Version:     "v0.0.1",
		Namespace:   "default",
		Description: "desc",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "set",
				Params: fftypes.FFIParams{
					{
						Name:   "gears",
						Schema: schema,
					},
				},
				Returns: fftypes.FFIParams{},
				Details: map[string]interface{}{},
			},
		},
		Events: []*fftypes.FFIEvent{},
	}

	actualFFI, err := ConvertABIToFFI(context.Background(), "default", "WidgetTest", "v0.0.1", "desc", abi)
	assert.NoError(t, err)
	assert.NotNil(t, actualFFI)
	expectedFFIJSON, err := json.Marshal(expectedFFI)
	assert.NoError(t, err)
	actualFFIJSON, err := json.Marshal(actualFFI)
	assert.NoError(t, err)
	fmt.Printf("%s\n", actualFFIJSON)
	fmt.Printf("%s\n", expectedFFIJSON)
	assert.JSONEq(t, string(expectedFFIJSON), string(actualFFIJSON))
}

func TestConvertABIToFFIBadInputType(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"outputs": [],
			"stateMutability": "payable",
			"type": "function",
			"payable": true,
			"constant": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIToFFIBadOutputType(t *testing.T) {
	abiJSON := `[
		{
			"outputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"stateMutability": "viewable",
			"type": "function"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIToFFIBadEventType(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "set",
			"type": "event"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIToFFIBadErrorType(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "foobar"
				}
			],
			"name": "BadError",
			"type": "error"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	_, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.Regexp(t, "FF22025", err)
}

func TestConvertABIEventFFIEvent(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "string"
				}
			],
			"name": "set",
			"type": "event",
			"anonymous": true
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	ffi, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.NoError(t, err)

	actualABIEvent, err := ConvertFFIEventDefinitionToABI(context.Background(), &ffi.Events[0].FFIEventDefinition)
	assert.NoError(t, err)

	expectedABIEventJSON, err := json.Marshal(abi.Events()["set"])
	assert.NoError(t, err)
	actualABIEventJSON, err := json.Marshal(actualABIEvent)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIEventJSON), string(actualABIEventJSON))
}

func TestConvertABIErrorFFIError(t *testing.T) {
	abiJSON := `[
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "string"
				}
			],
			"name": "MyError",
			"type": "error"
		}
	]`

	var abi *abi.ABI
	json.Unmarshal([]byte(abiJSON), &abi)
	ffi, err := ConvertABIToFFI(context.Background(), "ns1", "name", "version", "description", abi)
	assert.NoError(t, err)

	actualABIError, err := ConvertFFIErrorDefinitionToABI(context.Background(), &ffi.Errors[0].FFIErrorDefinition)
	assert.NoError(t, err)

	expectedABIErrorJSON, err := json.Marshal(abi.Errors()["MyError"])
	assert.NoError(t, err)
	actualABIErrorJSON, err := json.Marshal(actualABIError)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedABIErrorJSON), string(actualABIErrorJSON))
}

func TestConvertFFIEventDefinitionToABIInvalidSchema(t *testing.T) {
	e := &fftypes.FFIEventDefinition{
		Params: fftypes.FFIParams{
			&fftypes.FFIParam{
				Name:   "badField",
				Schema: fftypes.JSONAnyPtr("foobar"),
			},
		},
	}
	_, err := ConvertFFIEventDefinitionToABI(context.Background(), e)
	assert.Regexp(t, "FF22052", err)
}

func TestConvertFFIErrorDefinitionToABIInvalidSchema(t *testing.T) {
	e := &fftypes.FFIErrorDefinition{
		Params: fftypes.FFIParams{
			&fftypes.FFIParam{
				Name:   "badField",
				Schema: fftypes.JSONAnyPtr("foobar"),
			},
		},
	}
	_, err := ConvertFFIErrorDefinitionToABI(context.Background(), e)
	assert.Regexp(t, "FF22052", err)
}

func TestProcessFieldInvalidSchema(t *testing.T) {
	s := &Schema{
		Type:    "object",
		Details: &paramDetails{},
		Properties: map[string]*Schema{
			"badProperty": {
				Type: "foo",
			},
		},
	}
	_, err := processField(context.Background(), "badType", s)
	assert.Regexp(t, "FF22052", err)
}

func TestABIMethodToSignature(t *testing.T) {
	abi := &abi.Entry{
		Name: "set",
		Type: "function",
		Inputs: abi.ParameterArray{
			{
				Name:    "widget",
				Type:    "tuple",
				Indexed: false,
				Components: abi.ParameterArray{
					{
						Name:         "radius",
						Type:         "uint256",
						Indexed:      true,
						InternalType: "",
					},
					{
						Name:         "numbers",
						Type:         "uint256[]",
						Indexed:      false,
						InternalType: "",
					},
				},
			},
		},
		Outputs: abi.ParameterArray{},
	}

	signature := ABIMethodToSignature(abi)
	assert.Equal(t, "set((uint256,uint256[]))", signature)
}

func TestConvertFFIParamsToABIParametersInvalidSchema(t *testing.T) {
	params := []*fftypes.FFIParam{
		{
			Name: "widget",
			Schema: fftypes.JSONAnyPtr(`{
				"type": "invalidSchema"
			}`),
		},
	}
	_, err := convertFFIParamsToABIParameters(context.Background(), params)
	assert.Regexp(t, "FF22052", err)
}

func TestConvertFFIParamsToABIParametersInvalidEthereumType(t *testing.T) {
	params := []*fftypes.FFIParam{
		{
			Name: "firstName",
			Schema: fftypes.JSONAnyPtr(`{
				"type": "string",
				"details": {
					"type": "foobar"
				}
			}`),
		},
	}
	_, err := convertFFIParamsToABIParameters(context.Background(), params)
	assert.Regexp(t, "FF22052", err)
}

func TestConvertFFIParamsToABIParametersTypeMismatch(t *testing.T) {
	params := []*fftypes.FFIParam{
		{
			Name: "firstName",
			Schema: fftypes.JSONAnyPtr(`{
				"type": "integer",
				"details": {
					"type": "bool"
				}
			}`),
		},
	}
	_, err := convertFFIParamsToABIParameters(context.Background(), params)
	assert.Regexp(t, "FF22052", err)
}

func TestInputTypeValidForTypeComponent(t *testing.T) {
	inputSchema := &Schema{
		Type: "boolean",
	}
	param := abi.Parameter{
		Type: "bool",
	}
	tc, _ := param.TypeComponentTree()
	assert.NoError(t, inputTypeValidForTypeComponent(context.Background(), inputSchema, tc))
}

func TestInputTypeValidForTypeComponentOneOf(t *testing.T) {
	inputSchema := &Schema{
		OneOf: []SchemaType{
			{
				Type: "integer",
			},
			{
				Type: "string",
			},
		},
	}
	param := abi.Parameter{
		Type: "uint256",
	}
	tc, _ := param.TypeComponentTree()
	assert.NoError(t, inputTypeValidForTypeComponent(context.Background(), inputSchema, tc))
}

func TestInputTypeValidForTypeComponentInvalid(t *testing.T) {
	inputSchema := &Schema{
		Type: "foobar",
	}
	param := abi.Parameter{
		Type: "bool",
	}
	tc, _ := param.TypeComponentTree()
	assert.Regexp(t, "FF22055", inputTypeValidForTypeComponent(context.Background(), inputSchema, tc))
}
