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

package abi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
)

// walkTupleABIBytes is the main entry point to the logic, decoding a list of parameters at a position
func walkTupleABIBytes(ctx context.Context, block []byte, offset int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {
	return walkDynamicChildArrayABIBytes(ctx, "tup", "", block, offset, offset, component, component.tupleChildren)
}

// decodeABIElement is called for each entry in a tuple, or array, to process the head bytes,
// and any offset data bytes for that entry. The number of head bytes consumed is returned.
//
// Note that this is used for tuples embedded within a parent array or tuple, as that will be
// a variable structure pointed to from the header. But for the root tuples, the walkTupleABIBytes entry
// point is used that will kick off the recursive process directly from a given offset.
//
// - headStart is the absolute location in the byte array of the beginning of the current header, that all offsets will be relative to
// - headPosition is the absolute location of where we are reading the header from for this element
//
// So for example headStart=4,headPosition=4 would mean we are reading from the beginning of the primary header, after
// the 4 byte function selector in a function call parameter.
func decodeABIElement(ctx context.Context, breadcrumbs string, block []byte, headStart, headPosition int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {

	switch component.cType {
	case ElementaryComponent:
		// All elementary types consume exactly 32 bytes from the head.
		// Any variable data goes into the data section (calculated as an offset from the headStart)
		cv, err := component.elementaryType.decodeABIData(ctx, breadcrumbs, block, headStart, headPosition, component)
		if err != nil {
			return -1, nil, err
		}
		// So we move the position beyond the data length of the element
		return 32, cv, err
	case FixedArrayComponent:
		dynamic, err := isDynamicType(ctx, component)
		if err != nil {
			return -1, nil, err
		}
		if dynamic {
			headOffset, err := decodeABILength(ctx, breadcrumbs, block, headPosition)
			if err != nil {
				return -1, nil, err
			}
			headStart += headOffset
			headPosition = headStart

			// Fixed arrays of dynamic types are encoded identically to a tuple with all entries the same type
			children := make([]*typeComponent, component.arrayLength)
			for i := 0; i < component.arrayLength; i++ {
				children[i] = component.arrayChild
			}
			_, cv, err = walkDynamicChildArrayABIBytes(ctx, "fix", breadcrumbs, block, headStart, headPosition, component, children)
			return 32, cv, err // consumes 32 bytes from head
		}
		// If the fixed array, contains only fixed types - decode the fixed array at that position
		return decodeABIFixedArrayBytes(ctx, breadcrumbs, block, headStart, headPosition, component)
	case DynamicArrayComponent:
		headOffset, err := decodeABILength(ctx, breadcrumbs, block, headPosition)
		if err != nil {
			return -1, nil, err
		}
		cv, err := decodeABIDynamicArrayBytes(ctx, breadcrumbs, block, headStart+headOffset, component)
		if err != nil {
			return -1, nil, err
		}
		return 32, cv, err
	case TupleComponent:
		dynamic, err := isDynamicType(ctx, component)
		if err != nil {
			return -1, nil, err
		}
		if dynamic {
			headOffset, err := decodeABILength(ctx, breadcrumbs, block, headPosition)
			if err != nil {
				return -1, nil, err
			}
			headStart += headOffset
			headPosition = headStart
		}

		headBytesRead, cv, err := walkDynamicChildArrayABIBytes(ctx, "tup", breadcrumbs, block, headStart, headPosition, component, component.tupleChildren)
		if dynamic {
			// In the case where it's dynamic we only read one block
			headBytesRead = 32
		}
		return headBytesRead, cv, err
	default:
		return -1, nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, component.cType)
	}

}

func decodeABISignedInt(ctx context.Context, desc string, block []byte, _, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv = &ComponentValue{Component: component}
	if headPosition+32 > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}
	cv.Value = ParseInt256TwosComplementBytes(block[headPosition : headPosition+32])
	return cv, err
}

func decodeABIUnsignedInt(ctx context.Context, desc string, block []byte, _, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv = &ComponentValue{Component: component}
	if headPosition+32 > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}

	// When we're reading bytes, need to make sure we're reading the correct size number of bytes for the uint size
	cv.Value = new(big.Int).SetBytes(block[headPosition+(32-(int(component.m/8))) : headPosition+32])
	return cv, err
}

func intToFixed(ctx context.Context, component *typeComponent, cv *ComponentValue) (*ComponentValue, error) {
	if component.n == 0 {
		return nil, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, component)
	}
	f := new(big.Float).SetInt(cv.Value.(*big.Int))
	fN := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(component.n)), nil)
	cv.Value = f.Quo(f, new(big.Float).SetInt(fN))
	return cv, nil
}

func decodeABISignedFloat(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABISignedInt(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	return intToFixed(ctx, component, cv)
}

func decodeABIUnsignedFloat(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABIUnsignedInt(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	return intToFixed(ctx, component, cv)
}

func decodeABILength(ctx context.Context, desc string, block []byte, offset int) (count int, err error) {
	if offset+32 > len(block) {
		return -1, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIArrayCount, desc)
	}
	i := new(big.Int).SetBytes(block[offset : offset+32])
	if i.BitLen() > 32 {
		return -1, i18n.NewError(ctx, signermsgs.MsgABIArrayCountTooLarge, i.Text(10), desc)
	}
	return int(i.Int64()), nil
}

func decodeABIBytes(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	var byteLength int
	dataOffset := headPosition
	if component.m == 0 {
		// Variable length bytes. Offset to data in head...
		dataOffset, err = decodeABILength(ctx, desc, block, headPosition)
		if err != nil {
			return nil, err
		}
		dataOffset = headStart + dataOffset
		// ... then array length ahead of the bytes themselves
		byteLength, err = decodeABILength(ctx, desc, block, dataOffset)
		if err != nil {
			return nil, err
		}
		dataOffset += 32
	} else {
		byteLength = int(component.m)
	}
	cv = &ComponentValue{Component: component}
	if dataOffset+byteLength > len(block) {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABIValue, component, desc)
	}
	b := make([]byte, byteLength)
	copy(b, block[dataOffset:])
	cv.Value = b
	// Byte arrays always consume 32b in the head.
	return cv, err
}

func decodeABIString(ctx context.Context, desc string, block []byte, headStart, headPosition int, component *typeComponent) (cv *ComponentValue, err error) {
	cv, err = decodeABIBytes(ctx, desc, block, headStart, headPosition, component)
	if err != nil {
		return nil, err
	}
	cv.Value = string(cv.Value.([]byte))
	return cv, err
}

func decodeABIFixedArrayBytes(ctx context.Context, breadcrumbs string, block []byte, headStart, headPosition int, component *typeComponent) (headBytesRead int, cv *ComponentValue, err error) {

	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, component.arrayLength),
	}
	headBytesRead = 0
	for i := 0; i < component.arrayLength; i++ {
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[fix,i:%d,o:%d]", breadcrumbs, i, headPosition),
			block, headStart, headPosition, component.arrayChild)
		if err != nil {
			return -1, nil, err
		}
		cv.Children[i] = child
		headBytesRead += childHeadBytes
		headPosition += childHeadBytes
	}
	return headBytesRead, cv, err
}

// isDynamicType does a recursive check of a type sub-tree, to see if it is dynamic according to
// the rules in the ABI specification.
func isDynamicType(ctx context.Context, tc *typeComponent) (bool, error) {
	switch tc.cType {
	case TupleComponent:
		for _, childType := range tc.tupleChildren {
			childDynamic, err := isDynamicType(ctx, childType)
			if err != nil {
				return false, err
			}
			if childDynamic {
				return true, nil
			}
		}
		return false, nil
	case FixedArrayComponent:
		if tc.arrayLength == 0 {
			return false, nil
		}
		return isDynamicType(ctx, tc.arrayChild)
	case DynamicArrayComponent:
		return true, nil
	case ElementaryComponent:
		// The dynamic() function is because "bytes32" is fixed, but "bytes" is variable.
		return tc.elementaryType.dynamic(tc), nil
	default:
		return false, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, tc.cType)
	}
}

func decodeABIDynamicArrayBytes(ctx context.Context, breadcrumbs string, block []byte, dataOffset int, component *typeComponent) (cv *ComponentValue, err error) {
	arrayLength, err := decodeABILength(ctx, breadcrumbs, block, dataOffset)
	if err != nil {
		return nil, err
	}
	dataOffset += 32
	dataStart := dataOffset
	cv = &ComponentValue{
		Component: component,
		Children:  make([]*ComponentValue, arrayLength),
	}
	for i := 0; i < arrayLength; i++ {
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[dyn,i:%d,b:%d]", breadcrumbs, i, dataOffset),
			block, dataStart, dataOffset, component.arrayChild)
		if err != nil {
			return nil, err
		}
		cv.Children[i] = child
		dataOffset += childHeadBytes
	}
	return cv, err

}

func walkDynamicChildArrayABIBytes(ctx context.Context, desc, breadcrumbs string, block []byte, headStart, headPosition int, parent *typeComponent, children []*typeComponent) (headBytesRead int, cv *ComponentValue, err error) {
	cv = &ComponentValue{
		Component: parent,
		Children:  make([]*ComponentValue, len(children)),
	}
	headBytesRead = 0
	for i, child := range children {
		// Read the child at its head location
		childHeadBytes, child, err := decodeABIElement(ctx, fmt.Sprintf("%s[%s,i:%d,b:%d]", breadcrumbs, desc, i, headPosition),
			block, headStart, headPosition, child)
		if err != nil {
			return -1, nil, err
		}
		cv.Children[i] = child
		headBytesRead += childHeadBytes
		headPosition += childHeadBytes
	}
	return headBytesRead, cv, err
}
