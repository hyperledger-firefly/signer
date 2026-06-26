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

package eip712

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/abi"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

type TypedData struct {
	Types       TypeSet                `ffstruct:"TypedData" json:"types"`
	PrimaryType string                 `ffstruct:"TypedData" json:"primaryType"`
	Domain      map[string]interface{} `ffstruct:"TypedData" json:"domain"`
	Message     map[string]interface{} `ffstruct:"TypedData" json:"message"`
}

type TypeMember struct {
	Name string
	Type string
}

type Type []*TypeMember

type TypeSet map[string]Type

const EIP712Domain = "EIP712Domain"

func EncodeTypedDataV4(ctx context.Context, payload *TypedData) (encoded ethtypes.HexBytes0xPrefix, err error) {
	// Add empty EIP712Domain type specification if missing
	if payload.Types == nil {
		payload.Types = TypeSet{}
	}
	if _, found := payload.Types[EIP712Domain]; !found {
		payload.Types[EIP712Domain] = Type{}
	}
	if payload.Domain == nil {
		payload.Domain = make(map[string]interface{})
	}
	if payload.PrimaryType == "" {
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712PrimaryTypeRequired)
	}

	// Start with the EIP-712 prefix
	buf := new(bytes.Buffer)
	buf.Write([]byte{0x19, 0x01})

	// Encode EIP712Domain from message
	domainHash, err := hashStruct(ctx, EIP712Domain, payload.Domain, payload.Types, "domain")
	if err != nil {
		return nil, err
	}
	buf.Write(domainHash)

	// If that wasn't the primary type, encode the primary type
	if payload.PrimaryType != EIP712Domain {
		// Encode the hash
		structHash, err := hashStruct(ctx, payload.PrimaryType, payload.Message, payload.Types, "")
		if err != nil {
			return nil, err
		}
		buf.Write(structHash)
	}

	encoded = buf.Bytes()
	log.L(ctx).Tracef("Encoded EIP-712: %s", encoded)
	return keccak256(encoded), nil
}

// A map from type names to types is encoded per encodeType:
//
// > If the struct type references other struct types (and these in turn reference even more struct types),
// > then the set of referenced struct types is collected, sorted by name and appended to the encoding.
func (ts TypeSet) Encode(primaryType string) string {
	// Write the primary type first always
	buff := new(strings.Builder)
	buff.WriteString(ts[primaryType].Encode(primaryType))

	// Then the reference types sorted by name
	referenceTypes := make([]string, 0, len(ts))
	for typeName := range ts {
		if typeName != primaryType {
			referenceTypes = append(referenceTypes, typeName)
		}
	}
	sort.Strings(referenceTypes)
	for _, typeName := range referenceTypes {
		buff.WriteString(ts[typeName].Encode(typeName))
	}
	return buff.String()
}

// An individual member is encoded as:
//
// > type ‖ " " ‖ name
func (tm *TypeMember) Encode() string {
	return tm.Type + " " + tm.Name
}

// A type is encoded as:
//
// > name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"
func (t Type) Encode(name string) string {
	buff := new(strings.Builder)
	buff.WriteString(name)
	buff.WriteRune('(')
	for i, tm := range t {
		if i > 0 {
			buff.WriteRune(',')
		}
		buff.WriteString(tm.Encode())
	}
	buff.WriteRune(')')
	return buff.String()
}

func nextCrumb(breadcrumbs string, name string) string {
	if len(breadcrumbs) > 0 {
		return breadcrumbs + "." + name
	}
	return name
}

func idxCrumb(breadcrumbs string, idx int) string {
	return fmt.Sprintf("%s[%d]", breadcrumbs, idx)
}

func addNestedTypes(typeName string, allTypes TypeSet, typeSet TypeSet) {
	// We're not interested in array semantics here
	iBracket := strings.Index(typeName, "[")
	if iBracket >= 0 {
		typeName = typeName[0:iBracket]
	}
	// See if it's a defined structure type
	t, ok := allTypes[typeName]
	if ok && typeSet[typeName] == nil {
		typeSet[typeName] = t
		for _, tm := range t {
			addNestedTypes(tm.Type, allTypes, typeSet)
		}
	}
}

func keccak256(b []byte) ethtypes.HexBytes0xPrefix {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(b)
	return hash.Sum(nil)
}

// A map from type names to types is encoded per encodeType:
//
// > If the struct type references other struct types (and these in turn reference even more struct types),
// > then the set of referenced struct types is collected, sorted by name and appended to the encoding.
func encodeType(ctx context.Context, typeName string, allTypes TypeSet) (Type, string, error) {
	t := allTypes[typeName]
	if t == nil {
		return nil, "", i18n.NewError(ctx, signermsgs.MsgEIP712TypeNotFound, typeName)
	}

	depSet := make(TypeSet)
	addNestedTypes(typeName, allTypes, depSet)
	typeEncoded := depSet.Encode(typeName)
	log.L(ctx).Tracef("encodeType(%s): %s", typeName, typeEncoded)
	return t, typeEncoded, nil
}

func encodeData(ctx context.Context, typeName string, v interface{}, allTypes TypeSet, breadcrumbs string) (encoded ethtypes.HexBytes0xPrefix, err error) {
	// Get the local typeset for the struct and all its deps
	t, typeEncoded, err := encodeType(ctx, typeName, allTypes)
	if err != nil {
		return nil, err
	}
	// Check the value we have is a map
	var vMap map[string]interface{}
	switch vt := v.(type) {
	case nil:
	case map[string]interface{}:
		vMap = vt
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712ValueNotMap, breadcrumbs, v)
	}
	if vMap == nil {
		// V4 says the caller writes an empty bytes32, rather than a hash of anything
		return nil, nil
	}
	typeHashed := keccak256([]byte(typeEncoded))
	buf := bytes.NewBuffer(typeHashed)
	log.L(ctx).Tracef("hashType(%s): %s", typeName, typeHashed)
	// Encode the data of the struct, and write it after the hash of the type
	for _, tm := range t {
		b, err := encodeElement(ctx, tm.Type, vMap[tm.Name], allTypes, nextCrumb(breadcrumbs, tm.Name))
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	encoded = buf.Bytes()
	log.L(ctx).Tracef("encodeData(%s, %T): %s", typeName, v, encoded)
	return encoded, nil
}

// HashStruct allows hashing of an individual structure, without the EIP-712 domain
func HashStruct(ctx context.Context, typeName string, v interface{}, allTypes TypeSet) (result ethtypes.HexBytes0xPrefix, err error) {
	return hashStruct(ctx, typeName, v, allTypes, "")
}

func hashStruct(ctx context.Context, typeName string, v interface{}, allTypes TypeSet, breadcrumbs string) (result ethtypes.HexBytes0xPrefix, err error) {
	encoded, err := encodeData(ctx, typeName, v, allTypes, breadcrumbs)
	if err != nil {
		return nil, err
	}
	if encoded == nil {
		// special rule for a nil value - we don't even include the type info, just return a nil bytes array
		bytes32Enc, _ := abiElementaryType(ctx, "bytes32")
		encoded, _ = abiEncode(ctx, bytes32Enc, "0x0000000000000000000000000000000000000000000000000000000000000000", breadcrumbs)
		result = encoded
	} else {
		result = keccak256(encoded)
	}
	log.L(ctx).Tracef("hashStruct(%s): %s", typeName, result)
	return result, nil
}

func encodeElement(ctx context.Context, typeName string, v interface{}, allTypes TypeSet, breadcrumbs string) (ethtypes.HexBytes0xPrefix, error) {
	if strings.HasSuffix(typeName, "]") {
		// recurse into the array
		return hashArray(ctx, typeName, allTypes, v, breadcrumbs)
	} else if _, isStruct := allTypes[typeName]; isStruct {
		// recurse into the struct
		return hashStruct(ctx, typeName, v, allTypes, breadcrumbs)
	}
	// Need to process based on the Elementary type
	tc, err := abiElementaryType(ctx, typeName)
	if err != nil {
		return nil, err
	}
	baseType := tc.ElementaryType().BaseType()
	switch baseType {
	case abi.BaseTypeAddress, abi.BaseTypeBool, abi.BaseTypeInt, abi.BaseTypeUInt:
		return abiEncode(ctx, tc, v, breadcrumbs)
	case abi.BaseTypeBytes:
		// Handle fixed bytes1 to bytes32
		if baseType == abi.BaseTypeBytes && tc.ElementaryFixed() {
			return abiEncode(ctx, tc, v, breadcrumbs)
		}
		// These dynamic bytes/string arrays are special handling, where we need to use the same
		// rules as ABI to extract the byte string from the input... but we need to actually
		// return a keccak256 of the contents
		// - We have special knowledge here that the type will be coercible to []byte
		reader := tc.ElementaryType().DataReader()
		di, err := reader(ctx, breadcrumbs, v)
		if err != nil {
			return nil, err
		}
		return keccak256(di.([]byte)), nil
	case abi.BaseTypeString:
		reader := tc.ElementaryType().DataReader()
		di, err := reader(ctx, breadcrumbs, v)
		if err != nil {
			return nil, err
		}
		return keccak256([]byte(di.(string))), nil
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712UnsupportedABIType, tc)
	}
}

func abiElementaryType(ctx context.Context, typeName string) (abi.TypeComponent, error) {
	p := &abi.Parameter{Type: typeName}
	tc, err := p.TypeComponentTreeCtx(ctx)
	if err != nil {
		return nil, err
	}
	if tc.ComponentType() != abi.ElementaryComponent {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotElementary, tc)
	}
	return tc, nil
}

func abiEncode(ctx context.Context, tc abi.TypeComponent, v interface{}, breadcrumbs string) (b ethtypes.HexBytes0xPrefix, err error) {
	// Re-use the ABI function to parse the input value for Elementary types.
	// (we weren't able to do this for structs/tuples and arrays, due to EIP-712 specifics)
	cv, err := tc.ParseExternalDesc(ctx, v, breadcrumbs)
	if err == nil {
		b, err = cv.EncodeABIDataCtx(ctx)
	}
	if err != nil {
		return nil, err
	}
	log.L(ctx).Tracef("encodeElement(%s, %T): %s", tc.String(), v, b)
	return b, nil
}

// hashArray is only called when the last character of the type is `]`
func hashArray(ctx context.Context, typeName string, allTypes TypeSet, v interface{}, breadcrumbs string) (ethtypes.HexBytes0xPrefix, error) {
	// Extract the dimension of the array
	openPos := strings.LastIndex(typeName, "[")
	if openPos <= 0 || typeName[len(typeName)-1] != ']' {
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712InvalidArraySuffix, typeName)
	}
	dimStr := typeName[openPos+1 : len(typeName)-1]
	trimmedTypeName := typeName[0:openPos]

	// We should have an array in the input.
	// Note Go JSON unmarshal always gives []interface{}, regardless of type of entry.
	va, ok := v.([]interface{})
	if !ok {
		return nil, i18n.NewError(ctx, signermsgs.MsgEIP712ValueNotArray, typeName, v)
	}
	// If we have a fixed dimension, then check we have the right number of elements
	if dimStr != "" {
		dim, err := strconv.Atoi(dimStr)
		if err != nil {
			return nil, i18n.NewError(ctx, signermsgs.MsgEIP712InvalidArraySuffix, typeName)
		}
		if len(va) != dim {
			return nil, i18n.NewError(ctx, signermsgs.MsgEIP712InvalidArrayLen, typeName, dim, len(va))
		}
	}
	// Append all the data
	buf := new(bytes.Buffer)
	for i, ve := range va {
		b, err := encodeElement(ctx, trimmedTypeName, ve, allTypes, idxCrumb(breadcrumbs, i))
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	return keccak256(buf.Bytes()), nil
}
