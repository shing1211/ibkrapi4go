# 0017 — Logging Interface

- Status: Accepted
- Date: 2026-09-18

## Context

The SDK uses the standard library `log/slog` for structured logging. This
provides a consistent, dependency-free logging interface. This ADR documents
which subsystems log, what is logged, and what is redacted.

## Decision

### Logger configuration

Callers inject a `*slog.Logger` via `ibkr.WithLogger`:

```go
cli, _ := ibkr.NewClient(
    ibkr.WithLogger(slog.New(slog.NewJSONHandler(os.Stderr, nil))),
)
```

If no logger is provided, a no-op logger is used and no log output is produced.

### Subsystems

Logging is scoped to specific subsystems. Each subsystem uses a distinct
`slog.Logger` with a subsystem-specific key:

| Subsystem | Logger key | What it logs |
|-----------|------------|--------------|
| Client construction | `ibkr.config` | Option application, env var fallbacks, config errors |
| Session / tickle | `ibkr.session` | State transitions, heartbeat failures, reauthentication |
| Transport | `ibkr.transport` | Request/response headers (not bodies), latency, retry attempts |
| Rate limiter | `ibkr.ratelimit` | Wait time when rate-limited, global vs per-endpoint limits |
| Circuit breaker | `ibkr.breaker` | State transitions (closed/open), half-open probes |
| WebSocket | `ibkr.ws` | Connect/disconnect, message count, dropped update count |
| OAuth2 token | `ibkr.oauth` | Token acquisition, refresh, and failure (token values redacted) |

### What is logged

**Logged at Info level:**
- Client lifecycle events (created, closed).
- Session state transitions (`Disconnected → Initializing → Authenticated`).
- WebSocket connect/disconnect.
- Circuit breaker state changes.
- Rate limiter wait events (when a request is delayed).

**Logged at Warn level:**
- Non-terminal errors on best-effort operations.
- Retries on safe methods.
- OAuth2 token refresh failures (with reason, not token values).

**Logged at Error level:**
- Terminal errors returned to callers.
- Session heartbeat failures.
- WebSocket reconnection failures.

### What is NOT logged

The following are **never logged**, even at debug level:

- Request or response bodies (may contain account data, positions, orders).
- OAuth2 access tokens or refresh tokens (redacted to `[REDACTED]`).
- Passwords or PINs (not used by this SDK, but the policy stands).
- `sess` cookie values.
- `Authorization` header values (redacted to `[REDACTED]`).

### DebugInfo()

`Client.DebugInfo()` returns a human-readable snapshot of the client's
current state: gateway URL, session state, active subscriptions, and internal
counters. This is intended for bug reports and is safe to include in issue
tickets — it contains no credentials or sensitive data.

## Consequences

- Callers using `slog` can route `ibkr.*` subsystem logs to appropriate
  handlers.
- No credential leakage in normal log output.
- The `DebugInfo()` output can be shared publicly in bug reports.

## Alternatives considered

- **Custom logger interface**: Rejected — `slog.Logger` is the Go standard and
  sufficient.
- **Log levels as constants**: Already handled by `slog.Level`.
