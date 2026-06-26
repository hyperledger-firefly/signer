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
	"bytes"
	"context"
	"encoding/json"
	"math/big"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
)

func BigIntegerFromString(ctx context.Context, s string) (*big.Int, error) {
	// We use Go's default '0' base integer parsing, where `0x` means hex,
	// no prefix means decimal etc.
	i, ok := new(big.Int).SetString(s, 0)
	if !ok {
		// Fallback for decimal floats and scientific notation (e.g. "12345.0", "1e10").
		// big.Rat gives exact rational arithmetic — no precision limit or rounding mode.
		// Guard length before SetString to prevent unbounded memory use (CVE-2022-23772).
		// uint256 max is 78 decimal digits; float notation adds at most a sign, decimal
		// point, and exponent ("e+77"), so any valid value fits within 100 characters.
		if len(s) > 100 {
			log.L(ctx).Errorf("Error parsing numeric string '%s'", s)
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidNumberString, s)
		}
		r, ok := new(big.Rat).SetString(s) //nolint:gosec // G113: input bounded to 100 chars by the guard above
		if !ok {
			log.L(ctx).Errorf("Error parsing numeric string '%s'", s)
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidNumberString, s)
		}
		if !r.IsInt() {
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidIntPrecisionLoss, s)
		}
		return r.Num(), nil
	}
	return i, nil
}

func UnmarshalBigInt(ctx context.Context, b []byte) (*big.Int, error) {
	var i interface{}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	err := d.Decode(&i)
	if err != nil {
		return nil, err
	}
	switch i := i.(type) {
	case json.Number:
		return BigIntegerFromString(context.Background(), i.String())
	case string:
		return BigIntegerFromString(context.Background(), i)
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidJSONTypeForBigInt, i)
	}
}
