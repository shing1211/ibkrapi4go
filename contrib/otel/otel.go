// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package otel bridges ibkr.Metrics and internal.Telemetry onto OpenTelemetry
// instruments and spans.
//
// This is a contrib package — it imports OpenTelemetry and is not part of the
// core SDK's dependency set (ADR 0004). Import it only when you need OTel
// integration:
//
//	import "github.com/shing1211/ibkrapi4go/contrib/otel"
package otel

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// OTelMetrics implements ibkr.Metrics by forwarding observations to
// OpenTelemetry instruments. It is safe for concurrent use.
type OTelMetrics struct {
	meter      metric.Meter
	mu         sync.Mutex
	counters   map[string]metric.Int64Counter
	histograms map[string]metric.Float64Histogram
	gauges     map[string]metric.Float64Gauge
}

// New returns an ibkr.Metrics that bridges onto the given OTel meter.
// The meter should be created with a descriptive name, e.g.:
//
//	meter := otel.Meter("ibkrapi4go")
//	m := otel.New(meter)
func New(meter metric.Meter) *OTelMetrics {
	return &OTelMetrics{
		meter:      meter,
		counters:   map[string]metric.Int64Counter{},
		histograms: map[string]metric.Float64Histogram{},
		gauges:     map[string]metric.Float64Gauge{},
	}
}

// Counter adds delta to the named OTel counter.
func (m *OTelMetrics) Counter(ctx context.Context, name string, delta int64, attrs ...ibkr.Attr) {
	m.mu.Lock()
	c, ok := m.counters[name]
	if !ok {
		var err error
		if c, err = m.meter.Int64Counter(name); err != nil {
			m.mu.Unlock()
			return
		}
		m.counters[name] = c
	}
	m.mu.Unlock()
	c.Add(ctx, delta, metric.WithAttributes(toOTEL(attrs)...))
}

// Histogram records value under the named OTel histogram.
func (m *OTelMetrics) Histogram(ctx context.Context, name string, value float64, attrs ...ibkr.Attr) {
	m.mu.Lock()
	h, ok := m.histograms[name]
	if !ok {
		var err error
		if h, err = m.meter.Float64Histogram(name, metric.WithUnit("ms")); err != nil {
			m.mu.Unlock()
			return
		}
		m.histograms[name] = h
	}
	m.mu.Unlock()
	h.Record(ctx, value, metric.WithAttributes(toOTEL(attrs)...))
}

// Gauge records the latest value of the named OTel gauge.
func (m *OTelMetrics) Gauge(ctx context.Context, name string, value float64, attrs ...ibkr.Attr) {
	m.mu.Lock()
	g, ok := m.gauges[name]
	if !ok {
		var err error
		if g, err = m.meter.Float64Gauge(name); err != nil {
			m.mu.Unlock()
			return
		}
		m.gauges[name] = g
	}
	m.mu.Unlock()
	g.Record(ctx, value, metric.WithAttributes(toOTEL(attrs)...))
}

// toOTEL converts ibkr.Attr slices to OTel attribute.KeyValue slices.
func toOTEL(attrs []ibkr.Attr) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, attribute.String(a.Key, a.Value))
	}
	return out
}

// Compile-time interface satisfaction check.
var _ ibkr.Metrics = (*OTelMetrics)(nil)

// OTelTracing implements internal.Telemetry and bridges lifecycle events
// to OpenTelemetry spans. It is safe for concurrent use.
type OTelTracing struct {
	tracer trace.Tracer
}

// NewTracing returns an internal.Telemetry that bridges to OTel spans.
func NewTracing(tracer trace.Tracer) *OTelTracing {
	return &OTelTracing{tracer: tracer}
}

// TraceMetrics combines OTelTracing and OTelMetrics into a single
// telemetry sink that handles both traces and metrics.
type TraceMetrics struct {
	*OTelTracing
	*OTelMetrics
}

// NewTraceMetrics returns a TraceMetrics that bridges both metrics and traces.
// meter and tracer should come from the same OTel MeterProvider/TracerProvider.
func NewTraceMetrics(meter metric.Meter, tracer trace.Tracer) *TraceMetrics {
	return &TraceMetrics{
		OTelTracing: NewTracing(tracer),
		OTelMetrics: New(meter),
	}
}

func (t *OTelTracing) OnRequestStart(ctx context.Context, info ibkr.RequestInfo) context.Context {
	if t.tracer == nil {
		return ctx
	}
	ctx, _ = t.tracer.Start(ctx, info.Method+" "+info.Path,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", info.Method),
			attribute.String("http.url", info.Path),
			attribute.String("request.id", info.RequestID),
		),
	)
	return ctx
}

func (t *OTelTracing) OnRequestEnd(ctx context.Context, info ibkr.RequestInfo, resp ibkr.ResponseInfo) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("duration_ms", resp.Duration.String()),
	)
	if resp.Err != nil {
		span.RecordError(resp.Err)
	}
	span.End()
}

func (t *OTelTracing) OnWSConnect(ctx context.Context, info ibkr.WSConnInfo) {
	if t.tracer == nil {
		return
	}
	_, span := t.tracer.Start(ctx, "ws.connect",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("ws.event", info.Event),
			attribute.String("ws.url", info.URL),
			attribute.Int("ws.subscriptions", info.Subscriptions),
		),
	)
	span.End()
}

func (t *OTelTracing) OnWSDisconnect(ctx context.Context, info ibkr.WSConnInfo) {
	if t.tracer == nil {
		return
	}
	_, span := t.tracer.Start(ctx, "ws.disconnect",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("ws.event", info.Event),
			attribute.String("ws.url", info.URL),
			attribute.Int("ws.subscriptions", info.Subscriptions),
		),
	)
	if info.Err != nil {
		span.RecordError(info.Err)
	}
	span.End()
}

func (t *OTelTracing) OnWSSubscribe(ctx context.Context, info ibkr.WSSubInfo) {
	if t.tracer == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("ws.event", info.Event),
	}
	for _, id := range info.ConIDs {
		attrs = append(attrs, attribute.Int("ws.conids", id))
	}
	for _, f := range info.Fields {
		attrs = append(attrs, attribute.String("ws.fields", f))
	}
	_, span := t.tracer.Start(ctx, "ws.subscribe", trace.WithAttributes(attrs...))
	span.End()
}

func (t *OTelTracing) OnWSUnsubscribe(ctx context.Context, info ibkr.WSSubInfo) {
	if t.tracer == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("ws.event", info.Event),
	}
	for _, id := range info.ConIDs {
		attrs = append(attrs, attribute.Int("ws.conids", id))
	}
	_, span := t.tracer.Start(ctx, "ws.unsubscribe", trace.WithAttributes(attrs...))
	span.End()
}

func (t *OTelTracing) OnOrderSubmit(ctx context.Context, info ibkr.OrderEventInfo) {
	if t.tracer == nil {
		return
	}
	_, span := t.tracer.Start(ctx, "order.submit",
		trace.WithAttributes(
			attribute.String("order.event", info.Event),
			attribute.String("account.id", info.AccountID),
			attribute.String("order.client_id", info.ClientOrderID),
			attribute.Int("contract.conid", info.ConID),
		),
	)
	if info.Err != nil {
		span.RecordError(info.Err)
	}
	span.End()
}

func (t *OTelTracing) OnOrderUpdate(ctx context.Context, info ibkr.OrderEventInfo) {
	if t.tracer == nil {
		return
	}
	_, span := t.tracer.Start(ctx, "order.update",
		trace.WithAttributes(
			attribute.String("order.event", info.Event),
			attribute.String("order.id", info.OrderID),
			attribute.String("account.id", info.AccountID),
		),
	)
	if info.Err != nil {
		span.RecordError(info.Err)
	}
	span.End()
}

// Compile-time interface satisfaction check for OTelTracing.
var _ ibkr.Telemetry = (*OTelTracing)(nil)
