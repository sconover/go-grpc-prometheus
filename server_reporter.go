// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package grpc_prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
)

type serverReporter struct {
	metrics     *ServerMetrics
	rpcType     grpcType
	serviceName string
	methodName  string
	startTime   time.Time
	exemplarExtractor func(context.Context) prometheus.Labels
}

func newServerReporter(
	m *ServerMetrics,
	rpcType grpcType,
	fullMethod string,
	exemplarExtractor func(context.Context) prometheus.Labels,
) *serverReporter {
	r := &serverReporter{
		metrics: m,
		rpcType: rpcType,
	}
	if r.metrics.serverHandledHistogramEnabled {
		r.startTime = time.Now()
	}
	r.serviceName, r.methodName = splitMethodName(fullMethod)
	r.metrics.serverStartedCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
	r.exemplarExtractor = exemplarExtractor
	return r
}

func (r *serverReporter) ReceivedMessage() {
	r.metrics.serverStreamMsgReceived.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *serverReporter) SentMessage() {
	r.metrics.serverStreamMsgSent.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *serverReporter) Handled(code codes.Code, ctx context.Context) {
	r.metrics.serverHandledCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName, code.String()).Inc()
	if r.metrics.serverHandledHistogramEnabled {
		h := r.metrics.serverHandledHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName)
		if r.exemplarExtractor == nil {
			h.Observe(time.Since(r.startTime).Seconds())
		} else {
			h.(prometheus.ExemplarObserver).ObserveWithExemplar(time.Since(r.startTime).Seconds(), r.exemplarExtractor(ctx))
		}
	}
}
