// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package grpc_prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
)

type clientReporter struct {
	metrics           *ClientMetrics
	rpcType           grpcType
	serviceName       string
	methodName        string
	startTime         time.Time
	exemplarExtractor func(context.Context) prometheus.Labels
}

func newClientReporter(
	m *ClientMetrics,
	rpcType grpcType,
	fullMethod string,
	exemplarExtractor func(context.Context) prometheus.Labels,
) *clientReporter {
	r := &clientReporter{
		metrics: m,
		rpcType: rpcType,
	}
	if r.metrics.clientHandledHistogramEnabled {
		r.startTime = time.Now()
	}
	r.serviceName, r.methodName = splitMethodName(fullMethod)
	r.metrics.clientStartedCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
	r.exemplarExtractor = exemplarExtractor
	return r
}

// timer is a helper interface to time functions.
type timer interface {
	ObserveDuration() time.Duration
}

type noOpTimer struct {
}

func (noOpTimer) ObserveDuration() time.Duration {
	return 0
}

var emptyTimer = noOpTimer{}

func (r *clientReporter) ReceiveMessageTimer() timer {
	if r.metrics.clientStreamRecvHistogramEnabled {
		hist := r.metrics.clientStreamRecvHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName)
		return prometheus.NewTimer(hist)
	}

	return emptyTimer
}

func (r *clientReporter) ReceivedMessage() {
	r.metrics.clientStreamMsgReceived.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *clientReporter) SendMessageTimer() timer {
	if r.metrics.clientStreamSendHistogramEnabled {
		hist := r.metrics.clientStreamSendHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName)
		return prometheus.NewTimer(hist)
	}

	return emptyTimer
}

func (r *clientReporter) SentMessage() {
	r.metrics.clientStreamMsgSent.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *clientReporter) Handled(code codes.Code, ctx context.Context) {
	var exemplarLabels prometheus.Labels
	if r.exemplarExtractor != nil {
		exemplarLabels = r.exemplarExtractor(ctx)
	}

	m := r.metrics.clientHandledCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName, code.String())
	exemplarAdder, isExemplarAdder := m.(prometheus.ExemplarAdder)

	if isExemplarAdder && exemplarLabels != nil {
		exemplarAdder.AddWithExemplar(1, exemplarLabels)
	} else {
		m.Inc()
	}

	if r.metrics.clientHandledHistogramEnabled {
		h := r.metrics.clientHandledHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName)
		if exemplarLabels == nil {
			h.Observe(time.Since(r.startTime).Seconds())
		} else {
			h.(prometheus.ExemplarObserver).ObserveWithExemplar(time.Since(r.startTime).Seconds(), r.exemplarExtractor(ctx))
		}
	}
}
