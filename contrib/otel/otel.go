// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package otel bridges ibkr.Metrics onto OpenTelemetry instruments.
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
