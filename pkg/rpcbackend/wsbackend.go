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

package rpcbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/common/pkg/wsclient"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/sirupsen/logrus"
)

// WebSocketRPCClient performs communication over a websocket with an Ethereum JSON/RPC endpoint
//
// - Manages websocket connect/reconnect with keepalive etc.
// - Manages subscriptions with a local ID, so they re-established automatically after reconnect
// - Allows synchronous exchange over the WebSocket so you don't have to maintain a separate HTTP connection too
type WebSocketRPCClient interface {
	RPC
	Subscribe(ctx context.Context, params ...interface{}) (sub Subscription, error *RPCError)
	Subscriptions() []Subscription
	UnsubscribeAll(ctx context.Context) (error *RPCError)
	Connect(ctx context.Context) error
	Close()
}

// NewRPCClient Constructor
func NewWSRPCClient(wsConf *wsclient.WSConfig) WebSocketRPCClient {
	return &wsRPCClient{
		wsConf:             *wsConf,
		calls:              make(map[string]chan *RPCResponse),
		configuredSubs:     make(map[fftypes.UUID]*sub),
		pendingSubsByReqID: make(map[string]*sub),
		activeSubsBySubID:  make(map[string]*sub),
	}
}

type Subscription interface {
	LocalID() *fftypes.UUID // does not change through reconnects
	Notifications() chan *RPCSubscriptionNotification
	Unsubscribe(ctx context.Context) *RPCError
}

type RPCSubscriptionNotification struct {
	CurrentSubID string // will change on each reconnect
	Result       *fftypes.JSONAny
}

type wsRPCClient struct {
	mux                sync.Mutex
	wsConf             wsclient.WSConfig
	client             wsclient.WSClient
	requestCounter     int64
	connected          chan struct{}
	calls              map[string]chan *RPCResponse
	configuredSubs     map[fftypes.UUID]*sub
	pendingSubsByReqID map[string]*sub
	activeSubsBySubID  map[string]*sub
}

type sub struct {
	localID        *fftypes.UUID
	rc             *wsRPCClient
	ctx            context.Context
	cancelCtx      context.CancelFunc
	params         []interface{}
	pendingReqID   string
	currentSubID   string
	newSubResponse chan *RPCError
	notifications  chan *RPCSubscriptionNotification
}

func (rc *wsRPCClient) Connect(ctx context.Context) (err error) {
	rc.client, err = wsclient.New(ctx, &rc.wsConf, nil, rc.handleReconnect)
	if err != nil {
		return err
	}
	go rc.receiveLoop(log.WithLogFields(ctx, "role", "rpc_websocket"))

	// Wait until the afterConnect hook has been driven
	connected := make(chan struct{})
	rc.connected = connected
	if err := rc.client.Connect(); err != nil {
		return err
	}
	return rc.waitConnected(ctx, connected)
}

func (rc *wsRPCClient) waitConnected(ctx context.Context, connected chan struct{}) error {
	select {
	case <-connected:
	case <-ctx.Done():
		return i18n.NewError(ctx, signermsgs.MsgContextCancelledWSConnect)
	}
	return nil
}

func (rc *wsRPCClient) Close() {
	if rc.client != nil {
		rc.client.Close()
	}
}

func (rc *wsRPCClient) handleReconnect(ctx context.Context, w wsclient.WSClient) error {
	if !rc.wsConf.DisableReconnect {
		calls, subs := rc.clearActiveReturnConfiguredSubs()
		for rpcID, c := range calls {
			rc.deliverCallResponse(ctx, c, &RPCResponse{
				ID:    fftypes.JSONAnyPtr(`"` + rpcID + `"`),
				Error: NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgWebSocketReconnected),
			})
		}
		for _, s := range subs {
			log.L(ctx).Infof("Resubscribing %s after WebSocket reconnect", s.localID)
			_, rpcErr := s.sendSubscribe(ctx)
			if rpcErr != nil {
				log.L(ctx).Errorf("Failed to send resubscribe: %s", rpcErr)
				return rpcErr.Error()
			}
		}
	}
	if rc.connected != nil {
		close(rc.connected)
		rc.connected = nil
	}
	return nil
}

func (rc *wsRPCClient) addInflightRequest(req *RPCRequest) (string, chan *RPCResponse) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	rc.requestCounter++
	reqID := fmt.Sprintf("%.9d", rc.requestCounter)
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	resChl := make(chan *RPCResponse, 1)
	rc.calls[reqID] = resChl
	return reqID, resChl
}

func (rc *wsRPCClient) addInflightSub(s *sub) string {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	rc.requestCounter++
	s.pendingReqID = fmt.Sprintf("%.9d", rc.requestCounter)
	s.currentSubID = ""
	rc.pendingSubsByReqID[s.pendingReqID] = s
	return s.pendingReqID
}

func (rc *wsRPCClient) popInflight(rpcID string) (*sub, chan *RPCResponse) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	s, ok := rc.pendingSubsByReqID[rpcID]
	if ok {
		s.pendingReqID = ""
		delete(rc.pendingSubsByReqID, rpcID)
		return s, nil
	}
	inflightCall, ok := rc.calls[rpcID]
	if ok {
		delete(rc.calls, rpcID)
		return nil, inflightCall
	}
	return nil, nil
}

func (rc *wsRPCClient) addActiveSub(s *sub, subscriptionID string) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	s.currentSubID = subscriptionID
	rc.activeSubsBySubID[s.currentSubID] = s
}

func (rc *wsRPCClient) getActiveSub(subID string) *sub {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	return rc.activeSubsBySubID[subID]
}

func (rc *wsRPCClient) removeSubscription(s *sub) string {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	// Removes from configured, pending and active lists
	delete(rc.configuredSubs, *s.localID)
	if s.currentSubID != "" {
		delete(rc.activeSubsBySubID, s.currentSubID)
	}
	if s.pendingReqID != "" {
		delete(rc.pendingSubsByReqID, s.pendingReqID)
	}
	s.cancelCtx() // we unblock the receiver if it was previously trying to dispatch to this subscription, before invoking unsubscribe
	return s.currentSubID
}

func (rc *wsRPCClient) addConfiguredSub(ctx context.Context, params []interface{}) (*sub, chan *RPCError) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	s := &sub{
		rc:             rc,
		localID:        fftypes.NewUUID(),
		params:         params,
		newSubResponse: make(chan *RPCError, 1),
		notifications:  make(chan *RPCSubscriptionNotification), // blocking channel for these, but Unsubscribe will unblock by cancelling ctx
	}
	s.ctx, s.cancelCtx = context.WithCancel(ctx)
	rc.configuredSubs[*s.localID] = s
	// need to return newSubResponse because it's a use-once thing (not on reconnect)
	// and will be nilled out (under lock) when the first creation response comes in
	return s, s.newSubResponse
}

func (rc *wsRPCClient) removeConfiguredSub(id *fftypes.UUID) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	delete(rc.configuredSubs, *id)
}

func (rc *wsRPCClient) clearActiveReturnConfiguredSubs() (map[string]chan *RPCResponse, map[fftypes.UUID]*sub) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	// Return a copy of all the in-flight RPC calls, before we clear those (as they will all be defunct now)
	calls := rc.calls
	rc.calls = make(map[string]chan *RPCResponse)
	// Clear the active state as considered now invalid after a reconnect
	rc.activeSubsBySubID = make(map[string]*sub)
	rc.pendingSubsByReqID = make(map[string]*sub)
	// Return all the configured ones so we can re-establish them on the new connecti
	subs := make(map[fftypes.UUID]*sub)
	for id, s := range rc.configuredSubs {
		s.currentSubID = ""
		s.pendingReqID = ""
		subs[id] = s
	}
	return calls, subs
}

func (rc *wsRPCClient) getAllSubs() []*sub {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	subs := make([]*sub, 0)
	for _, s := range rc.configuredSubs {
		subs = append(subs, s)
	}
	return subs
}

func (rc *wsRPCClient) removeInflightRequest(reqID string) {
	rc.mux.Lock()
	defer rc.mux.Unlock()
	delete(rc.calls, reqID)
}

func (rc *wsRPCClient) Subscribe(ctx context.Context, params ...interface{}) (sub Subscription, error *RPCError) {

	s, newSubResponse := rc.addConfiguredSub(ctx, params)

	reqID, rpcErr := s.sendSubscribe(ctx)
	if rpcErr != nil {
		rc.removeConfiguredSub(s.localID)
		return nil, rpcErr
	}

	select {
	case rpcErr := <-newSubResponse:
		return s, rpcErr
	case <-ctx.Done():
		rc.removeConfiguredSub(s.localID)
		return nil, NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRequestCanceledContext, reqID)
	}
}

func (s *sub) sendSubscribe(ctx context.Context) (string, *RPCError) {
	rpcReq, rpcErr := buildRequest(ctx, "eth_subscribe", s.params)
	if rpcErr != nil {
		return "", rpcErr
	}
	reqID := s.rc.addInflightSub(s)
	rpcReq.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)

	return reqID, s.rc.sendRPC(ctx, s.pendingReqID, rpcReq)
}

func (s *sub) LocalID() *fftypes.UUID {
	return s.localID
}

func (s *sub) Notifications() chan *RPCSubscriptionNotification {
	return s.notifications
}

func (s *sub) Unsubscribe(ctx context.Context) *RPCError {
	currentSubID := s.rc.removeSubscription(s)
	var resultBool bool
	if currentSubID != "" {
		// If currently active, we need to unsubscribe
		rpcErr := s.rc.CallRPC(ctx, &resultBool, "eth_unsubscribe", currentSubID)
		if rpcErr != nil {
			return rpcErr
		}
	}
	log.L(ctx).Infof("Unsubscribed %s (subid=%s,result=%t)", s.localID, currentSubID, resultBool)
	close(s.notifications)
	return nil
}

func (rc *wsRPCClient) UnsubscribeAll(ctx context.Context) (lastErr *RPCError) {
	for _, s := range rc.getAllSubs() {
		if lastErr = s.Unsubscribe(ctx); lastErr != nil {
			log.L(ctx).Errorf("Failed to unsubscribe %s: %s", s.localID, lastErr)
		}
	}
	return lastErr
}

func (rc *wsRPCClient) Subscriptions() []Subscription {
	subs := rc.getAllSubs()
	iSubs := make([]Subscription, len(subs))
	for i, s := range subs {
		iSubs[i] = s
	}
	return iSubs
}

func (rc *wsRPCClient) sendRPC(ctx context.Context, reqID string, rpcReq *RPCRequest) *RPCError {
	jsonInput, err := json.Marshal(rpcReq)
	if err == nil {
		log.L(ctx).Debugf("RPC[%s] --> %s", reqID, rpcReq.Method)
		if logrus.IsLevelEnabled(logrus.TraceLevel) {
			log.L(ctx).Tracef("RPC[%s] INPUT: %s", reqID, jsonInput)
		}
		err = rc.client.Send(ctx, jsonInput)
	}
	if err != nil {
		rpcErr := NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRPCRequestFailed, err)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, err)
		return rpcErr
	}
	return nil
}

func (rc *wsRPCClient) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError {
	start := time.Now()
	rpcReq, rpcErr := buildRequest(ctx, method, params)
	if rpcErr != nil {
		recordRPCRequest(ctx, method, statusInvalidRequest, false, time.Since(start))
		return rpcErr
	}

	reqID, resChannel := rc.addInflightRequest(rpcReq)
	defer rc.removeInflightRequest(reqID)

	if rpcErr = rc.sendRPC(ctx, reqID, rpcReq); rpcErr != nil {
		recordRPCRequest(ctx, method, statusTransportError, false, time.Since(start))
		return rpcErr
	}
	return rc.waitResponse(ctx, result, reqID, rpcReq, start, resChannel)
}

func (rc *wsRPCClient) waitResponse(ctx context.Context, result interface{}, reqID string, rpcReq *RPCRequest, start time.Time, resChannel chan *RPCResponse) *RPCError {
	var rpcRes *RPCResponse
	select {
	case rpcRes = <-resChannel:
	case <-ctx.Done():
		rpcErr := NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgRequestCanceledContext, reqID)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, rpcErr.Error())
		recordRPCRequest(ctx, rpcReq.Method, statusCanceled, false, time.Since(start))
		return rpcErr
	}
	if rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", reqID, rpcRes.Message())
		recordRPCRequest(ctx, rpcReq.Method, statusFromRPCError(rpcRes.Error.Code), false, time.Since(start))
		return rpcRes.Error
	}
	log.L(ctx).Infof("RPC[%s] <-- %s OK (%s)", reqID, rpcReq.Method, time.Since(start))
	if rpcRes.Result == nil {
		// We don't want a result for errors, but a null success response needs to go in there
		rpcRes.Result = fftypes.JSONAnyPtr(fftypes.NullString)
	}
	if result != nil {
		if err := json.Unmarshal(rpcRes.Result.Bytes(), &result); err != nil {
			err = i18n.NewError(ctx, signermsgs.MsgResultParseFailed, result, err)
			recordRPCRequest(ctx, rpcReq.Method, statusParseError, false, time.Since(start))
			return &RPCError{Code: int64(RPCCodeParseError), Message: err.Error()}
		}
	}
	recordRPCRequest(ctx, rpcReq.Method, statusOK, false, time.Since(start))
	return nil
}

func (rc *wsRPCClient) handleSubscriptionNotification(ctx context.Context, rpcRes *RPCResponse) {
	type rpcSubscriptionParams struct {
		Subscription string           `json:"subscription"` // probably hex, but not protocol assured
		Result       *fftypes.JSONAny `json:"result,omitempty"`
	}
	var subParams rpcSubscriptionParams
	if rpcRes.Params != nil {
		_ = json.Unmarshal(rpcRes.Params.Bytes(), &subParams)
	}
	if len(subParams.Subscription) == 0 {
		log.L(ctx).Warnf("RPC[%s] <-- Unable to extract subscription id from notification: %s", rpcRes.ID.AsString(), rpcRes.Params)
		return
	}

	s := rc.getActiveSub(subParams.Subscription)
	if s == nil {
		log.L(ctx).Warnf("RPC[%s] <-- Notification for unknown subscription '%s'", rpcRes.ID.AsString(), subParams.Subscription)
		return
	}

	// This is a notification that should match an active subscription
	log.L(ctx).Debugf("RPC[%s] <-- Notification for subscription %s (serverId=%s)", rpcRes.ID.AsString(), s.localID, s.currentSubID)
	select {
	case s.notifications <- &RPCSubscriptionNotification{
		CurrentSubID: s.currentSubID,
		Result:       subParams.Result,
	}:
	case <-s.ctx.Done():
		// The subscription has been unsubscribed, or we're closing
		log.L(ctx).Warnf("RPC[%s] <-- Received subscription event after unsubscribe/close %s (serverId=%s)", rpcRes.ID.AsString(), s.localID, s.currentSubID)
	}
}

func (rc *wsRPCClient) handleSubscriptionConfirm(ctx context.Context, inflightSub *sub, rpcRes *RPCResponse) {
	resChl := inflightSub.newSubResponse
	inflightSub.newSubResponse = nil // we only dispatch once (it's only new once, on reconnect it's old and there's nobody to tell if we fail)
	if rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		log.L(ctx).Warnf("RPC[%s] <-- Error creating subscription %s: %s", rpcRes.ID.AsString(), inflightSub.localID, rpcRes.Params)
		if resChl != nil {
			resChl <- rpcRes.Error
		}
		return
	}
	var subscriptionID string // we know it's probably hex, but we cannot rely on that being guaranteed
	if rpcRes.Result != nil {
		_ = json.Unmarshal(rpcRes.Result.Bytes(), &subscriptionID)
	}
	if len(subscriptionID) == 0 {
		log.L(ctx).Warnf("RPC[%s] <-- Unable to extract subscription id from eth_subscribe response: %s", rpcRes.ID.AsString(), rpcRes.Params)
		if resChl != nil {
			resChl <- NewRPCError(ctx, RPCCodeInternalError, signermsgs.MsgSubscribeResponseInvalid)
		}
		return
	}
	log.L(ctx).Infof("Subscribed %s with server subscription ID '%s'", inflightSub.localID, subscriptionID)
	rc.addActiveSub(inflightSub, subscriptionID)
	// all was good, if someone is waiting to be told, notify them
	if resChl != nil {
		resChl <- nil
	}
}

func (rc *wsRPCClient) deliverCallResponse(ctx context.Context, inflightCall chan *RPCResponse, rpcRes *RPCResponse) {
	select {
	case inflightCall <- rpcRes:
	default:
		// only considered for the very edge case of reconnect - the inflight response should only be
		// in the map until it's sent a single response, and there's a slot to ensure it never blocks
	}
}

func (rc *wsRPCClient) receiveLoop(ctx context.Context) {
	for {
		bytes, ok := <-rc.client.Receive()
		if !ok {
			log.L(ctx).Debugf("WebSocket closed")
			return
		}
		rpcRes := RPCResponse{}
		err := json.Unmarshal(bytes, &rpcRes)
		switch {
		case err != nil:
			log.L(ctx).Errorf("RPC <-- ERROR invalid data '%s': %s", bytes, err)
		case rpcRes.Method == "eth_subscription":
			rc.handleSubscriptionNotification(ctx, &rpcRes)
		default:
			// ID should match a request we sent
			inflightSub, inflightCall := rc.popInflight(rpcRes.ID.AsString())
			switch {
			case inflightSub != nil:
				rc.handleSubscriptionConfirm(ctx, inflightSub, &rpcRes)
			case inflightCall != nil:
				rc.deliverCallResponse(ctx, inflightCall, &rpcRes)
			default:
				log.L(ctx).Warnf("RPC[%s] <-- Received unexpected RPC response: %+v", rpcRes.ID.AsString(), rpcRes)
			}
		}
	}
}
