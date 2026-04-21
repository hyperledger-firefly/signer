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
	"strings"

	"github.com/hyperledger/firefly-common/pkg/log"
)

// defaultErrorEntries are the built-in Solidity error types that are always
// tried when decoding revert data, even if the caller's ABI is empty.
var defaultErrorEntries = ABI{
	{Type: Error, Name: "Error", Inputs: ParameterArray{{Name: "reason", Type: "string"}}},
	{Type: Error, Name: "Panic", Inputs: ParameterArray{{Name: "code", Type: "uint256"}}},
}

const maxRevertErrorDepth = 10

// RevertError represents a decoded Solidity revert error. For nested errors
// (where a contract catches a revert and re-throws with the original error
// embedded in the string), the InnerError field links to the inner decoded
// error, forming a recursive chain.
type RevertError struct {
	ErrorEntry *Entry          `ffstruct:"RevertError" json:"errorEntry,omitempty"` // the matched ABI error entry at this level
	cv         *ComponentValue // decoded ABI data for this level (unexported)
	Prefix     string          `ffstruct:"RevertError" json:"prefix,omitempty"`     // readable text before the inner error
	InnerError *RevertError    `ffstruct:"RevertError" json:"innerError,omitempty"` // recursively decoded inner error, nil if none
}

// DecodeRevertError decodes raw EVM revert data into a RevertError,
// recursively unwrapping nested errors embedded in Error(string) values.
// Returns nil if the data does not match any known error selector.
func (a ABI) DecodeRevertError(revertData []byte) *RevertError {
	return a.DecodeRevertErrorCtx(context.Background(), revertData)
}

// DecodeRevertErrorCtx decodes raw EVM revert data into a RevertError,
// recursively unwrapping nested errors embedded in Error(string) values.
// The ABI's error entries are tried first, followed by the built-in
// Error(string) and Panic(uint256). Returns nil if no selector matches.
func (a ABI) DecodeRevertErrorCtx(ctx context.Context, revertData []byte) *RevertError {
	for _, source := range []ABI{a.errors(), defaultErrorEntries} {
		for _, e := range source {
			if cv, err := e.DecodeCallDataCtx(ctx, revertData); err == nil {
				r := &RevertError{ErrorEntry: e, cv: cv}
				// Only Error(string) is unwrapped for nesting, because the Solidity
				// catch-and-rethrow pattern (string.concat + string(reason)) always
				// produces Error(string). Custom errors with string/bytes params that
				// also embed error data are not yet handled since there is a high liklihood
				//that they are not intended to carry error data.
				if e.Name == "Error" && len(cv.Children) == 1 {
					if strVal, ok := cv.Children[0].Value.(string); ok {
						r.unwrapInnerError(ctx, a.selectorMap(), strVal, 0)
					}
				}
				return r
			}
		}
	}
	return nil
}

type selectorKey = [4]byte

// selectorMap builds a lookup of 4-byte selectors for all error entries in
// the ABI plus the builtins.
func (a ABI) selectorMap() map[selectorKey]*Entry {
	selectors := make(map[selectorKey]*Entry)
	var key selectorKey
	for _, source := range []ABI{a.errors(), defaultErrorEntries} {
		for _, e := range source {
			sel := e.FunctionSelectorBytes()
			if len(sel) >= 4 {
				copy(key[:], sel[:4])
				if _, exists := selectors[key]; !exists {
					selectors[key] = e
				}
			}
		}
	}
	return selectors
}

// unwrapInnerError scans a decoded string value for an embedded ABI error selector.
// If found, it populates r.Prefix and r.InnerError to form the recursive chain.
func (r *RevertError) unwrapInnerError(ctx context.Context, selectors map[selectorKey]*Entry, s string, depth int) {
	if depth >= maxRevertErrorDepth {
		return
	}

	raw := []byte(s)
	idx, entry := findSelector(raw, selectors)
	if idx < 0 {
		return
	}

	cv, err := entry.DecodeCallDataCtx(ctx, raw[idx:])
	if err != nil {
		log.L(ctx).Debugf("Could not decode inner error at depth %d: %s", depth, err)
		return
	}

	inner := &RevertError{ErrorEntry: entry, cv: cv}
	r.Prefix = SanitizeBinaryString(raw[:idx])
	r.InnerError = inner

	// If the inner error is also Error(string), keep unwrapping
	if entry.Name == "Error" && len(cv.Children) == 1 {
		if strVal, ok := cv.Children[0].Value.(string); ok {
			inner.unwrapInnerError(ctx, selectors, strVal, depth+1)
		}
	}
}

// findSelector scans raw bytes for the first occurrence of a known 4-byte error selector.
func findSelector(raw []byte, selectors map[selectorKey]*Entry) (int, *Entry) {
	if len(raw) < 4 {
		return -1, nil
	}
	var key selectorKey
	for i := 0; i <= len(raw)-4; i++ {
		copy(key[:], raw[i:i+4])
		if e, ok := selectors[key]; ok {
			return i, e
		}
	}
	return -1, nil
}

// errors returns only the Error-type entries from the ABI.
func (a ABI) errors() ABI {
	var out ABI
	for _, e := range a {
		if e.Type == Error {
			out = append(out, e)
		}
	}
	return out
}

// String returns a human-readable representation of the full error chain.
// It concatenates the Prefix at each level, with the leaf error formatted
// as ErrorName(arg1,arg2,...).
func (r *RevertError) String() string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(r.Prefix)
	if r.InnerError != nil {
		b.WriteString(r.InnerError.String())
	} else {
		b.WriteString(FormatErrorStringCtx(context.Background(), r.ErrorEntry, r.cv))
	}
	return b.String()
}

// ErrorString returns the formatted error at this level only, e.g.
// Error("not enough funds") or MyCustomError("0x1234","-100").
// Unlike String(), it does not walk the Cause chain — use it when
// you need the single-level description without recursive unwrapping.
func (r *RevertError) ErrorString() string {
	if r == nil {
		return ""
	}
	return FormatErrorStringCtx(context.Background(), r.ErrorEntry, r.cv)
}

// Signature returns the ABI signature of the error at this level,
// e.g. "Error(string)" or "AnError(string,uint256)".
func (r *RevertError) Signature() (string, error) {
	if r == nil || r.ErrorEntry == nil {
		return "", nil
	}
	return r.ErrorEntry.SignatureCtx(context.Background())
}

// SerializeJSON serializes the decoded error data at this level using
// the provided Serializer.
func (r *RevertError) SerializeJSON(ctx context.Context, s *Serializer) ([]byte, error) {
	if r == nil || r.cv == nil {
		return nil, nil
	}
	if s == nil {
		s = NewSerializer()
	}
	return s.SerializeJSONCtx(ctx, r.cv)
}

// GetInnerError returns the next error in the chain (one level deeper), or nil
// at the leaf.
func (r *RevertError) GetInnerError() *RevertError {
	if r == nil {
		return nil
	}
	return r.InnerError
}

// Innermost walks the chain to return the deepest RevertError — the
// original error that triggered the chain of catch-and-rethrow wrappers.
func (r *RevertError) Innermost() *RevertError {
	if r == nil {
		return nil
	}
	cur := r
	for cur.InnerError != nil {
		cur = cur.InnerError
	}
	return cur
}

// Errors returns a flattened slice of all RevertError entries in the chain,
// from outermost to innermost.
func (r *RevertError) Errors() []*RevertError {
	if r == nil {
		return nil
	}
	var result []*RevertError
	for cur := r; cur != nil; cur = cur.InnerError {
		result = append(result, cur)
	}
	return result
}
