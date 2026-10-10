// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Attr is a low-cardinality metric attribute. Keys and values must never carry
// PII, account ids, conids, order ids, or tokens.
type Attr struct {
	Key   string
	Value string
}

// Metrics receives metric observations. Implementations MUST be safe for
// concurrent use and MUST NOT block. It is shaped to bridge directly onto
// OpenTelemetry instruments (Counter.Add / Histogram.Record / Gauge.Record) but
// carries no OTel dependency (ADR 0013).
type Metrics interface {
	// Counter adds delta to the named monotonically increasing counter.
	Counter(ctx context.Context, name string, delta int64, attrs ...Attr)
	// Histogram records value under the named distribution.
	Histogram(ctx context.Context, name string, value float64, attrs ...Attr)
	// Gauge records the latest value of the named gauge.
	Gauge(ctx context.Context, name string, value float64, attrs ...Attr)
}

// Metric names. Attributes are limited to low-cardinality dimensions.
const (
	// MetricHTTPRequests counts outbound HTTP requests.
	MetricHTTPRequests = "ibkr.http.requests"
	// MetricHTTPErrors counts failed requests (transport error or status >= 400).
	MetricHTTPErrors = "ibkr.http.errors"
	// MetricHTTPDuration records request round-trip time in milliseconds.
	MetricHTTPDuration = "ibkr.http.request.duration_ms"
	// MetricOrdersSubmitted counts accepted order submissions.
	MetricOrdersSubmitted = "ibkr.orders.submitted"
	// MetricOrdersConfirmed counts confirmed order replies.
	MetricOrdersConfirmed = "ibkr.orders.confirmed"
	// MetricOrdersModified counts modified open orders.
	MetricOrdersModified = "ibkr.orders.modified"
	// MetricOrdersCancelled counts cancelled open orders.
	MetricOrdersCancelled = "ibkr.orders.cancelled"
	// MetricOrdersRejected counts rejected order mutations.
	MetricOrdersRejected = "ibkr.orders.rejected"
	// MetricRateLimitWaits counts rate-limiter waits that slept.
	MetricRateLimitWaits = "ibkr.ratelimit.waits"
	// MetricRateLimitWaitMS records rate-limiter sleep time in milliseconds.
	MetricRateLimitWaitMS = "ibkr.ratelimit.wait_ms"
	// MetricBreakerState reports the circuit breaker state (0=closed,
	// 1=half-open, 2=open).
	MetricBreakerState = "ibkr.breaker.state"
	// MetricWSConnects counts initial WebSocket connections.
	MetricWSConnects = "ibkr.ws.connects"
	// MetricWSReconnects counts WebSocket reconnect attempts.
	MetricWSReconnects = "ibkr.ws.reconnects"
	// MetricOAuthTokenRefreshes counts successful token acquisitions/refreshes.
	MetricOAuthTokenRefreshes = "ibkr.oauth.token.refreshes"
	// MetricOAuthTokenFailures counts failed token acquisitions/refreshes.
	MetricOAuthTokenFailures = "ibkr.oauth.token.failures"
	// MetricHTTPRetries counts HTTP retry attempts.
	MetricHTTPRetries = "ibkr.http.retries"
	// MetricHTTPRetryBackoffMS records the backoff delay before each retry attempt.
	MetricHTTPRetryBackoffMS = "ibkr.http.retry.backoff_ms"
	// MetricWSHeartbeatFailures counts WebSocket ping failures.
	MetricWSHeartbeatFailures = "ibkr.ws.heartbeat.failures"
	// MetricWSDroppedEvents counts market-data events dropped because no
	// subscription wanted the conid.
	MetricWSDroppedEvents = "ibkr.ws.events.dropped"
	// MetricWSQueueDepth records the current depth of the write channel.
	MetricWSQueueDepth = "ibkr.ws.queue.depth"
	// MetricWSActiveSubscriptions records the number of active WS subscriptions.
	MetricWSActiveSubscriptions = "ibkr.ws.subscriptions.active"
	// MetricRateLimit429s counts HTTP 429 Too Many Requests responses.
	MetricRateLimit429s = "ibkr.ratelimit.429s"
	// MetricOrderLatencyMS records order submission to confirmation latency.
	MetricOrderLatencyMS = "ibkr.orders.latency_ms"
)

// nopMetrics discards all observations.
type nopMetrics struct{}

func (nopMetrics) Counter(context.Context, string, int64, ...Attr)     {}
func (nopMetrics) Histogram(context.Context, string, float64, ...Attr) {}
func (nopMetrics) Gauge(context.Context, string, float64, ...Attr)     {}

// NopMetrics returns a metrics sink that discards every observation. It is
// useful as an explicit default where a nil Metrics would otherwise be passed.
func NopMetrics() Metrics { return nopMetrics{} }

// InMemoryMetrics is a dependency-free, thread-safe Metrics implementation
// suitable for tests and simple applications. It never blocks callers beyond the
// duration of a mutex acquisition.
type InMemoryMetrics struct {
	mu         sync.Mutex
	counters   map[string]int64
	histograms map[string]*histogramState
	gauges     map[string]float64
}

type histogramState struct {
	count int64
	sum   float64
	min   float64
	max   float64
}

// NewInMemoryMetrics returns an empty in-memory metrics sink.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:   map[string]int64{},
		histograms: map[string]*histogramState{},
		gauges:     map[string]float64{},
	}
}

// Counter implements Metrics.
func (m *InMemoryMetrics) Counter(_ context.Context, name string, delta int64, attrs ...Attr) {
	if m == nil {
		return
	}
	key := SeriesKey(name, attrs...)
	m.mu.Lock()
	m.counters[key] += delta
	m.mu.Unlock()
}

// Histogram implements Metrics.
func (m *InMemoryMetrics) Histogram(_ context.Context, name string, value float64, attrs ...Attr) {
	if m == nil {
		return
	}
	key := SeriesKey(name, attrs...)
	m.mu.Lock()
	h := m.histograms[key]
	if h == nil {
		h = &histogramState{min: value, max: value}
		m.histograms[key] = h
	} else {
		if value < h.min {
			h.min = value
		}
		if value > h.max {
			h.max = value
		}
	}
	h.count++
	h.sum += value
	m.mu.Unlock()
}

// Gauge implements Metrics.
func (m *InMemoryMetrics) Gauge(_ context.Context, name string, value float64, attrs ...Attr) {
	if m == nil {
		return
	}
	key := SeriesKey(name, attrs...)
	m.mu.Lock()
	m.gauges[key] = value
	m.mu.Unlock()
}

// Snapshot returns a plain copy of the current observations.
func (m *InMemoryMetrics) Snapshot() MetricsSnapshot {
	snap := MetricsSnapshot{
		Counters:   map[string]int64{},
		Histograms: map[string]HistogramSnapshot{},
		Gauges:     map[string]float64{},
	}
	if m == nil {
		return snap
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.counters {
		snap.Counters[k] = v
	}
	for k, v := range m.histograms {
		snap.Histograms[k] = HistogramSnapshot{
			Count: v.count,
			Sum:   v.sum,
			Min:   v.min,
			Max:   v.max,
		}
	}
	for k, v := range m.gauges {
		snap.Gauges[k] = v
	}
	return snap
}

// MetricsSnapshot is a point-in-time copy of an InMemoryMetrics. Entries are
// keyed by SeriesKey (metric name plus serialized attributes).
type MetricsSnapshot struct {
	// Counters maps a series key to its accumulated value.
	Counters map[string]int64
	// Histograms maps a series key to its aggregated distribution.
	Histograms map[string]HistogramSnapshot
	// Gauges maps a series key to its most recent value.
	Gauges map[string]float64
}

// HistogramSnapshot aggregates a single histogram series.
type HistogramSnapshot struct {
	// Count is the number of observations.
	Count int64
	// Sum is the sum of all observed values.
	Sum float64
	// Min is the smallest observed value.
	Min float64
	// Max is the largest observed value.
	Max float64
}

// SeriesKey builds the stable, attribute-ordered key used by InMemoryMetrics
// snapshots. Attributes are sorted by key (then value) so call order does not
// change the identity of a series.
func SeriesKey(name string, attrs ...Attr) string {
	if len(attrs) == 0 {
		return name
	}
	sorted := make([]Attr, len(attrs))
	copy(sorted, attrs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Key != sorted[j].Key {
			return sorted[i].Key < sorted[j].Key
		}
		return sorted[i].Value < sorted[j].Value
	})
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('|')
	for i, a := range sorted {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value)
	}
	return b.String()
}

// Instrument returns a middleware that records HTTP request counters, errors,
// and latency against metrics. A nil Metrics is a no-op.
func Instrument(metrics Metrics) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		if metrics == nil {
			return base
		}
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			path := normalizePath(req.URL.Path)
			start := time.Now()
			resp, err := base.RoundTrip(req)
			duration := time.Since(start)

			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			statusAttr := Attr{Key: "status", Value: strconv.Itoa(status)}
			reqAttrs := []Attr{{Key: "method", Value: req.Method}, {Key: "path", Value: path}, statusAttr}
			metrics.Counter(req.Context(), MetricHTTPRequests, 1, reqAttrs...)
			metrics.Histogram(req.Context(), MetricHTTPDuration, float64(duration.Nanoseconds())/1e6,
				Attr{Key: "method", Value: req.Method}, Attr{Key: "path", Value: path})
			if status == 429 {
				metrics.Counter(req.Context(), MetricRateLimit429s, 1,
					Attr{Key: "method", Value: req.Method}, Attr{Key: "path", Value: path})
			}
			if strings.HasPrefix(path, "/v1/api/order") {
				metrics.Histogram(req.Context(), MetricOrderLatencyMS, float64(duration.Nanoseconds())/1e6,
					Attr{Key: "method", Value: req.Method})
			}
			if err != nil || status >= 400 {
				class := "http"
				if err != nil {
					class = "transport"
				}
				errAttrs := append(reqAttrs, Attr{Key: "error.class", Value: class})
				metrics.Counter(req.Context(), MetricHTTPErrors, 1, errAttrs...)
			}
			return resp, err
		})
	}
}

// incrCounter increments a counter when metrics is non-nil. The caller's
// context is forwarded to the sink so downstream exporters can honor deadlines
// and cancellation.
func incrCounter(ctx context.Context, metrics Metrics, name string, delta int64, attrs ...Attr) {
	if metrics == nil {
		return
	}
	metrics.Counter(ctx, name, delta, attrs...)
}

// observeHistogram records a histogram value when metrics is non-nil.
func observeHistogram(ctx context.Context, metrics Metrics, name string, value float64, attrs ...Attr) {
	if metrics == nil {
		return
	}
	metrics.Histogram(ctx, name, value, attrs...)
}

// setGauge records a gauge value when metrics is non-nil.
func setGauge(ctx context.Context, metrics Metrics, name string, value float64, attrs ...Attr) {
	if metrics == nil {
		return
	}
	metrics.Gauge(ctx, name, value, attrs...)
}
