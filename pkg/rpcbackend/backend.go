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
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/sirupsen/logrus"
)

type RPCCode int64

const (
	RPCCodeParseError     RPCCode = -32700
	RPCCodeInvalidRequest RPCCode = -32600
	RPCCodeInternalError  RPCCode = -32603
)

type RPC interface {
	CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError
}

type BatchRPC interface {
	RPC
	CallRPCBatch(ctx context.Context, ops ...*RPCBatchOp) []*RPCError
}

// RPCBatchOp holds the method, params, and result destination for one call within a batch.
type RPCBatchOp struct {
	Result interface{}
	Method string
	Params []interface{}
}

// RPCBatchOpTyped is the typed equivalent of RPCBatchOp for use with CallRPCBatchTyped.
// All ops in a typed batch must share the same result type T.
type RPCBatchOpTyped[T any] struct {
	Result *T
	Method string
	Params []interface{}
}

// Backend performs communication with a backend
type Backend interface {
	BatchRPC
	SyncRequest(ctx context.Context, rpcReq *RPCRequest) (rpcRes *RPCResponse, err error)
}

// NewRPCClient Constructor
func NewRPCClient(client *resty.Client) Backend {
	return NewRPCClientWithOption(client, RPCClientOptions{})
}

// NewRPCClientWithOption Constructor
func NewRPCClientWithOption(client *resty.Client, options RPCClientOptions) Backend {
	rpcClient := &RPCClient{
		client: client,
	}

	if options.MaxConcurrentRequest > 0 {
		rpcClient.concurrencySlots = make(chan bool, options.MaxConcurrentRequest)
	}

	return rpcClient
}

type RPCClient struct {
	client           *resty.Client
	concurrencySlots chan bool
	requestCounter   int64
}

type RPCClientOptions struct {
	MaxConcurrentRequest int64
}

type RPCRequest struct {
	JSONRpc string             `json:"jsonrpc"`
	ID      *fftypes.JSONAny   `json:"id"`
	Method  string             `json:"method"`
	Params  []*fftypes.JSONAny `json:"params,omitempty"`
}

type RPCError struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    fftypes.JSONAny `json:"data,omitempty"`
}

func (e *RPCError) Error() error {
	return errors.New(e.Message)
}

func (e *RPCError) String() string {
	return e.Message
}

// RPCResponseTyped is a JSON-RPC response envelope that decodes Result directly
// into T, bypassing the fftypes.JSONAny intermediate used by RPCResponse.
type RPCResponseTyped[T any] struct {
	JSONRpc string           `json:"jsonrpc"`
	ID      *fftypes.JSONAny `json:"id"`
	Result  T                `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
	// Only for subscription notifications
	Method string           `json:"method,omitempty"`
	Params *fftypes.JSONAny `json:"params,omitempty"`
}

func (r *RPCResponseTyped[T]) Message() string {
	if r.Error != nil {
		return r.Error.Message
	}
	return ""
}

// RPCResponse is the standard JSON-RPC response type, where Result is captured
// as raw bytes in a fftypes.JSONAny for a second json.Unmarshal by the caller.
type RPCResponse = RPCResponseTyped[*fftypes.JSONAny]

func (rc *RPCClient) reserveConcurrencySlot(ctx context.Context, id any) (func(), *RPCError) {
	if rc.concurrencySlots == nil {
		return func() {}, nil
	}
	select {
	case rc.concurrencySlots <- true:
		return func() { <-rc.concurrencySlots }, nil
	case <-ctx.Done():
		err := i18n.NewError(ctx, signermsgs.MsgRequestCanceledContext, id)
		return nil, &RPCError{Code: int64(RPCCodeInternalError), Message: err.Error()}
	}
}

func (rc *RPCClient) allocateRequestID(req *RPCRequest) string {
	reqID := fmt.Sprintf(`%.9d`, atomic.AddInt64(&rc.requestCounter, 1))
	req.ID = fftypes.JSONAnyPtr(`"` + reqID + `"`)
	return reqID
}

// unmarshalRPCResult decodes a JSON-RPC result payload into the caller's target,
// treating an absent payload as JSON null. A success response with "result": null
// decodes to a nil JSONAny
func unmarshalRPCResult(data *fftypes.JSONAny, target interface{}) error {
	b := data.Bytes()
	if len(b) == 0 {
		b = []byte(fftypes.NullString)
	}
	return json.Unmarshal(b, target)
}

func (rc *RPCClient) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *RPCError {
	start := time.Now()
	rpcReq, rpcErr := buildRequest(ctx, method, params)
	if rpcErr != nil {
		recordRPCRequest(ctx, method, statusInvalidRequest, false, time.Since(start))
		return rpcErr
	}
	res, httpStatus, err := syncRequestTyped[*fftypes.JSONAny](ctx, rc, rpcReq, false)
	if err != nil {
		status := statusTransportError
		if httpStatus > 0 {
			status = statusFromHTTPCode(httpStatus)
		}
		recordRPCRequest(ctx, method, status, false, time.Since(start))
		if res.Error != nil && res.Error.Code != 0 {
			return res.Error
		}
		return &RPCError{Code: int64(RPCCodeInternalError), Message: err.Error()}
	}
	err = unmarshalRPCResult(res.Result, &result)
	if err != nil {
		err = i18n.NewError(ctx, signermsgs.MsgResultParseFailed, result, err)
		recordRPCRequest(ctx, method, statusParseError, false, time.Since(start))
		return &RPCError{Code: int64(RPCCodeParseError), Message: err.Error()}
	}
	recordRPCRequest(ctx, method, statusFromHTTPCode(httpStatus), false, time.Since(start))
	return nil
}

func fill[T any](arr []T, v T) []T {
	for i := range arr {
		arr[i] = v
	}
	return arr
}

func matchBatchResponse[T any](rpcRes *RPCResponseTyped[T], idToIndex map[string]int, ops []*RPCBatchOpTyped[T], errs []*RPCError, matched []bool) {
	if rpcRes == nil || rpcRes.ID == nil {
		return
	}
	var idStr string
	if jsonErr := json.Unmarshal(rpcRes.ID.Bytes(), &idStr); jsonErr != nil {
		return
	}
	idx, ok := idToIndex[idStr]
	if !ok {
		return
	}
	matched[idx] = true
	if rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		errs[idx] = rpcRes.Error
		return
	}
	*ops[idx].Result = rpcRes.Result
}

func (rc *RPCClient) CallRPCBatch(ctx context.Context, ops ...*RPCBatchOp) []*RPCError {
	typedOps := make([]*RPCBatchOpTyped[*fftypes.JSONAny], len(ops))
	intermediates := make([]*fftypes.JSONAny, len(ops))
	for i, op := range ops {
		typedOps[i] = &RPCBatchOpTyped[*fftypes.JSONAny]{
			Result: &intermediates[i],
			Method: op.Method,
			Params: op.Params,
		}
	}
	errs := CallRPCBatchTyped(ctx, rc, typedOps...)
	for i, op := range ops {
		if errs[i] != nil {
			continue
		}
		if unmarshalErr := unmarshalRPCResult(intermediates[i], op.Result); unmarshalErr != nil {
			unmarshalErr = i18n.NewError(ctx, signermsgs.MsgResultParseFailed, op.Result, unmarshalErr)
			errs[i] = &RPCError{Code: int64(RPCCodeParseError), Message: unmarshalErr.Error()}
		}
	}
	return errs
}

// CallRPCBatchTyped sends a JSON-RPC batch where every operation decodes its result directly
// into T, bypassing the fftypes.JSONAny intermediate used by CallRPCBatch.
// All ops in the batch must share the same result type.
func CallRPCBatchTyped[T any](ctx context.Context, rc *RPCClient, ops ...*RPCBatchOpTyped[T]) []*RPCError {
	errs := make([]*RPCError, len(ops))
	start := time.Now()
	recordRPCBatchSize(ctx, len(ops))

	batchReqs := make([]*RPCRequest, len(ops))
	idToIndex := make(map[string]int, len(ops))
	for i, op := range ops {
		rpcReq, rpcErr := buildRequest(ctx, op.Method, op.Params)
		if rpcErr != nil {
			errs[i] = rpcErr
			recordRPCRequest(ctx, op.Method, statusInvalidRequest, true, time.Since(start))
			return errs
		}
		rpcReq.JSONRpc = "2.0"
		idStr := fmt.Sprintf(`%.9d`, atomic.AddInt64(&rc.requestCounter, 1))
		rpcReq.ID = fftypes.JSONAnyPtr(`"` + idStr + `"`)
		batchReqs[i] = rpcReq
		idToIndex[idStr] = i
	}

	returnSlot, rpcErr := rc.reserveConcurrencySlot(ctx, "batch")
	if rpcErr != nil {
		batchDuration := time.Since(start)
		for _, op := range ops {
			recordRPCRequest(ctx, op.Method, statusCanceled, true, batchDuration)
		}
		return fill(errs, rpcErr)
	}
	defer returnSlot()

	var batchRes []*RPCResponseTyped[T]
	log.L(ctx).Debugf("RPC batch[%d] -->", len(ops))
	res, err := rc.client.R().
		SetContext(ctx).
		SetBody(batchReqs).
		SetResult(&batchRes).
		Post("")
	if err != nil {
		errMsg := i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, err).Error()
		log.L(ctx).Errorf("RPC batch[%d] <-- ERROR: %s", len(ops), errMsg)
		for _, op := range ops {
			recordRPCRequest(ctx, op.Method, statusTransportError, true, time.Since(start))
		}
		return fill(errs, &RPCError{Code: int64(RPCCodeInternalError), Message: errMsg})
	}
	log.L(ctx).Infof("RPC batch[%d] <-- [%d] (%s)", len(ops), res.StatusCode(), time.Since(start))
	if res.IsError() {
		errMsg := i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, res.Status()).Error()
		httpStatus := statusFromHTTPCode(res.StatusCode())
		for _, op := range ops {
			recordRPCRequest(ctx, op.Method, httpStatus, true, time.Since(start))
		}
		return fill(errs, &RPCError{Code: int64(RPCCodeInternalError), Message: errMsg})
	}

	matched := make([]bool, len(ops))
	for _, rpcRes := range batchRes {
		matchBatchResponse(rpcRes, idToIndex, ops, errs, matched)
	}

	httpStatus := statusFromHTTPCode(res.StatusCode())
	for i, op := range ops {
		if !matched[i] && errs[i] == nil {
			err := i18n.NewError(ctx, signermsgs.MsgBatchNoResponse, op.Method, i)
			errs[i] = &RPCError{Code: int64(RPCCodeInternalError), Message: err.Error()}
		}
		status := httpStatus
		if errs[i] != nil {
			switch {
			case errs[i].Code == int64(RPCCodeParseError):
				status = statusParseError
			case !matched[i]:
				status = statusTransportError
			}
		}
		recordRPCRequest(ctx, op.Method, status, true, time.Since(start))
	}

	return errs
}

// syncRequestTyped is the canonical implementation of a single JSON-RPC round-trip.
// It owns the concurrency slot, request ID allocation, logging, and HTTP call.
// SyncRequest and CallRPCTyped are thin wrappers around it.
func syncRequestTyped[T any](ctx context.Context, rc *RPCClient, rpcReq *RPCRequest, emitMetrics bool) (rpcRes RPCResponseTyped[T], httpStatus int, err error) {
	start := time.Now()
	method := rpcReq.Method
	record := func(status string) {
		if emitMetrics {
			recordRPCRequest(ctx, method, status, false, time.Since(start))
		}
	}

	returnSlot, rpcErr := rc.reserveConcurrencySlot(ctx, rpcReq.ID)
	if rpcErr != nil {
		rpcRes.ID = rpcReq.ID
		rpcRes.Error = rpcErr
		record(statusCanceled)
		return rpcRes, 0, rpcErr.Error()
	}
	defer returnSlot()

	// We always set the back-end request ID - as we need to support requests coming in from
	// multiple concurrent clients on our front-end that might use clashing IDs.
	var beReq = *rpcReq
	beReq.JSONRpc = "2.0"
	rpcTraceID := rc.allocateRequestID(&beReq)
	if rpcReq.ID != nil {
		// We're proxying a request with front-end RPC ID - log that as well
		rpcTraceID = fmt.Sprintf("%s->%s", rpcReq.ID, rpcTraceID)
	}

	log.L(ctx).Debugf("RPC[%s] --> %s", rpcTraceID, rpcReq.Method)
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		jsonInput, _ := json.Marshal(rpcReq)
		log.L(ctx).Tracef("RPC[%s] INPUT: %s", rpcTraceID, jsonInput)
	}
	rpcStartTime := time.Now()
	res, httpErr := rc.client.R().
		SetContext(ctx).
		SetBody(beReq).
		SetResult(&rpcRes).
		SetError(&rpcRes).
		Post("")

	// Restore the original ID
	rpcRes.ID = rpcReq.ID
	if httpErr != nil {
		httpErr = i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, httpErr)
		log.L(ctx).Errorf("RPC[%s] <-- ERROR: %s", rpcTraceID, httpErr)
		rpcRes.Error = &RPCError{Code: int64(RPCCodeInternalError), Message: httpErr.Error()}
		record(statusTransportError)
		return rpcRes, 0, httpErr
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		jsonOutput, _ := json.Marshal(rpcRes)
		log.L(ctx).Tracef("RPC[%s] OUTPUT: %s", rpcTraceID, jsonOutput)
	}
	// JSON/RPC allows errors to be returned with a 200 status code, as well as other status codes
	if res.IsError() || rpcRes.Error != nil && rpcRes.Error.Code != 0 {
		rpcMsg := rpcRes.Message()
		errLog := rpcMsg
		if rpcMsg == "" {
			// Log the raw result in the case of JSON parse error etc. (note that Resty no longer
			// returns this as an error - rather the body comes back raw)
			errLog = string(res.Body())
			rpcMsg = i18n.NewError(ctx, signermsgs.MsgRPCRequestFailed, res.Status()).Error()
		}
		log.L(ctx).Errorf("RPC[%s] <-- [%d]: %s", rpcTraceID, res.StatusCode(), errLog)
		record(statusFromHTTPCode(res.StatusCode()))
		return rpcRes, res.StatusCode(), errors.New(rpcMsg)
	}
	log.L(ctx).Infof("RPC[%s] <-- %s [%d] OK (%.2fms)", rpcTraceID, rpcReq.Method, res.StatusCode(), float64(time.Since(rpcStartTime))/float64(time.Millisecond))
	record(statusFromHTTPCode(res.StatusCode()))
	return rpcRes, res.StatusCode(), nil
}

// SyncRequest sends an individual RPC request to the backend (always over HTTP currently),
// and waits synchronously for the response, or an error.
//
// In all return paths *including error paths* the RPCResponse is populated
// so the caller has an RPC structure to send back to the front-end caller.
func (rc *RPCClient) SyncRequest(ctx context.Context, rpcReq *RPCRequest) (*RPCResponse, error) {
	rpcRes, _, err := syncRequestTyped[*fftypes.JSONAny](ctx, rc, rpcReq, true)
	if err != nil {
		return &rpcRes, err
	}
	if rpcRes.Result == nil {
		// We don't want a result for errors, but a null success response needs to go in there
		rpcRes.Result = fftypes.JSONAnyPtr(fftypes.NullString)
	}
	return &rpcRes, nil
}

// CallRPCTyped performs a single JSON-RPC call and decodes the result directly into T.
// More efficient than parsing into generic JSON bytes result structure first.
func CallRPCTyped[T any](ctx context.Context, rc *RPCClient, result *T, method string, params ...interface{}) *RPCError {
	start := time.Now()
	rpcReq, rpcErr := buildRequest(ctx, method, params)
	if rpcErr != nil {
		recordRPCRequest(ctx, method, statusInvalidRequest, false, time.Since(start))
		return rpcErr
	}
	typed, httpStatus, err := syncRequestTyped[T](ctx, rc, rpcReq, false)
	if err != nil {
		status := statusTransportError
		if httpStatus > 0 {
			status = statusFromHTTPCode(httpStatus)
		}
		recordRPCRequest(ctx, method, status, false, time.Since(start))
		if typed.Error != nil && typed.Error.Code != 0 {
			return typed.Error
		}
		return &RPCError{Code: int64(RPCCodeInternalError), Message: err.Error()}
	}
	*result = typed.Result
	recordRPCRequest(ctx, method, statusFromHTTPCode(httpStatus), false, time.Since(start))
	return nil
}

func RPCErrorResponse(err error, id *fftypes.JSONAny, code RPCCode) *RPCResponse {
	return &RPCResponse{
		JSONRpc: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    int64(code),
			Message: err.Error(),
		},
	}
}

func NewRPCError(ctx context.Context, code RPCCode, msg i18n.ErrorMessageKey, inserts ...interface{}) *RPCError {
	return &RPCError{Code: int64(code), Message: i18n.NewError(ctx, msg, inserts...).Error()}
}

func buildRequest(ctx context.Context, method string, params []interface{}) (*RPCRequest, *RPCError) {
	req := &RPCRequest{
		JSONRpc: "2.0",
		Method:  method,
		Params:  make([]*fftypes.JSONAny, len(params)),
	}
	for i, param := range params {
		b, err := json.Marshal(param)
		if err != nil {
			return nil, NewRPCError(ctx, RPCCodeInvalidRequest, signermsgs.MsgInvalidParam, i, method, err)
		}
		req.Params[i] = fftypes.JSONAnyPtrBytes(b)
	}
	return req, nil
}
