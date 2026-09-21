// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func testMeter(t *testing.T) (metric.Meter, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { meterProvider.Shutdown(context.Background()) })
	return meterProvider.Meter("test"), reader
}

func TestOTelMetrics_Counter(t *testing.T) {
	meter, reader := testMeter(t)
	m := New(meter)

	ctx := context.Background()
	m.Counter(ctx, "ibkr.http.requests", 1,
		ibkr.Attr{Key: "method", Value: "GET"},
		ibkr.Attr{Key: "status", Value: "200"},
	)
	m.Counter(ctx, "ibkr.http.requests", 3,
		ibkr.Attr{Key: "method", Value: "GET"},
		ibkr.Attr{Key: "status", Value: "200"},
	)

	rm := metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("no scope metrics")
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			if metric.Name == "ibkr.http.requests" {
				found = true
				data, ok := metric.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("expected Sum[int64], got %T", metric.Data)
				}
				if len(data.DataPoints) == 0 {
					t.Fatal("no data points")
				}
				if data.DataPoints[0].Value != 4 {
					t.Fatalf("expected 4, got %d", data.DataPoints[0].Value)
				}
			}
		}
	}
	if !found {
		t.Fatal("counter ibkr.http.requests not found")
	}
}

func TestOTelMetrics_Histogram(t *testing.T) {
	meter, reader := testMeter(t)
	m := New(meter)

	ctx := context.Background()
	m.Histogram(ctx, "ibkr.http.request.duration_ms", 100,
		ibkr.Attr{Key: "method", Value: "GET"},
	)
	m.Histogram(ctx, "ibkr.http.request.duration_ms", 200,
		ibkr.Attr{Key: "method", Value: "GET"},
	)

	rm := metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			if metric.Name == "ibkr.http.request.duration_ms" {
				found = true
				data, ok := metric.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("expected Histogram[float64], got %T", metric.Data)
				}
				if len(data.DataPoints) == 0 {
					t.Fatal("no data points")
				}
				if data.DataPoints[0].Count != 2 {
					t.Fatalf("expected count 2, got %d", data.DataPoints[0].Count)
				}
			}
		}
	}
	if !found {
		t.Fatal("histogram ibkr.http.request.duration_ms not found")
	}
}

func TestOTelMetrics_Gauge(t *testing.T) {
	meter, reader := testMeter(t)
	m := New(meter)

	ctx := context.Background()
	m.Gauge(ctx, "ibkr.breaker.state", 0)
	m.Gauge(ctx, "ibkr.breaker.state", 2)

	rm := metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			if metric.Name == "ibkr.breaker.state" {
				found = true
				data, ok := metric.Data.(metricdata.Gauge[float64])
				if !ok {
					t.Fatalf("expected Gauge[float64], got %T", metric.Data)
				}
				if len(data.DataPoints) == 0 {
					t.Fatal("no data points")
				}
				if data.DataPoints[0].Value != 2 {
					t.Fatalf("expected 2, got %f", data.DataPoints[0].Value)
				}
			}
		}
	}
	if !found {
		t.Fatal("gauge ibkr.breaker.state not found")
	}
}

func TestOTelMetrics_InterfaceSatisfaction(t *testing.T) {
	meter, _ := testMeter(t)
	m := New(meter)
	var _ ibkr.Metrics = m
}

func TestOTelMetrics_ConcurrentSafety(t *testing.T) {
	meter, _ := testMeter(t)
	m := New(meter)
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.Counter(ctx, "ibkr.http.requests", 1)
				m.Histogram(ctx, "ibkr.http.request.duration_ms", float64(j))
				m.Gauge(ctx, "ibkr.breaker.state", float64(j%3))
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
