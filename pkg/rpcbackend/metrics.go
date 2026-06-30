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
	"strconv"
	"time"

	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/common/pkg/metric"
)

const (
	MetricsSubsystem = "ffsigner_rpcbackend"

	metricCounterRPCRequestTotal        = "rpc_request_total"
	metricHistogramRPCRequestDurationMs = "rpc_request_duration_milliseconds"
	metricHistogramRPCBatchSize         = "rpc_batch_size"

	labelMethod = "method"
	labelStatus = "status"
	labelBatch  = "batch"

	statusTransportError = "transport_error"
	statusCanceled       = "canceled"
	statusInvalidRequest = "invalid_request"
	statusParseError     = "parse_error"
	statusOK             = "ok"
)

var rpcMetrics metric.MetricsManager

var rpcRequestLabels = []string{labelMethod, labelStatus, labelBatch}
var rpcDurationLabels = []string{labelMethod}

// rpcDurationMsBuckets covers fast local RPC through slow remote node calls.
var rpcDurationMsBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}

// rpcBatchSizeBuckets covers typical JSON-RPC HTTP batch sizes.
var rpcBatchSizeBuckets = []float64{1, 2, 5, 10, 25, 50, 100}

func durationMs(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func statusFromHTTPCode(code int) string {
	return strconv.Itoa(code)
}

func statusFromRPCError(code int64) string {
	if code == 0 {
		return statusOK
	}
	return strconv.FormatInt(code, 10)
}

func rpcMethodLabel(method string) string {
	if method == "" {
		return "unknown"
	}
	return method
}

// EnableMetrics registers RPC backend metrics
func EnableMetrics(ctx context.Context, metricsRegistry metric.MetricsRegistry) {
	if rpcMetrics != nil {
		return
	}
	if metricsRegistry == nil {
		return
	}
	mm, err := metricsRegistry.NewMetricsManagerForSubsystem(ctx, MetricsSubsystem)
	if err != nil {
		log.L(ctx).Errorf("failed to create metrics manager for subsystem %s: %v", MetricsSubsystem, err)
		return
	}
	rpcMetrics = mm

	rpcMetrics.NewCounterMetricWithLabels(ctx, metricCounterRPCRequestTotal, "Total number of RPC backend requests", rpcRequestLabels, false)
	rpcMetrics.NewHistogramMetricWithLabels(ctx, metricHistogramRPCRequestDurationMs, "Duration of RPC backend requests in milliseconds", rpcDurationMsBuckets, rpcDurationLabels, false)
	rpcMetrics.NewHistogramMetric(ctx, metricHistogramRPCBatchSize, "Number of RPC calls per HTTP batch request", rpcBatchSizeBuckets, false)
}

func recordRPCRequest(ctx context.Context, method, status string, batch bool, duration time.Duration) {
	if rpcMetrics == nil {
		return
	}
	method = rpcMethodLabel(method)
	labels := map[string]string{
		labelMethod: method,
		labelStatus: status,
		labelBatch:  strconv.FormatBool(batch),
	}
	rpcMetrics.IncCounterMetricWithLabels(ctx, metricCounterRPCRequestTotal, labels, nil)
	rpcMetrics.ObserveHistogramMetricWithLabels(ctx, metricHistogramRPCRequestDurationMs, durationMs(duration), map[string]string{
		labelMethod: method,
	}, nil)
}

func recordRPCBatchSize(ctx context.Context, batchSize int) {
	if rpcMetrics == nil || batchSize <= 0 {
		return
	}
	rpcMetrics.ObserveHistogramMetric(ctx, metricHistogramRPCBatchSize, float64(batchSize), nil)
}
