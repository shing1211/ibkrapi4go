// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInMemoryMetrics_CounterAccumulation(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()
	attrs := []Attr{{Key: "method", Value: "GET"}, {Key: "path", Value: "/v1/api/accounts"}}
	m.Counter(ctx, MetricHTTPRequests, 1, attrs...)
	m.Counter(ctx, MetricHTTPRequests, 2, attrs...)

	snap := m.Snapshot()
	key := SeriesKey(MetricHTTPRequests, attrs...)
	if got := snap.Counters[key]; got != 3 {
		t.Fatalf("counter = %d; want 3", got)
	}
}

func TestInMemoryMetrics_AttributeOrderIsStable(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()
	m.Counter(ctx, MetricHTTPRequests, 1, Attr{Key: "method", Value: "GET"}, Attr{Key: "path", Value: "/x"})
	m.Counter(ctx, MetricHTTPRequests, 1, Attr{Key: "path", Value: "/x"}, Attr{Key: "method", Value: "GET"})

	snap := m.Snapshot()
	if len(snap.Counters) != 1 {
		t.Fatalf("series count = %d; want 1 (attrs must be order-independent)", len(snap.Counters))
	}
	for _, v := range snap.Counters {
		if v != 2 {
			t.Fatalf("counter = %d; want 2", v)
		}
	}
}

func TestInMemoryMetrics_HistogramAggregates(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()
	for _, v := range []float64{10, 5, 20, 15} {
		m.Histogram(ctx, MetricHTTPDuration, v)
	}
	snap := m.Snapshot()
	h, ok := snap.Histograms[SeriesKey(MetricHTTPDuration)]
	if !ok {
		t.Fatalf("histogram missing from snapshot: %#v", snap.Histograms)
	}
	if h.Count != 4 {
		t.Errorf("count = %d; want 4", h.Count)
	}
	if h.Sum != 50 {
		t.Errorf("sum = %v; want 50", h.Sum)
	}
	if h.Min != 5 {
		t.Errorf("min = %v; want 5", h.Min)
	}
	if h.Max != 20 {
		t.Errorf("max = %v; want 20", h.Max)
	}
}

func TestInMemoryMetrics_GaugeLastValue(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()
	m.Gauge(ctx, MetricBreakerState, 0, Attr{Key: "state", Value: "closed"})
	m.Gauge(ctx, MetricBreakerState, 2, Attr{Key: "state", Value: "open"})

	snap := m.Snapshot()
	key := SeriesKey(MetricBreakerState, Attr{Key: "state", Value: "open"})
	if got := snap.Gauges[key]; got != 2 {
		t.Fatalf("gauge = %v; want 2", got)
	}
}

func TestInMemoryMetrics_ConcurrentSafety(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Counter(ctx, MetricHTTPRequests, 1, Attr{Key: "method", Value: "GET"})
			m.Histogram(ctx, MetricHTTPDuration, 1)
			m.Gauge(ctx, MetricBreakerState, 0)
			_ = m.Snapshot()
		}()
	}
	wg.Wait()

	snap := m.Snapshot()
	key := SeriesKey(MetricHTTPRequests, Attr{Key: "method", Value: "GET"})
	if got := snap.Counters[key]; got != 50 {
		t.Fatalf("counter = %d; want 50", got)
	}
}

func TestNopMetrics(t *testing.T) {
	m := NopMetrics()
	m.Counter(context.Background(), MetricHTTPRequests, 1)
	m.Histogram(context.Background(), MetricHTTPDuration, 1)
	m.Gauge(context.Background(), MetricBreakerState, 1)
}

func TestNilMetricsAndHelpersAreSafe(t *testing.T) {
	var m *InMemoryMetrics
	m.Counter(context.Background(), MetricHTTPRequests, 1)
	m.Histogram(context.Background(), MetricHTTPDuration, 1)
	m.Gauge(context.Background(), MetricBreakerState, 1)
	if snap := m.Snapshot(); len(snap.Counters) != 0 {
		t.Fatalf("nil snapshot counters = %d; want 0", len(snap.Counters))
	}

	incrCounter(context.Background(), nil, MetricHTTPRequests, 1)
	observeHistogram(context.Background(), nil, MetricHTTPDuration, 1)
	setGauge(context.Background(), nil, MetricBreakerState, 1)

	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if got := Instrument(nil)(base); got == nil {
		t.Fatal("Instrument(nil) returned nil transport")
	}
}

func TestInstrument_RecordsSuccess(t *testing.T) {
	m := NewInMemoryMetrics()
	rt := Instrument(m)(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/v1/api/accounts", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	snap := m.Snapshot()
	reqKey := SeriesKey(MetricHTTPRequests,
		Attr{Key: "method", Value: "GET"},
		Attr{Key: "path", Value: "/v1/api/accounts"},
		Attr{Key: "status", Value: "200"},
	)
	if got := snap.Counters[reqKey]; got != 1 {
		t.Errorf("requests counter = %d; want 1", got)
	}
	durKey := SeriesKey(MetricHTTPDuration,
		Attr{Key: "method", Value: "GET"},
		Attr{Key: "path", Value: "/v1/api/accounts"},
	)
	if h, ok := snap.Histograms[durKey]; !ok || h.Count != 1 {
		t.Errorf("duration histogram = %#v; want one observation", snap.Histograms)
	}
	if len(snap.Counters) != 1 {
		t.Errorf("unexpected error counter recorded: %#v", snap.Counters)
	}
}

func TestInstrument_RecordsHTTPError(t *testing.T) {
	m := NewInMemoryMetrics()
	rt := Instrument(m)(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.test/v1/api/orders", nil)
	resp, _ := rt.RoundTrip(req)
	resp.Body.Close()

	key := SeriesKey(MetricHTTPErrors,
		Attr{Key: "method", Value: "POST"},
		Attr{Key: "path", Value: "/v1/api/orders"},
		Attr{Key: "status", Value: "500"},
		Attr{Key: "error.class", Value: "http"},
	)
	if got := m.Snapshot().Counters[key]; got != 1 {
		t.Fatalf("http error counter = %d; want 1", got)
	}
}

func TestInstrument_RecordsTransportError(t *testing.T) {
	m := NewInMemoryMetrics()
	rt := Instrument(m)(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	}))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/v1/api/accounts", nil)
	_, _ = rt.RoundTrip(req)

	key := SeriesKey(MetricHTTPErrors,
		Attr{Key: "method", Value: "GET"},
		Attr{Key: "path", Value: "/v1/api/accounts"},
		Attr{Key: "status", Value: "0"},
		Attr{Key: "error.class", Value: "transport"},
	)
	if got := m.Snapshot().Counters[key]; got != 1 {
		t.Fatalf("transport error counter = %d; want 1", got)
	}
}

func TestBreaker_MetricsStateTransitions(t *testing.T) {
	m := NewInMemoryMetrics()
	b := NewBreaker(2, 10*time.Millisecond)
	b.SetMetrics(m)

	closedKey := SeriesKey(MetricBreakerState, Attr{Key: "state", Value: "closed"})
	if got := m.Snapshot().Gauges[closedKey]; got != 0 {
		t.Fatalf("initial gauge = %v; want 0 (closed)", got)
	}

	b.Record(errors.New("boom"), 0)
	b.Record(errors.New("boom"), 0)
	openKey := SeriesKey(MetricBreakerState, Attr{Key: "state", Value: "open"})
	if got := m.Snapshot().Gauges[openKey]; got != 2 {
		t.Fatalf("gauge after threshold = %v; want 2 (open)", got)
	}

	time.Sleep(15 * time.Millisecond)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow probe: %v", err)
	}
	halfKey := SeriesKey(MetricBreakerState, Attr{Key: "state", Value: "half-open"})
	if got := m.Snapshot().Gauges[halfKey]; got != 1 {
		t.Fatalf("gauge on probe = %v; want 1 (half-open)", got)
	}

	b.Record(nil, 200)
	if got := m.Snapshot().Gauges[closedKey]; got != 0 {
		t.Fatalf("gauge after recovery = %v; want 0 (closed)", got)
	}
}

func TestLimiter_MetricsRecordsWait(t *testing.T) {
	m := NewInMemoryMetrics()
	l := NewLimiter(100, 1, 0)
	l.SetMetrics(m)

	ctx := context.Background()
	if err := l.Wait(ctx, http.MethodGet, "/v1/api/accounts"); err != nil {
		t.Fatalf("first Wait: %v", err)
	}
	if err := l.Wait(ctx, http.MethodGet, "/v1/api/accounts"); err != nil {
		t.Fatalf("second Wait: %v", err)
	}

	snap := m.Snapshot()
	if got := snap.Counters[SeriesKey(MetricRateLimitWaits)]; got < 1 {
		t.Fatalf("wait counter = %d; want >= 1", got)
	}
	h, ok := snap.Histograms[SeriesKey(MetricRateLimitWaitMS)]
	if !ok || h.Count < 1 || h.Sum <= 0 {
		t.Fatalf("wait histogram = %#v; want positive observations", h)
	}
}

func TestTokenSource_MetricsRecordsRefreshAndFailure(t *testing.T) {
	srv := newTokenServer(t, []string{"tok-1"}, []string{""})
	m := NewInMemoryMetrics()
	ts := NewTokenSource(OAuthConfig{TokenURL: srv.URL, ClientID: "cid", ClientSecret: "sec", Metrics: m})
	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	if got := m.Snapshot().Counters[SeriesKey(MetricOAuthTokenRefreshes)]; got != 1 {
		t.Fatalf("refresh counter = %d; want 1", got)
	}

	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer fail.Close()

	m2 := NewInMemoryMetrics()
	ts2 := NewTokenSource(OAuthConfig{TokenURL: fail.URL, ClientID: "cid", ClientSecret: "sec", Metrics: m2})
	if _, err := ts2.Token(context.Background()); err == nil {
		t.Fatal("expected token failure")
	}
	if got := m2.Snapshot().Counters[SeriesKey(MetricOAuthTokenFailures)]; got != 1 {
		t.Fatalf("failure counter = %d; want 1", got)
	}
}

func TestComponentsWithoutMetricsDoNotPanic(t *testing.T) {
	b := NewBreaker(2, time.Millisecond)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow: %v", err)
	}
	b.Record(errors.New("boom"), 0)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after failure: %v", err)
	}

	l := NewLimiter(1000, 10, 1000)
	if err := l.Wait(context.Background(), http.MethodGet, "/v1/api/accounts"); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	ts := NewTokenSource(OAuthConfig{})
	ts.SetMetrics(nil)
	if ts.RefreshToken() != "" {
		t.Fatal("unexpected refresh token")
	}

	var nilBreaker *Breaker
	nilBreaker.SetMetrics(nil)
	if err := nilBreaker.Allow(); err != nil {
		t.Fatalf("nil breaker Allow: %v", err)
	}
	var nilLimiter *Limiter
	nilLimiter.SetMetrics(nil)
	if err := nilLimiter.Wait(context.Background(), "GET", "/"); err != nil {
		t.Fatalf("nil limiter Wait: %v", err)
	}
}

func TestNewMetrics_AllConstantsRegistered(t *testing.T) {
	m := NewInMemoryMetrics()
	ctx := context.Background()

	m.Counter(ctx, MetricHTTPRetries, 1, Attr{Key: "attempt", Value: "1"})
	m.Histogram(ctx, MetricHTTPRetryBackoffMS, 1.0, Attr{Key: "attempt", Value: "1"})
	m.Counter(ctx, MetricWSHeartbeatFailures, 1)
	m.Counter(ctx, MetricWSDroppedEvents, 1)
	m.Gauge(ctx, MetricWSQueueDepth, 1.0)
	m.Gauge(ctx, MetricWSActiveSubscriptions, 1.0)
	m.Counter(ctx, MetricRateLimit429s, 1, Attr{Key: "method", Value: "GET"}, Attr{Key: "path", Value: "/v1/api/order"})
	m.Histogram(ctx, MetricOrderLatencyMS, 1.0, Attr{Key: "method", Value: "POST"})

	snap := m.Snapshot()
	if len(snap.Counters) == 0 && len(snap.Gauges) == 0 && len(snap.Histograms) == 0 {
		t.Error("expected metrics to be registered")
	}
}

func TestInstrument_Records429AndOrderLatency(t *testing.T) {
	m := NewInMemoryMetrics()
	rt := Instrument(m)(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/v1/api/accounts", nil)
	resp, _ := rt.RoundTrip(req)
	resp.Body.Close()

	key429 := SeriesKey(MetricRateLimit429s,
		Attr{Key: "method", Value: "GET"},
		Attr{Key: "path", Value: "/v1/api/accounts"},
	)
	if got := m.Snapshot().Counters[key429]; got != 1 {
		t.Errorf("429 counter = %d; want 1", got)
	}

	m2 := NewInMemoryMetrics()
	rt2 := Instrument(m2)(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	req2, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.test/v1/api/order/U1234567", nil)
	resp2, _ := rt2.RoundTrip(req2)
	resp2.Body.Close()

	orderKey := SeriesKey(MetricOrderLatencyMS, Attr{Key: "method", Value: "POST"})
	if h, ok := m2.Snapshot().Histograms[orderKey]; !ok || h.Count != 1 {
		t.Errorf("order latency histogram = %#v; want one observation", m2.Snapshot().Histograms)
	}
}

func TestRetryPolicy_WithMetrics(t *testing.T) {
	m := NewInMemoryMetrics()
	attempts := 0
	rt := Retry(RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      10 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{429, 500},
		Metrics:       m,
	})(RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return &http.Response{StatusCode: 429, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/v1/api/accounts", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if attempts != 3 {
		t.Fatalf("attempts = %d; want 3", attempts)
	}

	snap := m.Snapshot()
	retryKey := SeriesKey(MetricHTTPRetries, Attr{Key: "attempt", Value: "1"})
	if got := snap.Counters[retryKey]; got != 1 {
		t.Errorf("retry counter[attempt=1] = %d; want 1", got)
	}

	backoffKey := SeriesKey(MetricHTTPRetryBackoffMS, Attr{Key: "attempt", Value: "1"})
	if h, ok := snap.Histograms[backoffKey]; !ok || h.Count < 1 {
		t.Errorf("backoff histogram = %#v; want observations", snap.Histograms)
	}
}
