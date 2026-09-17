# Observability

The SDK reports numeric metrics through a tiny, dependency-free interface. It is
shaped to bridge directly onto [OpenTelemetry](https://opentelemetry.io/)
instruments but carries **no** OpenTelemetry dependency
([ADR 0013](./adr/0013-metrics.md), [ADR 0004](./adr/0004-minimal-dependencies.md)).
Tracing-style callbacks live separately in [LOGGING.md](./LOGGING.md) and the
`Telemetry` interface.

## Installation

Install a sink once at client construction with `WithMetrics`:

```go
m := ibkr.NewInMemoryMetrics()

c, err := ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithMetrics(m),
)
```

A nil sink (the default) disables metrics. Every subsystem reports to the same
sink: the HTTP transport, circuit breaker, rate limiter, OAuth token source,
WebSocket, and the order manager.

## Metric names

All names are stable and prefixed `ibkr.`. Attributes are deliberately
**low-cardinality**: never account ids, conids, order ids, or tokens.

| Name | Type | Unit | Attributes |
|------|------|------|------------|
| `ibkr.http.requests` | counter | 1 | `method`, `path` (normalized), `status` |
| `ibkr.http.errors` | counter | 1 | `method`, `path`, `status`, `error.class` (`transport` or `http`) |
| `ibkr.http.request.duration_ms` | histogram | milliseconds | `method`, `path` |
| `ibkr.orders.submitted` | counter | 1 | — |
| `ibkr.orders.confirmed` | counter | 1 | — |
| `ibkr.orders.modified` | counter | 1 | — |
| `ibkr.orders.cancelled` | counter | 1 | — |
| `ibkr.orders.rejected` | counter | 1 | — |
| `ibkr.ratelimit.waits` | counter | 1 | — |
| `ibkr.ratelimit.wait_ms` | histogram | milliseconds | — |
| `ibkr.breaker.state` | gauge | state | `state` (`closed`=0, `half-open`=1, `open`=2) |
| `ibkr.ws.connects` | counter | 1 | — |
| `ibkr.ws.reconnects` | counter | 1 | — |
| `ibkr.oauth.token.refreshes` | counter | 1 | — |
| `ibkr.oauth.token.failures` | counter | 1 | — |

The same names are exported as constants (`ibkr.MetricHTTPRequests`,
`ibkr.MetricOrdersSubmitted`, …) for discoverability.

## In-memory example

`InMemoryMetrics` is a mutex-guarded sink for tests and simple applications. It
is safe for concurrent use and never blocks callers beyond a short lock.

```go
m := ibkr.NewInMemoryMetrics()
cli, _ := ibkr.NewClient(ibkr.WithGatewayURL(url), ibkr.WithMetrics(m))
// ... make calls ...
snap := m.Snapshot()
fmt.Println(snap.Counters[ibkr.SeriesKey(ibkr.MetricHTTPRequests,
    ibkr.Attr{Key: "method", Value: "GET"},
    ibkr.Attr{Key: "path", Value: "/v1/api/portfolio/{}/summary"},
    ibkr.Attr{Key: "status", Value: "200"})])
```

Snapshot keys are built with `SeriesKey`, which sorts attributes so call order
does not change a series' identity. Histogram snapshots expose `Count`, `Sum`,
`Min`, and `Max`; gauges keep the most recent value.

## OpenTelemetry bridge

Implement `ibkr.Metrics` over any backend. The snippet below is documentation
only — the SDK itself never imports OpenTelemetry. Counters are cached by name;
histograms and gauges follow the same pattern.

```go
import (
    "context"
    "sync"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"

    "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// OTelMetrics bridges ibkr.Metrics onto OpenTelemetry instruments.
type OTelMetrics struct {
    meter      metric.Meter
    mu         sync.Mutex
    counters   map[string]metric.Int64Counter
    histograms map[string]metric.Float64Histogram
    gauges     map[string]metric.Float64Gauge
}

func NewOTelMetrics(meter metric.Meter) *OTelMetrics {
    return &OTelMetrics{
        meter:      meter,
        counters:   map[string]metric.Int64Counter{},
        histograms: map[string]metric.Float64Histogram{},
        gauges:     map[string]metric.Float64Gauge{},
    }
}

func otelAttrs(attrs []ibkr.Attr) []attribute.KeyValue {
    out := make([]attribute.KeyValue, 0, len(attrs))
    for _, a := range attrs {
        out = append(out, attribute.String(a.Key, a.Value))
    }
    return out
}

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
    c.Add(ctx, delta, metric.WithAttributes(otelAttrs(attrs)...))
}

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
    h.Record(ctx, value, metric.WithAttributes(otelAttrs(attrs)...))
}

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
    g.Record(ctx, value, metric.WithAttributes(otelAttrs(attrs)...))
}

// Wire it in:
//   metricMeter := otel.Meter("ibkrapi4go")
//   cli, _ := ibkr.NewClient(ibkr.WithGatewayURL(url), ibkr.WithMetrics(NewOTelMetrics(metricMeter)))
```

## Benchmarks

`pkg/ibkr/benchmark_test.go` and `internal/benchmark_test.go` contain `testing.B`
benchmarks for all hot paths:

| Benchmark | What it measures |
|-----------|-----------------|
| `Benchmark*JSON/Marshal` | `json.Marshal` throughput per public type (ns/op, MB/s) |
| `Benchmark*JSON/Unmarshal` | `json.Unmarshal` throughput per public type (ns/op, MB/s) |
| `BenchmarkHTTP*` | Full round-trip HTTP latency against mock gateway (p50/p95/p99) |
| `BenchmarkWSSubscribeUnsubscribe` | WS subscribe + receive one tick + unsubscribe (ms/op) |
| `BenchmarkSessionInit` | Cold session init: `NewClient` + `Session.Initialize` (ms/op) |
| `BenchmarkWSMessageDecode/Dispatch` | WS message decode and dispatch (ns/op) |

Baseline numbers are stored in `benchmark.baseline` (root). The CI benchmark job
(`.github/workflows/ci.yml`) compares every run against this baseline and fails
on >10% regression.

```bash
# Run benchmarks
go test -bench=. ./pkg/ibkr/... ./internal/... -benchtime=5s -benchmem

# Capture a new baseline
go test -bench=. -json ./pkg/ibkr/... ./internal/... > benchmark.baseline

# Compare against baseline
go run scripts/bench_compare.go /tmp/bench_current.json benchmark.baseline
```

## Fuzz Tests

`pkg/ibkr/fuzz_test.go` and `internal/fuzz_test.go` contain `testing.F` fuzz tests:

- **JSON decode fuzzing** (47 `Fuzz*` functions): every major public `pkg/ibkr` type
  is fuzzed with valid JSON mutations (byte-flip, deletion, truncation, swap,
  duplication) and invalid inputs to verify clean errors — never panics.
- **Response decode fuzzing** (all 185 op fixtures): each mock gateway fixture is
  mutated across all 185 operations and decoded through the real SDK response path.
- **Money/quantity round-trip** (`FuzzMoneyQuantityRoundTrip`): ADR 0008 compliance —
  decimal strings survive encode/decode without precision loss.

```bash
# Run fuzz tests (30s seed)
go test ./pkg/ibkr/... ./internal/... -fuzz=. -fuzztime=30s

# Verify no panics (fast)
go test ./pkg/ibkr/... ./internal/... -test.run=Fuzz -test.fuzztime=10s
```

Corpus entries are written to `pkg/ibkr/testdata/fuzz/` and `internal/testdata/fuzz/`.

## Rules

- Implementations must be safe for concurrent use and must **not block**.
- A nil `Metrics` is a safe no-op; the SDK never nil-checks at call sites.
- Never put account ids, conids, order ids, or tokens in attribute values
  ([SECURITY.md](../SECURITY.md)).

## Related

- [LOGGING.md](./LOGGING.md) — structured logs and `Telemetry` hooks.
- [CONFIG.md](./CONFIG.md) — client options and precedence.
- [RATE-LIMITING.md](./RATE-LIMITING.md) — `ibkr.ratelimit` waits.
- [STREAMING.md](./STREAMING.md) — `ibkr.ws` lifecycle.
- [ADR 0013](./adr/0013-metrics.md) — the metrics decision record.
- `benchmark.baseline` — baseline benchmark results for CI regression detection.
- `scripts/bench_compare.go` — pure-stdlib benchmark comparison tool.
