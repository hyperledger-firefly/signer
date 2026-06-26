// Copyright © 2022 Kaleido, Inc.
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

package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/ethsigner"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/rpcbackend"
)

func (s *rpcServer) processRPC(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {
	if rpcReq.ID == nil {
		err := i18n.NewError(ctx, signermsgs.MsgMissingRequestID)
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInvalidRequest), err
	}

	switch rpcReq.Method {
	case "eth_accounts", "personal_accounts":
		return s.processEthAccounts(ctx, rpcReq)
	case "eth_sendTransaction":
		return s.processEthSendTransaction(ctx, rpcReq)
	default:
		return s.backend.SyncRequest(ctx, rpcReq)
	}
}

func (s *rpcServer) processEthAccounts(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {
	accounts, err := s.wallet.GetAccounts(ctx)
	if err != nil {
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInternalError), err
	}
	b, _ := json.Marshal(&accounts)
	return &rpcbackend.RPCResponse{
		JSONRpc: "2.0",
		ID:      rpcReq.ID,
		Result:  fftypes.JSONAnyPtrBytes(b),
	}, nil
}

func (s *rpcServer) processEthSendTransaction(ctx context.Context, rpcReq *rpcbackend.RPCRequest) (*rpcbackend.RPCResponse, error) {

	if len(rpcReq.Params) < 1 {
		err := i18n.NewError(ctx, signermsgs.MsgInvalidParamCount, 1, len(rpcReq.Params))
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInvalidRequest), err
	}

	var txn ethsigner.Transaction
	err := json.Unmarshal(rpcReq.Params[0].Bytes(), &txn)
	if err != nil {
		err := i18n.WrapError(ctx, err, signermsgs.MsgInvalidTransaction)
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeParseError), err
	}

	if txn.From == nil {
		err := i18n.NewError(ctx, signermsgs.MsgMissingFrom)
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInvalidRequest), err
	}

	// We have trivial nonce management built-in for sequential signing API calls, by making a JSON/RPC request
	// to the up-stream node. This should not be relied upon for production use cases.
	// See FireFly Transaction Manager, or FireFly EthConnect, for more advanced nonce management capabilities.
	if txn.Nonce == nil {
		var from ethtypes.Address0xHex
		err := json.Unmarshal(txn.From, &from)
		if err != nil {
			return nil, err
		}
		rpcErr := s.backend.CallRPC(ctx, &txn.Nonce, "eth_getTransactionCount", &from, "pending")
		if rpcErr != nil {
			return rpcbackend.RPCErrorResponse(rpcErr.Error(), rpcReq.ID, rpcbackend.RPCCodeInternalError), rpcErr.Error()
		}
	}

	// Sign the transaction
	var hexData ethtypes.HexBytes0xPrefix
	hexData, err = s.wallet.Sign(ctx, &txn, s.chainID)
	if err != nil {
		return rpcbackend.RPCErrorResponse(err, rpcReq.ID, rpcbackend.RPCCodeInternalError), err
	}

	// Progress with the original request, now updated with a raw transaction fully signed
	rpcReq.Method = "eth_sendRawTransaction"
	rpcReq.Params = []*fftypes.JSONAny{fftypes.JSONAnyPtr(fmt.Sprintf(`"%s"`, hexData))}
	return s.backend.SyncRequest(ctx, rpcReq)

}
