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
	"fmt"
	"strconv"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
)

// HexUint64 is a positive integer - serializes to JSON as an 0x hex string (no leading zeros), and parses flexibly depending on the prefix (so 0x for hex, or base 10 for plain string / float64)
type HexUint64 uint64

func (h *HexUint64) String() string {
	if h == nil {
		return "0x0"
	}
	return "0x" + strconv.FormatUint(uint64(*h), 16)
}

func (h HexUint64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.String())), nil
}

func (h *HexUint64) UnmarshalJSON(b []byte) error {
	bi, err := UnmarshalBigInt(context.Background(), b)
	if err != nil {
		return err
	}
	if !bi.IsUint64() {
		return i18n.NewError(context.Background(), signermsgs.MsgInvalidUint64PrecisionLoss, b)
	}
	*h = HexUint64(bi.Uint64())
	return nil
}

func (h HexUint64) Uint64() uint64 {
	return uint64(h)
}

func (h *HexUint64) Uint64OrZero() uint64 {
	if h == nil {
		return 0
	}
	return uint64(*h)
}

func (h *HexUint64) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil
	case int64:
		if src < 0 {
			return i18n.NewError(context.Background(), signermsgs.MsgHexUintNegative, src)
		}
		*h = HexUint64(src)
		return nil
	case uint64:
		*h = HexUint64(src)
		return nil
	default:
		return i18n.NewError(context.Background(), i18n.MsgTypeRestoreFailed, src, h)
	}
}
