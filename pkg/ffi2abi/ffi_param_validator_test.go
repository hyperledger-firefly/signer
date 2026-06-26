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
	"encoding/json"
	"strings"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
)

func NewTestSchema(input string) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	f := fftypes.BaseFFIParamValidator{}
	c.RegisterExtension(f.GetExtensionName(), f.GetMetaSchema(), f)
	v := &ParamValidator{}
	c.RegisterExtension(v.GetExtensionName(), v.GetMetaSchema(), v)
	err := c.AddResource("schema.json", strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	return c.Compile("schema.json")
}

func jsonDecode(input string) interface{} {
	var output interface{}
	json.Unmarshal([]byte(input), &output)
	return output
}

func TestSchemaValid(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "integer",
	"details": {
		"type": "uint256"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaValidBytes(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "bytes"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaValidBytes32(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "bytes32"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaTypeInvalid(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "foobar",
	"details": {
		"type": "uint256"
	}
}`)
	assert.Regexp(t, "'/type' does not validate", err)
}

func TestSchemaTypeInvalidFFIType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "null",
	"details": {
		"type": "uint256"
	}
}`)
	assert.Regexp(t, "compilation failed", err)
}

func TestSchemaTypeMissing(t *testing.T) {
	_, err := NewTestSchema(`{}`)
	assert.Regexp(t, "compilation failed", err)
}

func TestSchemaDetailsTypeMissing(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"indexed": true
	}
}`)
	assert.Regexp(t, "compilation failed", err)
}

func TestSchemaDetailsIndexedWrongType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "string",
		"indexed": "string"
	}
}`)
	assert.Regexp(t, "compilation failed", err)
}

func TestInputString(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "string"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(`"banana"`)
	assert.NoError(t, err)
}

func TestInputInteger(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "integer",
	"details": {
		"type": "uint256"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(1)
	assert.NoError(t, err)
}

func TestInputBoolean(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "boolean",
	"details": {
		"type": "bool"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(true)
	assert.NoError(t, err)
}

func TestInputStruct(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "object",
	"details": {
		"type": "tuple"
	},
	"properties": {
		"x": {
			"type": "integer",
			"details": {
				"type": "uint8",
				"index": 0
			}
		},
		"y": {
			"type": "integer",
			"details": {
				"type": "uint8",
				"index": 1
			}
		},
		"z": {
			"type": "integer",
			"details": {
				"type": "uint8",
				"index": 2
			}
		}
	},
	"required": ["x", "y", "z"]
}`)

	input := `{
	"x": 123,
	"y": 456,
	"z": 789
}`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.NoError(t, err)
}

func TestInputArray(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "array",
	"details": {
		"type": "uint8[]"
	},
	"items": {
		"type": "integer"
	}
}`)

	input := `[123,456,789]`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.NoError(t, err)
}

func TestValidOneOf(t *testing.T) {
	_, err := NewTestSchema(`
	{
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
			"internalType": "uint256"
		}
	}`)
	assert.NoError(t, err)
}

func TestInputInvalidOneOf(t *testing.T) {
	_, err := NewTestSchema(`
	{
		"oneOf": "banana",
		"details": {
			"type": "uint256",
			"internalType": "uint256"
		}
	}`)
	assert.Regexp(t, "'/oneOf' does not validate", err)
}

func TestInputInvalidOneOfType(t *testing.T) {
	_, err := NewTestSchema(`
	{
		"oneOf": [
			{
				"type": "banana"
			}
		],
		"details": {
			"type": "uint256",
			"internalType": "uint256"
		}
	}`)
	assert.Regexp(t, "'/oneOf/0/type' does not validate", err)
}

func TestInputNoAdditionalProperties(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "object",
	"details": {
		"type": "tuple"
	},
	"properties": {
		"foo": {
			"type": "string",
			"details": {
				"type": "string",
				"index": 0
			}
		}
	},
	"additionalProperties": false
}`)

	input := `{
	"foo": "foo",
	"bar": "bar"
}`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.Regexp(t, "additionalProperties 'bar' not allowed", err)
}

func TestInputFixedArraySizeType(t *testing.T) {
	_, err := NewTestSchema(`
	{
		"type": "array",
		"details": {
			"type": "uint64[][32]",
			"internalType": "uint64[][32]"
		}
	}`)
	assert.NoError(t, err)
}

func TestOneOfSyntax(t *testing.T) {
	_, err := NewTestSchema(`{
		"type": "object",
		"details": {
			"type": "tuple",
			"internalType": "struct Custom.Thing"
		},
		"properties": {
			"customProp": {
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
					"index": 0
				},
				"description": "An integer. You are recommended to use a JSON string. A JSON number can be used for values up to the safe maximum."
			}
		}
	}`)
	assert.NoError(t, err)
}
