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
	"testing"
	"time"

	"github.com/hyperledger-firefly/common/pkg/metric"
	"github.com/stretchr/testify/assert"
)

func resetRPCMetrics() { rpcMetrics = nil }

func TestEnableMetrics(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()
	mr := metric.NewPrometheusMetricsRegistry("test_rpcbackend")

	EnableMetrics(ctx, mr)
	assert.NotNil(t, rpcMetrics)
}

func TestEnableMetricsIdempotent(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()
	mr := metric.NewPrometheusMetricsRegistry("test_rpcbackend_idem")

	EnableMetrics(ctx, mr)
	first := rpcMetrics
	EnableMetrics(ctx, mr)
	assert.Same(t, first, rpcMetrics)
}

func TestEnableMetricsNilRegistry(t *testing.T) {
	defer resetRPCMetrics()
	EnableMetrics(context.Background(), nil)
	assert.Nil(t, rpcMetrics)
}

func TestEnableMetricsSubsystemConflict(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()
	mr := metric.NewPrometheusMetricsRegistry("test_rpcbackend_conflict")
	_, _ = mr.NewMetricsManagerForSubsystem(ctx, MetricsSubsystem)

	EnableMetrics(ctx, mr)
	assert.Nil(t, rpcMetrics)
}

func TestRecordRPCRequestNoOpWhenDisabled(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		recordRPCRequest(ctx, "eth_blockNumber", statusOK, false, 10*time.Millisecond)
	})
}

func TestRecordRPCBatchNoOpWhenDisabled(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		recordRPCBatchSize(ctx, 3)
	})
}

func TestRecordRPCBatchEmitWhenEnabled(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()
	mr := metric.NewPrometheusMetricsRegistry("test_rpcbackend_batch")
	EnableMetrics(ctx, mr)

	assert.NotPanics(t, func() {
		recordRPCBatchSize(ctx, 1)
		recordRPCBatchSize(ctx, 25)
		recordRPCBatchSize(ctx, 0)
	})
}

func TestRecordRPCRequestEmitWhenEnabled(t *testing.T) {
	defer resetRPCMetrics()
	ctx := context.Background()
	mr := metric.NewPrometheusMetricsRegistry("test_rpcbackend_emit")
	EnableMetrics(ctx, mr)

	assert.NotPanics(t, func() {
		recordRPCRequest(ctx, "eth_blockNumber", "200", false, 25*time.Millisecond)
		recordRPCRequest(ctx, "", statusTransportError, false, time.Millisecond)
		recordRPCRequest(ctx, "eth_call", statusFromRPCError(-32000), false, 5*time.Millisecond)
	})
}

func TestStatusHelpers(t *testing.T) {
	assert.Equal(t, "200", statusFromHTTPCode(200))
	assert.Equal(t, statusOK, statusFromRPCError(0))
	assert.Equal(t, "-32000", statusFromRPCError(-32000))
	assert.Equal(t, "unknown", rpcMethodLabel(""))
	assert.Equal(t, "eth_chainId", rpcMethodLabel("eth_chainId"))
}
