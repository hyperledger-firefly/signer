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

package ethtypes

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegerParsing(t *testing.T) {
	ctx := context.Background()

	i, err := BigIntegerFromString(ctx, "1.0000000000000000000000001e+25")
	assert.NoError(t, err)
	assert.Equal(t, "10000000000000000000000001", i.String())

	i, err = BigIntegerFromString(ctx, "10000000000000000000000000000001")
	assert.NoError(t, err)
	assert.Equal(t, "10000000000000000000000000000001", i.String())

	i, err = BigIntegerFromString(ctx, "20000000000000000000000000000002")
	assert.NoError(t, err)
	assert.Equal(t, "20000000000000000000000000000002", i.String())

	_, err = BigIntegerFromString(ctx, "0xGG")
	assert.Regexp(t, "FF22088", err)

	_, err = BigIntegerFromString(ctx, "3.0000000000000000000000000000003")
	assert.Regexp(t, "FF22089", err)

	// big.Rat handles e-notation exactly — "1e10" is 10^10, no float rounding
	i, err = BigIntegerFromString(ctx, "1e10")
	assert.NoError(t, err)
	assert.Equal(t, "10000000000", i.String())

	// Non-integer rational is a precision-loss error
	_, err = BigIntegerFromString(ctx, "1.5")
	assert.Regexp(t, "FF22089", err)

	// String exceeding 100 chars is rejected before Rat.SetString (CVE-2022-23772 guard)
	_, err = BigIntegerFromString(ctx, "1."+strings.Repeat("0", 99)+"1e+100")
	assert.Regexp(t, "FF22088", err)
}
