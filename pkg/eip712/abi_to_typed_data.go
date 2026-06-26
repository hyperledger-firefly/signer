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
	"fmt"
	"regexp"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/abi"
)

var internalTypeStructExtractor = regexp.MustCompile(`^struct (.*\.)?([^.\[\]]+)(\[\d*\])*$`)

// Convert an ABI tuple definition, into the EIP-712 structure that's embedded into the
// "eth_signTypedData" signing request payload. It's a much simpler structure that
// flattens out a map of types (requiring each type to be named by a struct definition)
func ABItoTypedDataV4(ctx context.Context, tc abi.TypeComponent) (primaryType string, typeSet TypeSet, err error) {
	if tc.ComponentType() != abi.TupleComponent {
		return "", nil, i18n.NewError(ctx, signermsgs.MsgEIP712PrimaryNotTuple, tc.String())
	}
	primaryType, err = extractSolidityTypeName(ctx, tc.Parameter())
	if err != nil {
		return "", nil, err
	}
	// First we need to build the sorted array of types for `encodeType`
	typeSet = make(TypeSet)
	if err := addABITypes(ctx, tc, typeSet); err != nil {
		return "", nil, err
	}
	return primaryType, typeSet, nil
}

// Maps a parsed ABI component type to an EIP-712 type string
// - Subset of elementary types, with aliases resolved
// - Struct types are simply the name of the type
// - Fixed and dynamic array suffixes are supported
func mapABIType(ctx context.Context, tc abi.TypeComponent) (string, error) {
	switch tc.ComponentType() {
	case abi.TupleComponent:
		return extractSolidityTypeName(ctx, tc.Parameter())
	case abi.DynamicArrayComponent, abi.FixedArrayComponent:
		child, err := mapABIType(ctx, tc.ArrayChild())
		if err != nil {
			return "", err
		}
		if tc.ComponentType() == abi.FixedArrayComponent {
			return fmt.Sprintf("%s[%d]", child, tc.FixedArrayLen()), nil
		}
		return child + "[]", nil
	default:
		return mapElementaryABIType(ctx, tc)
	}
}

// Maps one of the parsed ABI elementary types to an EIP-712 elementary type
func mapElementaryABIType(ctx context.Context, tc abi.TypeComponent) (string, error) {
	if tc.ComponentType() != abi.ElementaryComponent {
		return "", i18n.NewError(ctx, signermsgs.MsgNotElementary, tc)
	}
	et := tc.ElementaryType()
	switch et.BaseType() {
	case abi.BaseTypeAddress, abi.BaseTypeBool, abi.BaseTypeString, abi.BaseTypeInt, abi.BaseTypeUInt:
		// Types that need no transposition
		return string(et.BaseType()) + tc.ElementarySuffix(), nil
	case abi.BaseTypeBytes:
		// Bytes is special
		if tc.ElementaryFixed() {
			return string(et.BaseType()) + tc.ElementarySuffix(), nil
		}
		return string(et.BaseType()), nil
	default:
		// EIP-712 does not support the other types
		return "", i18n.NewError(ctx, signermsgs.MsgEIP712UnsupportedABIType, tc)
	}
}

// ABI does not formally contain the Struct name - as it's not required for encoding the value.
// EIP-712 requires the Struct name as it is used through the standard, including to de-dup definitions
//
// Solidity uses the "internalType" field by convention as an extension to ABI, so we require
// that here for EIP-712 encoding to be successful.
func extractSolidityTypeName(ctx context.Context, param *abi.Parameter) (string, error) {
	match := internalTypeStructExtractor.FindStringSubmatch(param.InternalType)
	if match == nil {
		return "", i18n.NewError(ctx, signermsgs.MsgEIP712BadInternalType, param.InternalType)
	}
	return match[2], nil
}

// Recursively find all types, with a name -> encoded name map.
func addABITypes(ctx context.Context, tc abi.TypeComponent, typeSet TypeSet) error {
	switch tc.ComponentType() {
	case abi.TupleComponent:
		typeName, err := extractSolidityTypeName(ctx, tc.Parameter())
		if err != nil {
			return err
		}

		if _, mapped := typeSet[typeName]; mapped {
			// we've already mapped this type
			return nil
		}
		t := make(Type, len(tc.TupleChildren()))
		for i, child := range tc.TupleChildren() {
			ts, err := mapABIType(ctx, child)
			if err != nil {
				return err
			}
			t[i] = &TypeMember{
				Name: child.KeyName(),
				Type: ts,
			}
		}
		typeSet[typeName] = t
		// recurse
		for _, child := range tc.TupleChildren() {
			if err := addABITypes(ctx, child, typeSet); err != nil {
				return err
			}
		}
		return nil
	case abi.DynamicArrayComponent, abi.FixedArrayComponent:
		// recurse into the child
		return addABITypes(ctx, tc.ArrayChild(), typeSet)
	default:
		// from a type collection perspective, this is a leaf - nothing to do
		return nil
	}
}
