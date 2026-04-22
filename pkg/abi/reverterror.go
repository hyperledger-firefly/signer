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

// maxInnerErrorScanBytes caps how far into a string value we search for an
// embedded ABI selector. Revert strings are measured in tens of bytes in
// practice, so 1024 is already far beyond any realistic prefix length.
const maxInnerErrorScanBytes = 1024

// minABIEncodedLen is the minimum byte length for a decodable ABI-encoded
// error: 4 bytes for the selector plus at least one 32-byte word for the
// first parameter. A candidate selector with fewer bytes remaining is
// guaranteed to fail decoding, so it is skipped.
const minABIEncodedLen = 4 + 32

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

// DecodeRevertError decodes raw EVM revert data into a RevertError.
// Pass ErrorFormatOption{SearchForWrappedBinaryErrors: true} to scan for inner errors
// binary-encoded within an Error(string) value.
// Returns nil if the data does not match any known error selector.
func (a ABI) DecodeRevertError(revertData []byte, options ...ErrorFormatOption) *RevertError {
	return a.DecodeRevertErrorCtx(context.Background(), revertData, options...)
}

// DecodeRevertErrorCtx decodes raw EVM revert data into a RevertError.
// The ABI's error entries are tried first, followed by the built-in
// Error(string) and Panic(uint256). Returns nil if no selector matches.
// Pass ErrorFormatOption{SearchForWrappedBinaryErrors: true} to scan for inner errors
// binary-encoded within an Error(string) value.
func (a ABI) DecodeRevertErrorCtx(ctx context.Context, revertData []byte, options ...ErrorFormatOption) *RevertError {
	searchBinary := false
	for _, o := range options {
		searchBinary = searchBinary || o.SearchForWrappedBinaryErrors
	}
	abiErrors := a.FilterType(Error)
	for _, source := range []ABI{abiErrors, defaultErrorEntries} {
		for _, e := range source {
			if cv, err := e.DecodeCallDataCtx(ctx, revertData); err == nil {
				r := &RevertError{ErrorEntry: e, cv: cv}
				// Only Error(string) is scanned for binary-wrapped inner errors.
				// Custom errors with string/bytes params are not scanned since there
				// is a high likelihood that they are not intended to carry error data.
				if searchBinary && e.Name == "Error" && len(cv.Children) == 1 {
					if strVal, ok := cv.Children[0].Value.(string); ok {
						// Build a selector map covering both the caller's error entries
						// and the built-in defaults, so inner errors of either kind can
						// be recognised during recursive unwrapping.
						selectors := append(abiErrors, defaultErrorEntries...).SelectorMap()
						r.scanForBinaryWrappedError(ctx, selectors, strVal, 0)
					}
				}
				return r
			}
		}
	}
	return nil
}

// scanForBinaryWrappedError scans a decoded string value for an embedded ABI error selector.
// If found, it populates r.Prefix and r.InnerError to form the recursive chain.
func (r *RevertError) scanForBinaryWrappedError(ctx context.Context, selectors map[[4]byte]*Entry, s string, depth int) {
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

	// If the inner error is also Error(string), keep scanning recursively
	if entry.Name == "Error" && len(cv.Children) == 1 {
		if strVal, ok := cv.Children[0].Value.(string); ok {
			inner.scanForBinaryWrappedError(ctx, selectors, strVal, depth+1)
		}
	}
}

// findSelector scans raw bytes for the first occurrence of a known 4-byte
// error selector. Two constraints are folded into the loop bound:
//   - We stop scanning after maxInnerErrorScanBytes, bounding performance on
//     large payloads (revert strings are tiny in practice).
//   - We only consider positions where at least minABIEncodedLen bytes remain,
//     since a selector with fewer bytes after it cannot decode successfully.
func findSelector(raw []byte, selectors map[[4]byte]*Entry) (int, *Entry) {
	scanLimit := len(raw)
	if scanLimit > maxInnerErrorScanBytes {
		scanLimit = maxInnerErrorScanBytes
	}
	// limit is the highest start index that satisfies both constraints.
	limit := scanLimit - 4
	if remainingLimit := len(raw) - minABIEncodedLen; remainingLimit < limit {
		limit = remainingLimit
	}
	var key [4]byte
	for i := 0; i <= limit; i++ {
		copy(key[:], raw[i:i+4])
		if e, ok := selectors[key]; ok {
			return i, e
		}
	}
	return -1, nil
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
// Unlike String(), it does not walk the chain — use it when
// you need the single-level description.
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
// innermost binary-wrapped error.
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
