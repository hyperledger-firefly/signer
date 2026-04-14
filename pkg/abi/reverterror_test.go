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

func TestRevertErrorNilCause(t *testing.T) {
	var r *RevertError
	assert.Nil(t, r.Cause())
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

func TestRevertErrorSingleCause(t *testing.T) {
	entry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	cv := decodeTestError(t, entry, `{"message":"something went wrong"}`)
	r := &RevertError{ErrorEntry: entry, cv: cv}
	assert.Nil(t, r.Cause())
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

// --- Two-level nested error ---

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
		Nested:     inner,
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
		Nested:     inner,
	}

	outerSig, err := outer.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "Error(string)", outerSig)

	innerSig, err := inner.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "AnError(string)", innerSig)
}

func TestRevertErrorNestedTwoLevelCause(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", Nested: inner}

	assert.Equal(t, inner, outer.Cause())
	assert.Nil(t, inner.Cause())
}

func TestRevertErrorNestedTwoLevelInnermost(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", Nested: inner}

	assert.Equal(t, inner, outer.Innermost())
	assert.Equal(t, inner, inner.Innermost())
}

func TestRevertErrorNestedTwoLevelErrors(t *testing.T) {
	innerEntry := &Entry{Type: Error, Name: "AnError", Inputs: ParameterArray{{Name: "message", Type: "string"}}}
	innerCV := decodeTestError(t, innerEntry, `{"message":"I am an error"}`)
	inner := &RevertError{ErrorEntry: innerEntry, cv: innerCV}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "[404]caught bytes", Nested: inner}

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
	outer := &RevertError{ErrorEntry: outerEntry, cv: outerCV, Prefix: "[404]caught bytes", Nested: inner}

	b, err := outer.Innermost().SerializeJSON(context.Background(), nil)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "I am an error")
}

// --- Three-level nested error ---

func TestRevertErrorNestedThreeLevelString(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middleEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	middle := &RevertError{ErrorEntry: middleEntry, Prefix: "middleware: ", Nested: leaf}

	outerEntry := &Entry{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}}
	outer := &RevertError{ErrorEntry: outerEntry, Prefix: "gateway: ", Nested: middle}

	assert.Equal(t, `gateway: middleware: RootCause("the real problem")`, outer.String())
}

func TestRevertErrorNestedThreeLevelInnermost(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "middleware: ", Nested: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "gateway: ", Nested: middle}

	assert.Equal(t, leaf, outer.Innermost())
}

func TestRevertErrorNestedThreeLevelErrors(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "middleware: ", Nested: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Prefix: "gateway: ", Nested: middle}

	errs := outer.Errors()
	require.Len(t, errs, 3)
	assert.Equal(t, outer, errs[0])
	assert.Equal(t, middle, errs[1])
	assert.Equal(t, leaf, errs[2])
}

func TestRevertErrorNestedThreeLevelCauseChain(t *testing.T) {
	leafEntry := &Entry{Type: Error, Name: "RootCause", Inputs: ParameterArray{{Name: "detail", Type: "string"}}}
	leafCV := decodeTestError(t, leafEntry, `{"detail":"the real problem"}`)
	leaf := &RevertError{ErrorEntry: leafEntry, cv: leafCV}

	middle := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Nested: leaf}
	outer := &RevertError{ErrorEntry: &Entry{Type: Error, Name: "Error"}, Nested: middle}

	assert.Equal(t, middle, outer.Cause())
	assert.Equal(t, leaf, outer.Cause().Cause())
	assert.Nil(t, outer.Cause().Cause().Cause())
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
		Nested:     inner,
	}
	assert.Equal(t, `AnError("direct")`, outer.String())
}

func TestRevertErrorSignatureNilEntry(t *testing.T) {
	r := &RevertError{}
	sig, err := r.Signature()
	assert.NoError(t, err)
	assert.Equal(t, "", sig)
}
