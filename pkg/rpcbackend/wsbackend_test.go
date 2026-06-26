// Copyright © 2023 Kaleido, Inc.
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
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/common/pkg/wsclient"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func generateConfig() *wsclient.WSConfig {
	return &wsclient.WSConfig{}
}

func newTestWSRPC(t *testing.T) (context.Context, *wsRPCClient, chan string, chan string, func()) {
	logrus.SetLevel(logrus.TraceLevel)

	toServer, fromServer, url, close := wsclient.NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test", req.URL.Path)
	})

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond
	wsConfig.InitialConnectAttempts = 2
	wsConfig.DisableReconnect = true

	rc := NewWSRPCClient(wsConfig)
	ctx, cancelCtx := context.WithCancel(context.Background())
	return ctx, rc.(*wsRPCClient), toServer, fromServer, func() {
		close()
		rc.Close()
		cancelCtx()
	}
}

func TestWSRPCConnect(t *testing.T) {
	_, rc, _, _, done := newTestWSRPC(t)
	defer done()

	err := rc.Connect(context.Background())
	assert.NoError(t, err)
}

func TestWSRPCConfError(t *testing.T) {
	// Init clean config
	wsConfig := generateConfig()
	wsConfig.HTTPURL = "!!!!:::"

	wsRPCClient := NewWSRPCClient(wsConfig)

	err := wsRPCClient.Connect(context.Background())
	assert.Regexp(t, "FF00149", err)
}

func TestWSRPCConnectError(t *testing.T) {
	// Init clean config
	wsConfig := generateConfig()

	wsRPCClient := NewWSRPCClient(wsConfig)

	err := wsRPCClient.Connect(context.Background())
	assert.Regexp(t, "FF00148", err)
}

func TestWSRPCSubscribe(t *testing.T) {
	ctx, rc, toServer, fromServer, done := newTestWSRPC(t)
	defer done()

	err := rc.Connect(ctx)
	assert.NoError(t, err)

	go func() {
		msg := <-toServer
		assert.JSONEq(t, `{"jsonrpc":"2.0","id":"000000001","method":"eth_subscribe","params":["newHeads"]}`, msg)

		// Test error cases first to make sure client ignores stuff it doesn't care about
		// should log: WARN: Received subscription event for untracked subscription
		fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x99999999999999999999999999999999"}}`
		// should log: ERROR: Unable to process received message
		fromServer <- `{"nonsense": true}`
		// should log a deserialization error
		fromServer <- `notjson`

		// Then test real subscription message
		fromServer <- `{"jsonrpc":"2.0","id":"000000001","result":"0x9ce59a13059e417087c02d3236a0b1cc"}`
	}()

	s, rpcErr := rc.Subscribe(ctx, "newHeads")
	assert.Nil(t, rpcErr)
	assert.NotEmpty(t, s, s.LocalID())

	assert.Len(t, rc.Subscriptions(), 1)

	fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x9ce59a13059e417087c02d3236a0b1cc"}}`

	newHead := <-s.Notifications()
	assert.NotNil(t, newHead)

	blockNumber := ethtypes.NewHexInteger(newHead.Result.JSONObject().GetInteger("number"))
	assert.Equal(t, big.NewInt(1263817), blockNumber.BigInt())

	hash := newHead.Result.JSONObject().GetString("hash")
	assert.Equal(t, "0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378", hash)

	go func() {
		msg := <-toServer
		assert.JSONEq(t, `{"jsonrpc":"2.0","id":"000000002","method":"eth_unsubscribe","params":["0x9ce59a13059e417087c02d3236a0b1cc"]}`, msg)
		fromServer <- `{"jsonrpc":"2.0","id":"000000002","result":true}`
	}()

	rpcErr = s.Unsubscribe(ctx)
	assert.Nil(t, rpcErr)
	assert.Empty(t, rc.pendingSubsByReqID)
	assert.Empty(t, rc.activeSubsBySubID)
	assert.Empty(t, rc.configuredSubs)

	res, ok := <-s.Notifications()
	assert.Nil(t, res)
	assert.False(t, ok)
}

func TestWSRPCSubscribeError(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)

	err := rc.Connect(context.Background())
	assert.NoError(t, err)

	done()
	_, rpcErr := rc.Subscribe(ctx, []bool{false})
	assert.Regexp(t, "FF22012|FF22063", rpcErr.Error())
}

func TestWSRPCSubscribeClose(t *testing.T) {
	ctx, rc, toServer, _, done := newTestWSRPC(t)

	err := rc.Connect(context.Background())
	assert.NoError(t, err)

	go func() {
		<-toServer
		done()
	}()

	_, rpcErr := rc.Subscribe(ctx, []bool{false})
	assert.Regexp(t, "FF22063", rpcErr.Error())
}

func TestWSRPCCallRPCError(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	err := rc.Connect(ctx)
	assert.NoError(t, err)

	bad := map[bool]bool{false: true}
	rpcErr := rc.CallRPC(ctx, nil, "eth_call", bad)
	assert.Error(t, rpcErr.Error())
}

func TestWSRPCSubscribeRPCError(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	err := rc.Connect(ctx)
	assert.NoError(t, err)

	bad := map[bool]bool{false: true}
	_, rpcErr := rc.Subscribe(ctx, bad)
	assert.Error(t, rpcErr.Error())
}

func TestWSRPCUnsubscribeError(t *testing.T) {
	ctx, rc, toServer, fromServer, done := newTestWSRPC(t)

	err := rc.Connect(context.Background())
	assert.NoError(t, err)

	go func() {
		msg := <-toServer
		assert.Equal(t, `{"jsonrpc":"2.0","id":"000000001","method":"eth_subscribe","params":["newHeads"]}`, msg)
		fromServer <- `{"jsonrpc":"2.0","id":"000000001","result":"0x9ce59a13059e417087c02d3236a0b1cc"}`
		fromServer <- `{"jsonrpc":"2.0","method":"eth_subscription","params":{"result":{"extraData":"0xd983010305844765746887676f312e342e328777696e646f7773","gasLimit":"0x47e7c4","gasUsed":"0x38658","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","nonce":"0x084149998194cc5f","number":"0x1348c9","parentHash":"0x7736fab79e05dc611604d22470dadad26f56fe494421b5b333de816ce1f25701","receiptRoot":"0x2fab35823ad00c7bb388595cb46652fe7886e00660a01e867824d3dceb1c8d36","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378","timestamp":"0x56ffeff8","transactionsRoot":"0x0167ffa60e3ebc0b080cdb95f7c0087dd6c0e61413140e39d94d3468d7c9689f","hash":"0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378"},"subscription":"0x9ce59a13059e417087c02d3236a0b1cc"}}`

		msg = <-toServer
		assert.Equal(t, `{"jsonrpc":"2.0","id":"000000002","method":"eth_unsubscribe","params":["0x9ce59a13059e417087c02d3236a0b1cc"]}`, msg)
		fromServer <- `{"jsonrpc":"2.0","id":"000000002","result":false}`

	}()

	s, rpcErr := rc.Subscribe(ctx, "newHeads")
	assert.Nil(t, rpcErr)

	newHead := <-s.Notifications()
	assert.NotNil(t, newHead)

	blockNumber := ethtypes.NewHexInteger(newHead.Result.JSONObject().GetInteger("number"))
	assert.Equal(t, big.NewInt(1263817), blockNumber.BigInt())

	hash := newHead.Result.JSONObject().GetString("hash")
	assert.Equal(t, "0xb3346685172db67de536d8765c43c31009d0eb3bd9c501c9be3229203f15f378", hash)

	done()
	rpcErr = rc.UnsubscribeAll(ctx)
	assert.Regexp(t, "FF22012|FF22063", rpcErr.Error())
}

func TestCallRPC(t *testing.T) {
	ctx, rc, toServer, fromServer, done := newTestWSRPC(t)

	err := rc.Connect(context.Background())
	assert.NoError(t, err)

	go func() {
		msg := <-toServer
		assert.JSONEq(t, `{"jsonrpc":"2.0","id":"000000001","method":"net_version"}`, msg)
		fromServer <- `{"jsonrpc":"2.0","id":"000000001","result":"0x123456"}`
	}()

	var verResult ethtypes.HexInteger
	rpcErr := rc.CallRPC(ctx, &verResult, "net_version")
	assert.Nil(t, rpcErr)
	assert.Equal(t, int64(0x123456), verResult.Int64())

	done()
}

func TestWaitResponseClosedContext(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)

	done()
	rpcErr := rc.waitResponse(ctx, nil, "000000001", &RPCRequest{}, time.Now(), make(chan *RPCResponse))
	assert.Regexp(t, "FF22063", rpcErr.Error())
}

func TestWaitResponseErrorCode(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	resChl := make(chan *RPCResponse)
	go func() {
		resChl <- &RPCResponse{
			Error: &RPCError{
				Code:    int64(RPCCodeInternalError),
				Message: "pop",
			},
		}
	}()

	rpcErr := rc.waitResponse(ctx, nil, "000000001", &RPCRequest{}, time.Now(), resChl)
	assert.Regexp(t, "pop", rpcErr.Error())
}

func TestWaitResponseNilNilOk(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	resChl := make(chan *RPCResponse)
	go func() {
		resChl <- &RPCResponse{}
	}()

	rpcErr := rc.waitResponse(ctx, nil, "000000001", &RPCRequest{}, time.Now(), resChl)
	assert.Nil(t, rpcErr)
}

func TestWaitResponseBadUnmarshal(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	resChl := make(chan *RPCResponse)
	go func() {
		resChl <- &RPCResponse{
			Result: fftypes.JSONAnyPtr(`false`),
		}
	}()

	var needString string
	rpcErr := rc.waitResponse(ctx, &needString, "000000001", &RPCRequest{}, time.Now(), resChl)
	assert.Regexp(t, "FF22065", rpcErr.Error())
}

func TestHandleSubscriptionNotificationBadSubID(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	rc.handleSubscriptionNotification(ctx, &RPCResponse{})
}

func TestHandleSubscriptionNotificationClosed(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	done()

	rc.activeSubsBySubID["12345"] = &sub{
		ctx: ctx, // closed
	}
	rc.handleSubscriptionNotification(ctx, &RPCResponse{
		Params: fftypes.JSONAnyPtr(`{"subscription":"12345"}`),
	})
}

func TestHandleSubscriptionConfirmServerError(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	errChl := make(chan *RPCError, 1)
	rc.handleSubscriptionConfirm(ctx, &sub{
		newSubResponse: errChl,
	}, &RPCResponse{
		Error: &RPCError{
			Code:    int64(RPCCodeInternalError),
			Message: "pop",
		},
	})
	rpcErr := <-errChl
	assert.Regexp(t, "pop", rpcErr.Error())
}

func TestHandleSubscriptionConfirmBadSub(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	errChl := make(chan *RPCError, 1)
	rc.handleSubscriptionConfirm(ctx, &sub{newSubResponse: errChl}, &RPCResponse{})
	rpcErr := <-errChl
	assert.Regexp(t, "FF22066", rpcErr.Error())
}

func TestRemoveSubscriptionPending(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	s := &sub{
		localID:      fftypes.NewUUID(),
		pendingReqID: "12345",
	}
	s.ctx, s.cancelCtx = context.WithCancel(ctx)
	rc.pendingSubsByReqID["12345"] = s
	rc.removeSubscription(s)
	assert.Empty(t, rc.pendingSubsByReqID)
}

func TestHandleReonnnectOK(t *testing.T) {
	ctx, rc, toServer, fromServer, done := newTestWSRPC(t)
	defer done()

	var err error
	rc.client, err = wsclient.New(ctx, &rc.wsConf, nil, nil /* so we can invoke it directly */)
	assert.NoError(t, err)

	err = rc.client.Connect()
	assert.NoError(t, err)

	rc.wsConf.DisableReconnect = false

	go rc.receiveLoop(ctx)

	s, errChl := rc.addConfiguredSub(ctx, []interface{}{"newHeads"})

	inflightID, inflightRes := rc.addInflightRequest(&RPCRequest{})

	go func() {
		msg := <-toServer
		assert.Equal(t, `{"jsonrpc":"2.0","id":"000000002","method":"eth_subscribe","params":["newHeads"]}`, msg)
		fromServer <- `{"jsonrpc":"2.0","id":"000000002","result":"0x9ce59a13059e417087c02d3236a0b1cc"}`
	}()

	err = rc.handleReconnect(ctx, rc.client)
	assert.NoError(t, err)
	rpcErr := <-errChl
	assert.Nil(t, rpcErr)
	assert.Equal(t, "0x9ce59a13059e417087c02d3236a0b1cc", s.currentSubID)

	rpcRes := <-inflightRes
	assert.Regexp(t, "FF22067", rpcRes.Error.Error())
	assert.Equal(t, inflightID, rpcRes.ID.AsString())
}

func TestHandleReonnnectFail(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	var err error
	rc.client, err = wsclient.New(ctx, &rc.wsConf, nil, nil /* so we can invoke it directly */)
	assert.NoError(t, err)

	rc.wsConf.DisableReconnect = false

	_, _ = rc.addConfiguredSub(ctx, []interface{}{
		map[bool]bool{false: true}, // cannot be serialized
	})

	err = rc.handleReconnect(ctx, rc.client)
	assert.Regexp(t, "FF22011", err)
}

func TestConnectClosedContextFail(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)

	err := rc.Connect(ctx)
	assert.NoError(t, err)
	done()

	connected := make(chan struct{})
	rc.connected = connected
	err = rc.waitConnected(ctx, connected)
	assert.Regexp(t, "FF22068", err)
}

func TestDeliverCallResponseNonBlocking(t *testing.T) {
	ctx, rc, _, _, done := newTestWSRPC(t)
	defer done()

	rc.deliverCallResponse(ctx, make(chan *RPCResponse), &RPCResponse{})
}
