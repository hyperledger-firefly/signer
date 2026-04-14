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
)

// RevertError represents a decoded Solidity revert error. For nested errors
// (where a contract catches a revert and re-throws with the original error
// embedded in the string), the Nested field links to the inner decoded error,
// forming a recursive chain.
type RevertError struct {
	ErrorEntry *Entry         // the matched ABI error entry at this level
	cv         *ComponentValue // decoded ABI data for this level
	Prefix     string         // readable text before a nested error
	Nested     *RevertError   // recursively decoded inner error, nil if none
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
	if r.Nested != nil {
		b.WriteString(r.Nested.String())
	} else {
		b.WriteString(FormatErrorStringCtx(context.Background(), r.ErrorEntry, r.cv))
	}
	return b.String()
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
// the provided Serializer. This is most useful on the Innermost() error
// where the data is cleanly structured.
func (r *RevertError) SerializeJSON(ctx context.Context, s *Serializer) ([]byte, error) {
	if r == nil || r.cv == nil {
		return nil, nil
	}
	if s == nil {
		s = NewSerializer()
	}
	return s.SerializeJSONCtx(ctx, r.cv)
}

// Cause returns the next error in the chain (one level deeper), or nil
// at the leaf. Analogous to Java's Throwable.getCause().
func (r *RevertError) Cause() *RevertError {
	if r == nil {
		return nil
	}
	return r.Nested
}

// Innermost walks the chain to return the deepest RevertError — the
// original error that triggered the chain of catch-and-rethrow wrappers.
func (r *RevertError) Innermost() *RevertError {
	if r == nil {
		return nil
	}
	cur := r
	for cur.Nested != nil {
		cur = cur.Nested
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
	for cur := r; cur != nil; cur = cur.Nested {
		result = append(result, cur)
	}
	return result
}
