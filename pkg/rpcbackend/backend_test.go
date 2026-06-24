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

package rpcbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-signer/internal/signerconfig"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testRPCHander func(rpcReq *RPCRequest) (int, *RPCResponse)

func newTestServer(t *testing.T, rpcHandler testRPCHander, options ...RPCClientOptions) (context.Context, *RPCClient, func()) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var rpcReq *RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReq)
		assert.NoError(t, err)

		status, rpcRes := rpcHandler(rpcReq)
		b := []byte(`{}`)
		if rpcRes != nil {
			b, err = json.Marshal(rpcRes)
			assert.NoError(t, err)
		}
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.WriteHeader(status)
		w.Write(b)

	}))

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", server.Listener.Addr()))

	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)

	rb := NewRPCClient(c).(*RPCClient)

	return ctx, rb, func() {
		cancelCtx()
		server.Close()
	}
}

func TestSyncRequestOK(t *testing.T) {

	rpcRequestBytes := []byte(`{
		"id": 2,
		"method": "eth_getTransactionByHash",
		"params": [
			"0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1"
		]
	}`)

	rpcServerResponseBytes := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"result": {
			"accessList": [],
			"blockHash": "0x471a236bac44222faf63e3d7808a2a68a704a75ca2f0774f072764867f458268",
			"blockNumber": "0xd536bc",
			"chainId": "0x1",
			"from": "0xfb075bb99f2aa4c49955bf703509a227d7a12248",
			"gas": "0x2b13d",
			"gasPrice": "0x3b6e7f5f09",
			"hash": "0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1",
			"input": "0xa0712d680000000000000000000000000000000000000000000000000000000000000001",
			"maxFeePerGas": "0x4e58be5c3c",
			"maxPriorityFeePerGas": "0x59682f00",
			"nonce": "0x24",
			"r": "0xea6e1513d716146af3a02e1497fbe7fc3b2ffb08ccb4a1bfef4eaa2a122f62df",
			"s": "0xddc23aec20948a55d3e1f8afd29b5570d8d279450a472b55561ef6afe4a07ff",
			"to": "0x3c99f2a4b366d46bcf2277639a135a6d1288eceb",
			"transactionIndex": "0x1d",
			"type": "0x2",
			"v": "0x1",
			"value": "0x8e1bc9bf040000"
		}
	}`)

	var rpcRequest RPCRequest
	err := json.Unmarshal(rpcRequestBytes, &rpcRequest)
	assert.NoError(t, err)

	var rpcServerResponse RPCResponse
	err = json.Unmarshal(rpcServerResponseBytes, &rpcServerResponse)
	assert.NoError(t, err)

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionByHash", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1"`, rpcReq.Params[0].String())
		rpcServerResponse.ID = rpcReq.ID
		return 200, &rpcServerResponse
	})
	rb.requestCounter = 12345
	defer done()

	rpcRes, err := rb.SyncRequest(ctx, &rpcRequest)
	assert.NoError(t, err)
	assert.Equal(t, `2`, rpcRes.ID.String())
	assert.Equal(t, `0x24`, rpcRes.Result.JSONObject().GetString(`nonce`))
}

func TestSyncRPCCallOK(t *testing.T) {

	logrus.SetLevel(logrus.TraceLevel)

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionCount", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0xfb075bb99f2aa4c49955bf703509a227d7a12248"`, rpcReq.Params[0].String())
		assert.Equal(t, `"pending"`, rpcReq.Params[1].String())
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  fftypes.JSONAnyPtr(`"0x26"`),
		}
	})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Empty(t, err)
	assert.Equal(t, int64(0x26), txCount.BigInt().Int64())
}

func TestSyncRPCCallNullResponse(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		assert.Equal(t, "2.0", rpcReq.JSONRpc)
		assert.Equal(t, "eth_getTransactionReceipt", rpcReq.Method)
		assert.Equal(t, `"000012346"`, rpcReq.ID.String())
		assert.Equal(t, `"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`, rpcReq.Params[0].String())
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  nil,
		}
	})
	rb.requestCounter = 12345
	defer done()

	rpcRes, err := rb.SyncRequest(ctx, &RPCRequest{
		ID:     fftypes.JSONAnyPtr("1"),
		Method: "eth_getTransactionReceipt",
		Params: []*fftypes.JSONAny{
			fftypes.JSONAnyPtr(`"0xf44d5387087f61237bdb5132e9cf0f38ab20437128f7291b8df595305a1a8284"`),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, `null`, rpcRes.Result.String())
}

func TestSyncRPCCallErrorResponse(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 500, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Error: &RPCError{
				Message: "pop",
			},
		}
	})
	rb.requestCounter = 12345
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "pop", err)
}

func TestSyncRPCCallBadJSONResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{!!!!`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClient(c).(*RPCClient)

	var txCount ethtypes.HexInteger
	rpcErr := rb.CallRPC(context.Background(), &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22012", rpcErr.Error())
}

func TestSyncRPCCallFailParseJSONResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"not an object"}`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(context.Background(), signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClient(c).(*RPCClient)

	var mapResult map[string]interface{}
	rpcErr := rb.CallRPC(context.Background(), &mapResult, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Regexp(t, "FF22065", rpcErr.Error())
}

func TestSyncRPCCallErrorBadInput(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	defer done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "test-bad-params", map[bool]bool{false: true})
	assert.Regexp(t, "FF22011", err)
}

func TestSyncRPCCallServerDown(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	done()

	var txCount ethtypes.HexInteger
	err := rb.CallRPC(ctx, &txCount, "net_version")
	assert.Regexp(t, "FF22012", err)
}

func TestSafeMessageGetter(t *testing.T) {

	assert.Empty(t, (&RPCResponse{}).Message())
}

func TestRPCErrorResponse(t *testing.T) {

	id := fftypes.JSONAnyPtr(`"test-id"`)
	res := RPCErrorResponse(fmt.Errorf("something went wrong"), id, RPCCodeInternalError)
	assert.Equal(t, "2.0", res.JSONRpc)
	assert.Equal(t, id, res.ID)
	assert.Equal(t, int64(RPCCodeInternalError), res.Error.Code)
	assert.Equal(t, "something went wrong", res.Error.Message)
}

type testBatchRPCHandler func(rpcReqs []*RPCRequest) (int, []*RPCResponse)

func newBatchTestServer(t *testing.T, rpcHandler testBatchRPCHandler) (context.Context, *RPCClient, func()) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReqs []*RPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReqs)
		assert.NoError(t, err)

		status, rpcResps := rpcHandler(rpcReqs)
		b, err := json.Marshal(rpcResps)
		assert.NoError(t, err)
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.WriteHeader(status)
		w.Write(b)
	}))

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", server.Listener.Addr()))
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClient(c).(*RPCClient)

	return ctx, rb, func() {
		cancelCtx()
		server.Close()
	}
}

func TestBatchRPCCallOK(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		assert.Len(t, rpcReqs, 2)
		assert.Equal(t, "eth_blockNumber", rpcReqs[0].Method)
		assert.Equal(t, "eth_chainId", rpcReqs[1].Method)
		// Return out of order to verify ID-based matching
		return 200, []*RPCResponse{
			{JSONRpc: "2.0", ID: rpcReqs[1].ID, Result: fftypes.JSONAnyPtr(`"0x1"`)},
			{JSONRpc: "2.0", ID: rpcReqs[0].ID, Result: fftypes.JSONAnyPtr(`"0x100"`)},
		}
	})
	defer done()

	var blockNumber ethtypes.HexInteger
	var chainID ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx,
		&RPCBatchOp{Result: &blockNumber, Method: "eth_blockNumber"},
		&RPCBatchOp{Result: &chainID, Method: "eth_chainId"},
	)
	assert.Nil(t, errs[0])
	assert.Nil(t, errs[1])
	assert.Equal(t, int64(0x100), blockNumber.BigInt().Int64())
	assert.Equal(t, int64(0x1), chainID.BigInt().Int64())
}

func TestBatchRPCCallPartialError(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 200, []*RPCResponse{
			{JSONRpc: "2.0", ID: rpcReqs[0].ID, Result: fftypes.JSONAnyPtr(`"0x26"`)},
			{JSONRpc: "2.0", ID: rpcReqs[1].ID, Error: &RPCError{Code: -32000, Message: "not found"}},
		}
	})
	defer done()

	var txCount ethtypes.HexInteger
	var blockResult ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx,
		&RPCBatchOp{Result: &txCount, Method: "eth_getTransactionCount", Params: []interface{}{"0xaddr", "pending"}},
		&RPCBatchOp{Result: &blockResult, Method: "eth_getBlockByHash", Params: []interface{}{"0xbad", false}},
	)
	assert.Nil(t, errs[0])
	assert.Regexp(t, "not found", errs[1])
	assert.Equal(t, int64(0x26), txCount.BigInt().Int64())
}

func TestBatchRPCCallBadInput(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 500, nil
	})
	defer done()

	var result ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx,
		&RPCBatchOp{Result: &result, Method: "eth_test", Params: []interface{}{map[bool]bool{false: true}}},
	)
	assert.Regexp(t, "FF22011", errs[0])
}

func TestBatchRPCCallServerDown(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 500, nil
	})
	done()

	var result ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx,
		&RPCBatchOp{Result: &result, Method: "eth_blockNumber"},
	)
	assert.Regexp(t, "FF22012", errs[0])
}

func TestBatchRPCCallConcurrencyCancel(t *testing.T) {
	blocked := make(chan struct{})
	blocking := make(chan bool, 1)
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blocking <- true
		<-blocked
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	signerconfig.Reset()
	signerconfig.BackendConfig.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{MaxConcurrentRequest: 1}).(*RPCClient)

	bgDone := make(chan struct{})
	go func() {
		_, err := rb.SyncRequest(ctx, &RPCRequest{})
		assert.Regexp(t, "FF22012", err)
		close(bgDone)
	}()
	<-blocking

	cancelCtx()
	var result ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx, &RPCBatchOp{Result: &result, Method: "eth_blockNumber"})
	assert.Regexp(t, "FF22063", errs[0])

	close(blocked)
	<-bgDone
}

func TestBatchRPCCallHTTPError(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 500, nil
	})
	defer done()

	var result ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx, &RPCBatchOp{Result: &result, Method: "eth_blockNumber"})
	assert.Regexp(t, "FF22012", errs[0])
}

func TestBatchRPCCallNoResponse(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 200, []*RPCResponse{} // empty — no op matched
	})
	defer done()

	var result ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx, &RPCBatchOp{Result: &result, Method: "eth_blockNumber"})
	assert.Regexp(t, "FF22093", errs[0])
}

func TestBatchRPCCallNullResult(t *testing.T) {

	// Server returns null result — matchBatchResponse assigns nil to the intermediate,
	// CallRPCBatch converts nil → fftypes.NullString before the final json.Unmarshal.
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 200, []*RPCResponse{
			{JSONRpc: "2.0", ID: rpcReqs[0].ID, Result: nil},
		}
	})
	defer done()

	var result *ethtypes.HexInteger
	errs := rb.CallRPCBatch(ctx, &RPCBatchOp{Result: &result, Method: "eth_getTransactionReceipt"})
	assert.Nil(t, errs[0])
	assert.Nil(t, result)
}

func TestMatchBatchResponseNilResponse(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	matchBatchResponse[any](nil, map[string]int{}, nil, errs, matched)
	assert.Nil(t, errs[0])
	assert.False(t, matched[0])
}

func TestMatchBatchResponseNilID(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	matchBatchResponse(&RPCResponse{}, map[string]int{}, nil, errs, matched)
	assert.Nil(t, errs[0])
	assert.False(t, matched[0])
}

func TestMatchBatchResponseNonStringID(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	rpcRes := &RPCResponse{ID: fftypes.JSONAnyPtr("123")} // JSON number — unmarshal into string fails
	matchBatchResponse(rpcRes, map[string]int{"123": 0}, nil, errs, matched)
	assert.Nil(t, errs[0])
	assert.False(t, matched[0])
}

func TestMatchBatchResponseUnknownID(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	rpcRes := &RPCResponse{ID: fftypes.JSONAnyPtr(`"unknown"`)}
	matchBatchResponse(rpcRes, map[string]int{"other": 0}, nil, errs, matched)
	assert.Nil(t, errs[0])
	assert.False(t, matched[0])
}

func TestCallRPCTypedOK(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Result:  fftypes.JSONAnyPtr(`"0x26"`),
		}
	})
	defer done()

	var txCount ethtypes.HexInteger
	rpcErr := CallRPCTyped(ctx, rb, &txCount, "eth_getTransactionCount", ethtypes.MustNewAddress("0xfb075bb99f2aa4c49955bf703509a227d7a12248"), "pending")
	assert.Nil(t, rpcErr)
	assert.Equal(t, int64(0x26), txCount.BigInt().Int64())
}

func TestCallRPCTypedRPCError(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 200, &RPCResponse{
			JSONRpc: "2.0",
			ID:      rpcReq.ID,
			Error:   &RPCError{Code: -32000, Message: "not found"},
		}
	})
	defer done()

	var txCount ethtypes.HexInteger
	rpcErr := CallRPCTyped(ctx, rb, &txCount, "eth_getTransactionByHash", "0x1234")
	assert.Regexp(t, "not found", rpcErr.Error())
	assert.Equal(t, int64(-32000), rpcErr.Code)
}

func TestCallRPCTypedBadInput(t *testing.T) {

	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) { return 500, nil })
	defer done()

	var txCount ethtypes.HexInteger
	rpcErr := CallRPCTyped(ctx, rb, &txCount, "test-bad-params", map[bool]bool{false: true})
	assert.Regexp(t, "FF22011", rpcErr.Error())
}

func TestCallRPCTypedHTTPErrorNoRPCBody(t *testing.T) {

	// 500 with no RPCResponse body — syncRequestTyped returns err but typed.Error is nil,
	// so CallRPCTyped must fall through to the generic error path.
	ctx, rb, done := newTestServer(t, func(rpcReq *RPCRequest) (status int, rpcRes *RPCResponse) {
		return 500, nil
	})
	defer done()

	var txCount ethtypes.HexInteger
	rpcErr := CallRPCTyped(ctx, rb, &txCount, "eth_blockNumber")
	assert.Regexp(t, "FF22012", rpcErr.Error())
}

func TestMatchBatchResponseRPCError(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	result := new(ethtypes.HexInteger)
	ops := []*RPCBatchOpTyped[ethtypes.HexInteger]{{Result: result, Method: "eth_test"}}
	rpcRes := &RPCResponseTyped[ethtypes.HexInteger]{
		ID:    fftypes.JSONAnyPtr(`"000000001"`),
		Error: &RPCError{Code: -32000, Message: "oops"},
	}
	matchBatchResponse[ethtypes.HexInteger](rpcRes, map[string]int{"000000001": 0}, ops, errs, matched)
	assert.True(t, matched[0])
	assert.Regexp(t, "oops", errs[0])
}

func TestMatchBatchResponseNullResult(t *testing.T) {
	errs := make([]*RPCError, 1)
	matched := make([]bool, 1)
	var rawResult *fftypes.JSONAny
	ops := []*RPCBatchOpTyped[*fftypes.JSONAny]{{Result: &rawResult, Method: "eth_test"}}
	rpcRes := &RPCResponse{
		ID:     fftypes.JSONAnyPtr(`"000000001"`),
		Result: nil,
	}
	matchBatchResponse(rpcRes, map[string]int{"000000001": 0}, ops, errs, matched)
	assert.True(t, matched[0])
	assert.Nil(t, errs[0])
}

func TestBatchRPCCallUnmarshalFail(t *testing.T) {
	ctx, rb, done := newBatchTestServer(t, func(rpcReqs []*RPCRequest) (int, []*RPCResponse) {
		return 200, []*RPCResponse{
			{JSONRpc: "2.0", ID: rpcReqs[0].ID, Result: fftypes.JSONAnyPtr(`"not-an-object"`)},
		}
	})
	defer done()

	var mapResult map[string]interface{}
	errs := rb.CallRPCBatch(ctx, &RPCBatchOp{Result: &mapResult, Method: "eth_test"})
	assert.Regexp(t, "FF22065", errs[0])
}

func TestSyncRequestConcurrency(t *testing.T) {

	blocked := make(chan struct{})
	blocking := make(chan bool, 1)
	ctx, cancelCtx := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blocking <- true
		<-blocked
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	signerconfig.Reset()
	prefix := signerconfig.BackendConfig
	prefix.Set(ffresty.HTTPConfigURL, server.URL)
	c, err := ffresty.New(ctx, signerconfig.BackendConfig)
	assert.NoError(t, err)
	rb := NewRPCClientWithOption(c, RPCClientOptions{
		MaxConcurrentRequest: 1,
	}).(*RPCClient)

	bgDone := make(chan struct{})
	go func() {
		_, err := rb.SyncRequest(ctx, &RPCRequest{})
		assert.Regexp(t, "FF22012", err)
		close(bgDone)
	}()
	<-blocking

	cancelCtx()
	_, err = rb.SyncRequest(ctx, &RPCRequest{})
	assert.Regexp(t, "FF22063", err)

	close(blocked)
	<-bgDone

}
