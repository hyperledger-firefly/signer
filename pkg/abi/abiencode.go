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

func (cv *ComponentValue) EncodeABIData() ([]byte, error) {
	return cv.EncodeABIDataCtx(context.Background())
}

func (cv *ComponentValue) EncodeABIDataCtx(ctx context.Context) ([]byte, error) {
	data, _, err := cv.encodeABIData(ctx, "")
	return data, err
}

func (cv *ComponentValue) encodeABIData(ctx context.Context, desc string) ([]byte, bool, error) {

	if cv == nil || cv.Component == nil {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, "nil")
	}
	tc := cv.Component.(*typeComponent)
	switch tc.cType {
	case ElementaryComponent:
		return tc.elementaryType.encodeABIData(ctx, desc, tc, cv.Value)
	case FixedArrayComponent:
		return cv.encodeABIChildren(ctx, desc, false /* only dynamic if the children are dynamic */, false /* no length */)
	case DynamicArrayComponent:
		return cv.encodeABIChildren(ctx, desc, true /* always dynamic */, true /* need length */)
	case TupleComponent:
		return cv.encodeABIChildren(ctx, desc, false /* only dynamic if the children are dynamic */, false /* no length */)
	default:
		return nil, false, i18n.NewError(ctx, signermsgs.MsgBadABITypeComponent, tc.cType)
	}

}

func (cv *ComponentValue) encodeABIChildren(ctx context.Context, desc string, knownDynamic, includeLen bool) (data []byte, dynamic bool, err error) {

	cData := make([][]byte, len(cv.Children))
	cDynamic := make([]bool, len(cv.Children))

	// Pass 1 generates the data
	for i, child := range cv.Children {
		cData[i], cDynamic[i], err = child.encodeABIData(ctx, fmt.Sprintf("%s[%d]", desc, i))
		if err != nil {
			return nil, false, err
		}
	}

	// Pass 2 calculates the length of the head
	headLen := 0
	tailLen := 0
	dynamic = knownDynamic // if we're a tuple, or variable length array, we're known to be dynamic
	for i := range cv.Children {
		if cDynamic[i] {
			headLen += 32
			tailLen += len(cData[i])
			// If any child is dynamic, we are dynamic
			dynamic = true
		} else {
			headLen += len(cData[i])
		}
	}

	// Pass 3 writes all the data into a single block
	startOffset := 0
	if includeLen {
		startOffset = 32
	}
	data = make([]byte, startOffset+headLen+tailLen)
	wData := data // where the head starts (might be after the length)
	headOffset := 0
	tailOffset := headLen
	if includeLen {
		big.NewInt(int64(len(cv.Children))).FillBytes(data[0:32])
		wData = data[32:]
	}
	for i := range cv.Children {
		if cDynamic[i] {
			// Write the offset of the data as uint256 in the head
			big.NewInt(int64(tailOffset)).FillBytes(wData[headOffset : headOffset+32])
			headOffset += 32
			// Write the data itself at that offset
			copy(wData[tailOffset:], cData[i])
			tailOffset += len(cData[i])
		} else {
			// Write the data itself in the head
			copy(wData[headOffset:], cData[i])
			headOffset += len(cData[i])
		}
	}
	return data, dynamic, nil

}

func encodeABIBytes(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	b, ok := value.([]byte)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "[]byte", value, desc)
	}

	// The type "bytes" (without a length suffix) is a variable encoding.
	// This comes out with a zero M value as the way we distinguish it from "function" which has default M of 24 (and also no suffix)
	if tc.m == 0 {
		return encodeABIDynamicBytes(b)
	}
	fixedLength := int(tc.m)

	// Belt and braces length check, although responsibility for generation of all the input data is within this package
	if len(b) < fixedLength || fixedLength > 32 {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgInsufficientDataABIEncode, fixedLength, len(b), desc)
	}

	// Copy into the front of a 32byte block, with trailing zeros.
	// That is the head, the data is empty
	data = make([]byte, 32)
	copy(data, b[0:fixedLength])
	return data, false, nil
}

func encodeABIString(ctx context.Context, desc string, _ *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	s, ok := value.(string)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "string", value, desc)
	}

	// Note we assume UTF-8 encoding has been assured of all input strings. No special handling here.
	return encodeABIDynamicBytes([]byte(s))
}

func encodeABIDynamicBytes(value []byte) (data []byte, dynamic bool, err error) {

	dataLen := 32 + // length is prefixed as uint256
		(len(value)/32)*32 // count of whole 32 byte chunks
	if (len(value) % 32) != 0 {
		dataLen += 32 // add 32 byte chunk for remainder
	}
	data = make([]byte, dataLen)
	_ = big.NewInt(int64(len(value))).FillBytes(data[0:32])
	copy(data[32:], value)

	return data, true, nil

}

func encodeABISignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	i, ok := value.(*big.Int)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if !checkSignedIntFits(i, tc.m) {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	return SerializeInt256TwosComplementBytes(i), false, nil
}

func encodeABIUnsignedInteger(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	i, ok := value.(*big.Int)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Int", value, desc)
	}

	// Reject integers that do not fit in the specified type
	if i.Sign() < 0 {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgNegativeUnsignedABIEncode, tc.m, desc)
	}
	if i.BitLen() > int(tc.m) {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgNumberTooLargeABIEncode, tc.m, desc)
	}

	data = make([]byte, 32)
	_ = i.FillBytes(data)
	return data, false, nil
}

func encodeFixed(ctx context.Context, desc string, tc *typeComponent, f *big.Float) (data []byte, dynamic bool, err error) {
	// Encoded as X * 10**N integer
	fN := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tc.n)), nil)
	f1 := new(big.Float).Mul(f, new(big.Float).SetInt(fN))
	i, _ := f1.Abs(f1).Int(nil)
	return encodeABISignedInteger(ctx, desc, tc, i)
}

func encodeABISignedFloat(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	f, ok := value.(*big.Float)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Float", value, desc)
	}
	return encodeFixed(ctx, desc, tc, f)
}

func encodeABIUnsignedFloat(ctx context.Context, desc string, tc *typeComponent, value interface{}) (data []byte, dynamic bool, err error) {
	f, ok := value.(*big.Float)
	if !ok {
		return nil, false, i18n.NewError(ctx, signermsgs.MsgWrongTypeComponentABIEncode, "*big.Float", value, desc)
	}
	return encodeFixed(ctx, desc, tc, f)
}
