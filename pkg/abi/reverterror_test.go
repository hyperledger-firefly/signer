// Copyright © 2026 Kaleido, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeTestError builds a properly typed ComponentValue by encoding and then
// decoding through the ABI pipeline.
func decodeTestError(t *testing.T, entry *Entry, jsonArgs string) *ComponentValue {
	t.Helper()
	encoded, err := entry.EncodeCallDataJSON([]byte(jsonArgs))
	require.NoError(t, err)
	cv, err := entry.DecodeCallDataCtx(context.Background(), encoded)
	require.NoError(t, err)
	return cv
}

// --- Nil receiver safety ---

func TestRevertErrorNilString(t *testing.T) {
	var r *RevertError
	assert.Equal(t, "", r.String())
}

func TestRevertErrorNilSignature(t *testing.T) {
	var r *RevertError
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "", sig)
}

func TestRevertErrorNilSerializeJSON(t *testing.T) {
	var r *RevertError
	b, err := r.SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, b)
}

func TestRevertErrorNilInnerError(t *testing.T) {
	var r *RevertError
	assert.Nil(t, r.GetInnerError())
}

func TestRevertErrorNilInnermost(t *testing.T) {
	var r *RevertError
	assert.Nil(t, r.Innermost())
}

func TestRevertErrorNilErrors(t *testing.T) {
	var r *RevertError
	assert.Nil(t, r.Errors())
}

// --- Single (non-nested) error ---

func TestRevertErrorSingleString(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	assert.Equal(t, `AnError("something went wrong")`, r.String())
}

func TestRevertErrorSingleSignature(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "AnError(string)", sig)
}

func TestRevertErrorSingleSerializeJSON(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	b, err := r.SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "something went wrong")
}

func TestRevertErrorSingleSerializeJSONNilCV(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	r := &RevertError{ErrorEntry: entry}
	b, err := r.SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, b)
}

func TestRevertErrorSingleInnerError(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	assert.Nil(t, r.GetInnerError())
}

func TestRevertErrorSingleInnermost(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	assert.Equal(t, r, r.Innermost())
}

func TestRevertErrorSingleErrors(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	errs := r.Errors()
	require.Len(t, errs, 1)
	assert.Equal(t, r, errs[0])
}

func TestRevertErrorSingleWithPrefix(t *testing.T) {
	entry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"reason":"plain error"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv, Prefix: "context: "}
	assert.Equal(t, `context: Error("plain error")`, r.String())
}

func TestRevertErrorMultipleParams(t *testing.T) {
	entry := &Entry{Type: Error, Name: "ExampleError", Inputs: ParameterArray{
		{Name: "param1", Type: "string"},
		{Name: "param2", Type: "uint256"},
	}}
	cv := decodeTestError(t, entry, `{"param1":"test1","param2":12345}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	assert.Equal(t, `ExampleError("test1","12345")`, r.String())
}

// --- Two-level InnerError ---

func TestRevertErrorNestedTwoLevelString(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outerCV := decodeTestError(t, outerEntry, `{"reason":"raw outer value"}`)
	outer := &RevertError{
		ErrorEntry: outerEntry,
		cv:         outerCV,
		Prefix:     "[404]caught bytes",
		InnerError: inner,
	}
	assert.Equal(t, `[404]caught bytesAnError("I am an error")`, outer.String())
}

func TestRevertErrorNestedTwoLevelSignatures(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{
		ErrorEntry: outerEntry,
		Prefix:     "[404]caught bytes",
		InnerError: inner,
	}

	outerSig, err := outer.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Error(string)", outerSig)

	innerSig, err := inner.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "AnError(string)", innerSig)
}

func TestRevertErrorNestedTwoLevelInnerError(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", InnerError: inner}

	assert.Equal(t, inner, outer.GetInnerError())
	assert.Nil(t, inner.GetInnerError())
}

func TestRevertErrorNestedTwoLevelInnermost(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", InnerError: inner}

	assert.Equal(t, inner, outer.Innermost())
	assert.Equal(t, inner, inner.Innermost())
}

func TestRevertErrorNestedTwoLevelErrors(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", InnerError: inner}

	errs := outer.Errors()
	require.Len(t, errs, 2)
	assert.Equal(t, outer, errs[0])
	assert.Equal(t, inner, errs[1])
}

func TestRevertErrorNestedTwoLevelSerializeJSONInnermost(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outerCV := decodeTestError(t, outerEntry, `{"reason":"raw bytes here"}`)
	outer := &RevertError{ErrorEntry: outerEntry, cv: outerCV, Prefix: "[404]caught bytes", InnerError: inner}

	b, err := outer.Innermost().SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "I am an error")
}

// --- Three-level InnerError ---

func TestRevertErrorNestedThreeLevelString(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middleEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	middle := &RevertError{ErrorEntry: middleEntry, Prefix: "middleware: ", InnerError: leaf}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "gateway: ", InnerError: middle}

	assert.Equal(t, `gateway: middleware: RootCause("the real problem")`, outer.String())
}

func TestRevertErrorNestedThreeLevelInnermost(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "middleware: ", InnerError: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "gateway: ", InnerError: middle}

	assert.Equal(t, leaf, outer.Innermost())
}

func TestRevertErrorNestedThreeLevelErrors(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "middleware: ", InnerError: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "gateway: ", InnerError: middle}

	errs := outer.Errors()
	require.Len(t, errs, 3)
	assert.Equal(t, outer, errs[0])
	assert.Equal(t, middle, errs[1])
	assert.Equal(t, leaf, errs[2])
}

func TestRevertErrorNestedThreeLevelInnerErrorChain(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, InnerError: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, InnerError: middle}

	assert.Equal(t, middle, outer.GetInnerError())
	assert.Equal(t, leaf, outer.GetInnerError().GetInnerError())
	assert.Nil(t, outer.GetInnerError().GetInnerError().GetInnerError())
}

// --- SerializeJSON with custom serializer ---

func TestRevertErrorSerializeJSONCustomSerializer(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"test value"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	s := NewSerializer().SetPretty(true)
	b, err := r.SerializeJSON(context.Background(), s)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "\n")
	assert.Contains(t, string(b), "test value")
}

// --- Edge cases ---

func TestRevertErrorEmptyPrefix(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"direct"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outer := &RevertError{
		ErrorEntry: &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}},
		Prefix:     "",
		InnerError: inner,
	}
	assert.Equal(t, `AnError("direct")`, outer.String())
}

func TestRevertErrorSignatureNilEntry(t *testing.T) {
	r := &RevertError{}
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "", sig)
}

// --- DecodeRevertError / DecodeRevertErrorCtx ---

func TestDecodeRevertErrorDefaultErrorString(t *testing.T) {
	revertData := testEncodeError(t,
		&Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}},
		`{"reason":"Not enough Ether provided."}`,
	)
	r := ABI{}.DecodeRevertError(revertData)
	require.NotNil(t, r)
	assert.Equal(t, "Error", r.ErrorEntry.Name)
	assert.Equal(t, `Error("Not enough Ether provided.")`, r.String())
	assert.Nil(t, r.GetInnerError())
}

func TestDecodeRevertErrorDefaultPanic(t *testing.T) {
	revertData := testEncodeError(t,
		&Entry{Type: Error, Name: "Panic", Inputs: ParameterArray{{Name: "code", Type: "uint256"}}},
		`{"code":1}`,
	)
	r := ABI{}.DecodeRevertError(revertData)
	require.NotNil(t, r)
	assert.Equal(t, "Panic", r.ErrorEntry.Name)
	assert.Equal(t, `Panic("1")`, r.String())
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Panic(uint256)", sig)
}

func TestDecodeRevertErrorCustomError(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "InsufficientBalance", Inputs: ParameterArray{
		{Name: "available", Type: "uint256"},
		{Name: "required", Type: "uint256"},
	}}
	customABI := ABI{customEntry}
	revertData := testEncodeError(t, customEntry, `{"available":100,"required":200}`)

	r := customABI.DecodeRevertError(revertData)
	require.NotNil(t, r)
	assert.Equal(t, "InsufficientBalance", r.ErrorEntry.Name)
	assert.Equal(t, `InsufficientBalance("100","200")`, r.String())
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "InsufficientBalance(uint256,uint256)", sig)
}

func TestDecodeRevertErrorCustomBeforeBuiltin(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "MyError", Inputs: ParameterArray{{Name: "msg", Type: "string"}}}
	customABI := ABI{customEntry}
	revertData := testEncodeError(t, customEntry, `{"msg":"custom message"}`)

	r := customABI.DecodeRevertError(revertData)
	require.NotNil(t, r)
	assert.Equal(t, "MyError", r.ErrorEntry.Name, "custom ABI entries should be tried before builtins")
}

func TestDecodeRevertErrorNoMatch(t *testing.T) {
	r := ABI{}.DecodeRevertError([]byte{0x11, 0x22, 0x33, 0x44})
	assert.Nil(t, r)
}

func TestDecodeRevertErrorTooShort(t *testing.T) {
	r := ABI{}.DecodeRevertError([]byte{0x08})
	assert.Nil(t, r)
}

func TestDecodeRevertErrorNilData(t *testing.T) {
	r := ABI{}.DecodeRevertError(nil)
	assert.Nil(t, r)
}

func TestDecodeRevertErrorEmptyData(t *testing.T) {
	r := ABI{}.DecodeRevertError([]byte{})
	assert.Nil(t, r)
}

func TestDecodeRevertErrorCtxPassesContext(t *testing.T) {
	revertData := testEncodeError(t,
		&Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}},
		`{"reason":"with context"}`,
	)
	ctx := context.Background()
	r := ABI{}.DecodeRevertErrorCtx(ctx, revertData)
	require.NotNil(t, r)
	assert.Equal(t, `Error("with context")`, r.String())
}

func TestDecodeRevertErrorSerializeJSON(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "ExampleError", Inputs: ParameterArray{
		{Name: "param1", Type: "string"},
		{Name: "param2", Type: "uint256"},
	}}
	revertData := testEncodeError(t, customEntry, `{"param1":"test1","param2":12345}`)
	r := ABI{customEntry}.DecodeRevertError(revertData)
	require.NotNil(t, r)

	b, err := r.SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "test1")
	assert.Contains(t, string(b), "12345")
}

func TestDecodeRevertErrorNonErrorEntriesIgnored(t *testing.T) {
	fnEntry := &Entry{Type: Function, Name: "transfer", Inputs: ParameterArray{
		{Name: "to", Type: "address"},
		{Name: "amount", Type: "uint256"},
	}}
	r := ABI{fnEntry}.DecodeRevertError([]byte{0x11, 0x22, 0x33, 0x44})
	assert.Nil(t, r, "function entries should not be tried for error decoding")
}

// testEncodeError is a helper that ABI-encodes error data for a given entry and JSON args.
func testEncodeError(t *testing.T, entry *Entry, jsonArgs string) []byte {
	t.Helper()
	encoded, err := entry.EncodeCallDataJSON([]byte(jsonArgs))
	require.NoError(t, err)
	return encoded
}

// --- InnerError unwrapping (DecodeRevertError with nesting) ---

func TestDecodeRevertErrorSingleNested(t *testing.T) {
	innerABI := buildErrorStringABI([]byte("inner error message"))
	outerABI := buildErrorStringABI(append([]byte("outer: "), innerABI...))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Equal(t, "Error", r.ErrorEntry.Name)
	assert.Equal(t, "outer: ", r.Prefix)

	require.NotNil(t, r.GetInnerError())
	assert.Equal(t, "Error", r.GetInnerError().ErrorEntry.Name)
	assert.Nil(t, r.GetInnerError().GetInnerError())

	assert.Equal(t, `outer: Error("inner error message")`, r.String())
}

func TestDecodeRevertErrorDoubleNested(t *testing.T) {
	deepABI := buildErrorStringABI([]byte("deepest error"))
	midABI := buildErrorStringABI(append([]byte("level2: "), deepABI...))
	outerABI := buildErrorStringABI(append([]byte("level1: "), midABI...))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Equal(t, "level1: ", r.Prefix)

	mid := r.GetInnerError()
	require.NotNil(t, mid)
	assert.Equal(t, "level2: ", mid.Prefix)

	leaf := mid.GetInnerError()
	require.NotNil(t, leaf)
	assert.Nil(t, leaf.GetInnerError())

	assert.Equal(t, `level1: level2: Error("deepest error")`, r.String())
	assert.Equal(t, leaf, r.Innermost())

	errs := r.Errors()
	require.Len(t, errs, 3)
}

func TestDecodeRevertErrorNestedCustomError(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "MyCustomError", Inputs: ParameterArray{{Type: "bytes"}}}
	customEncoded := testEncodeError(t, customEntry, `{"0":"0xdeadbeef"}`)

	outerABI := buildErrorStringABI(append([]byte("[404]01d - caught bytes:"), customEncoded...))

	r := ABI{customEntry}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Equal(t, "Error", r.ErrorEntry.Name)
	assert.Equal(t, "[404]01d - caught bytes:", r.Prefix)

	inner := r.GetInnerError()
	require.NotNil(t, inner)
	assert.Equal(t, "MyCustomError", inner.ErrorEntry.Name)
	assert.Nil(t, inner.GetInnerError())

	sig, err := inner.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "MyCustomError(bytes)", sig)

	assert.Equal(t, `[404]01d - caught bytes:MyCustomError("0xdeadbeef")`, r.String())
}

func TestDecodeRevertErrorCustomBeforeDefaultNested(t *testing.T) {
	customEntry := &Entry{Type: Error, Name: "EarlyErr", Inputs: ParameterArray{{Type: "uint256"}}}
	customEncoded := testEncodeError(t, customEntry, `{"0":42}`)

	innerErrorABI := buildErrorStringABI([]byte("late-error"))
	// Custom selector appears before the Error(string) selector
	payload := append([]byte("head:"), customEncoded...)
	payload = append(payload, []byte("middle:")...)
	payload = append(payload, innerErrorABI...)
	outerABI := buildErrorStringABI(payload)

	r := ABI{customEntry}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Equal(t, "head:", r.Prefix)

	inner := r.GetInnerError()
	require.NotNil(t, inner)
	assert.Equal(t, "EarlyErr", inner.ErrorEntry.Name, "first matching selector wins")
}

func TestDecodeRevertErrorNoNesting(t *testing.T) {
	outerABI := buildErrorStringABI([]byte("plain error with no nesting"))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Equal(t, "", r.Prefix)
	assert.Nil(t, r.GetInnerError())
	assert.Equal(t, `Error("plain error with no nesting")`, r.String())
}

func TestDecodeRevertErrorMalformedNested(t *testing.T) {
	defaultErr := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	sel := defaultErr.FunctionSelectorBytes()

	badData := append([]byte("prefix:"), sel...)
	badData = append(badData, []byte("truncated")...)
	outerABI := buildErrorStringABI(badData)

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Nil(t, r.GetInnerError(), "malformed inner data should not produce an inner error")
	assert.Equal(t, "", r.Prefix)
}

func TestDecodeRevertErrorDepthLimit(t *testing.T) {
	// Build a chain deeper than maxRevertErrorDepth (10)
	data := []byte("leaf")
	for i := 0; i < maxRevertErrorDepth+2; i++ {
		data = buildErrorStringABI(append([]byte("L:"), data...))
	}

	r := ABI{}.DecodeRevertError(data)
	require.NotNil(t, r)

	depth := 0
	for cur := r; cur != nil; cur = cur.GetInnerError() {
		depth++
	}
	assert.LessOrEqual(t, depth, maxRevertErrorDepth+1, "chain should be capped by depth limit")
}

func TestDecodeRevertErrorInnermostSerializeJSON(t *testing.T) {
	innerABI := buildErrorStringABI([]byte("inner value"))
	outerABI := buildErrorStringABI(append([]byte("prefix:"), innerABI...))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)

	leaf := r.Innermost()
	require.NotNil(t, leaf)
	b, err := leaf.SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "inner value")
}

func TestDecodeRevertErrorNestedSignatures(t *testing.T) {
	innerABI := buildErrorStringABI([]byte("inner"))
	outerABI := buildErrorStringABI(append([]byte("prefix:"), innerABI...))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)

	outerSig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Error(string)", outerSig)

	innerSig, err := r.GetInnerError().Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Error(string)", innerSig)
}

// --- findSelector constraints ---

func TestFindSelectorScanCapExceeded(t *testing.T) {
	// Place a valid inner error beyond the maxInnerErrorScanBytes boundary.
	// It should not be found.
	innerABI := buildErrorStringABI([]byte("inner"))
	prefix := make([]byte, maxInnerErrorScanBytes) // pushes selector past the cap
	outerABI := buildErrorStringABI(append(prefix, innerABI...))

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Nil(t, r.GetInnerError(), "selector beyond scan cap should not be found")
}

func TestFindSelectorInsufficientBytesAfterSelector(t *testing.T) {
	// Build a payload where the selector appears near the end of the string
	// with fewer than minABIEncodedLen bytes remaining after it — too short
	// to hold a valid ABI encoding.
	entry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	sel := entry.FunctionSelectorBytes()

	// Append only 3 bytes after the selector — less than the 32-byte minimum word.
	truncated := append([]byte(nil), sel...)
	truncated = append(truncated, 0x00, 0x00, 0x00)
	outerABI := buildErrorStringABI(truncated)

	r := ABI{}.DecodeRevertError(outerABI)
	require.NotNil(t, r)
	assert.Nil(t, r.GetInnerError(), "selector with insufficient trailing bytes should be skipped")
}
