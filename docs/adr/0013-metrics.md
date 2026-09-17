# 0013 — Dependency-free metrics interface

- Status: Accepted
- Date: 2026-09-17

## Context

The SDK already provides request lifecycle hooks via a small `Telemetry`
interface ([ADR 0004](./0004-minimal-dependencies.md) keeps observability free of
third-party dependencies). Telemetry covers spans/tracing callbacks, but users
also need numeric metrics: request counts and latency, order outcomes, rate-limit
waits, circuit-breaker state, WebSocket connects, and OAuth token refreshes.

Importing the OpenTelemetry SDK would pull a large dependency tree into every
consumer ([ADR 0004](./0004-minimal-dependencies.md) forbids new runtime
dependencies without an ADR). The metrics surface must therefore be defined by
the SDK and *bridged* by the user to whichever backend they use.

## Decision

- Define a minimal, OTel-shaped `Metrics` interface in `internal/metrics.go`
  with three methods — `Counter.Add`, `Histogram.Record`, and `Gauge.Record`
  equivalents — carrying a `context.Context` and variadic `Attr` key/value
  pairs. It has **no** OpenTelemetry import.
- Ship a dependency-free `InMemoryMetrics` implementation (mutex-guarded maps
  plus a `Snapshot`) for tests and simple applications, and a `NopMetrics`
  no-op. A nil `Metrics` is always a safe no-op.
- Instrument the SDK's existing components rather than adding a parallel
  pipeline: HTTP transport (`Instrument` middleware), circuit breaker, rate
  limiter, OAuth token source, WebSocket, and the order manager.
- Emit a **fixed, documented set of metric names** (`ibkr.*`) with
  **low-cardinality attributes only**. Attribute values must never include PII,
  account ids, conids, order ids, or tokens.
- Expose the interface, the in-memory implementation, the metric-name
  constants, and a `WithMetrics` client option from `pkg/ibkr`. Generated code
  is untouched.

## Consequences

- Users bridge manually: implement `ibkr.Metrics` over the OTel SDK
  (`Counter.Add` / `Histogram.Record` / `Gauge.Record`) or any other backend.
  An example lives in [OBSERVABILITY.md](../OBSERVABILITY.md).
- The SDK carries zero metric dependencies and stays within the dependency
  budget of ADR 0004.
- Metric reporting must not block or panic; implementations are documented as
  concurrent-safe and non-blocking.
- `InMemoryMetrics` is intended for tests and small apps, not as an unbounded
  production store.
