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

package ethsigner

import (
	"context"

	"github.com/hyperledger-firefly/signer/pkg/eip712"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
)

// Wallet is the common interface can be implemented across wallet/signing capabilities
type Wallet interface {
	Sign(ctx context.Context, txn *Transaction, chainID int64) ([]byte, error)
	// SignPrivateTxn(ctx context.Context, addr ethtypes.Address, ptx *Transaction, chainID int64) ([]byte, error)
	Initialize(ctx context.Context) error
	GetAccounts(ctx context.Context) ([]*ethtypes.Address0xHex /* no checksum on returned values */, error)
	Refresh(ctx context.Context) error
	Close() error
}

type WalletTypedData interface {
	Wallet
	SignTypedDataV4(ctx context.Context, from ethtypes.Address0xHex, payload *eip712.TypedData) (*EIP712Result, error)
}
