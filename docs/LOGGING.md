# Logging

The SDK emits structured logs through the standard library
[`log/slog`](https://pkg.go.dev/log/slog) package. There is no third-party
logging dependency ([ADR 0004](./adr/0004-minimal-dependencies.md)).

## Configuration

Logging is opt-in and configured on the client with `WithLogger`:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

c, err := ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithLogger(logger),
)
```

The environment variable `IBKR_LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
builds a text logger on `stderr` when no logger option is supplied. Options
always win over the environment (see [CONFIG.md](./CONFIG.md)).

### Nil-safety contract

**A logger is never nil after construction.** When no logger is configured, the
SDK substitutes a no-op logger writing to `io.Discard`:

```go
slog.New(slog.NewTextHandler(io.Discard, nil))
```

Every component (`Session`, `WSConn`, `TokenSource`, `Limiter`, `Breaker`)
applies this default in its constructor, so call sites never nil-check a
logger. Passing `nil` to `WithLogger` is treated the same as not setting it.

## Message scheme

Messages use a `ibkr.<subsystem> <event>` prefix so log streams can be filtered
by subsystem.

| Prefix | Subsystem | Examples |
|--------|-----------|----------|
| `ibkr.http` | Request middleware (`internal/observability.go`) | `request`, `error`, `failed` |
| `ibkr.session` | Session state machine and tickle | `initializing`, `authenticated`, `tickle failure`, `expired after tickle failures`, `closing` |
| `ibkr.ws` | WebSocket connection lifecycle | `reconnect failed`, `write failed`, `ping failed` |
| `ibkr.oauth` | OAuth2 token acquisition | `token refreshed`, `token refresh failed` |
| `ibkr.ratelimit` | Limiter wait diagnostics | `wait` (only when a wait exceeds 100ms) |
| `ibkr.breaker` | Circuit breaker transitions | `open`, `half-open`, `closed` |
| `ibkr.error` | Decoded `*ibkr.Error` | `ibkr.error` |
| `ibkr.config` | Configuration warnings | `insecure TLS skip-verify enabled for non-loopback host` |

## Levels

| Level | What is logged |
|-------|----------------|
| `Debug` | Successful HTTP requests (`method`, normalized `path`, `status`, `duration`, `request_id`); rate-limiter waits over 100ms; successful OAuth token refreshes |
| `Info` | Session lifecycle (`initializing`, `authenticated`, `closing`); circuit breaker `half-open` and `closed` |
| `Warn` | Transport failures and HTTP status ≥ 400; session `auth_status`/tickle failures; circuit breaker `open`; OAuth refresh failures; insecure TLS against a non-loopback gateway |
| `Error` | Not emitted directly; decoded errors surface at `Warn` via `ibkr.error` |

## Redaction

Tokens, cookies, and `Authorization` headers are never logged. Request and
response bodies are never logged; error text is passed through the redaction
helper before it reaches a handler. See [SECURITY.md](../SECURITY.md).

## Examples

Filter to a single subsystem with a `slog.Handler` wrapper, or by inspecting the
message prefix:

```go
h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
slog.SetDefault(slog.New(h))

// Debug logs now include per-request diagnostics:
//   level=DEBUG msg="ibkr.http request" method=GET path=/v1/api/portfolio/{}/summary status=200
```

Run at `debug` while diagnosing an issue and drop to `warn` in production to
keep only failures and warnings.

## Related

- [CONFIG.md](./CONFIG.md) — options, environment variables, precedence.
- [ERRORS.md](./ERRORS.md) — error envelope and retry semantics.
- [RATE-LIMITING.md](./RATE-LIMITING.md) — `ibkr.ratelimit` waits.
- [SESSIONS.md](./SESSIONS.md) — `ibkr.session` lifecycle.
- [STREAMING.md](./STREAMING.md) — `ibkr.ws` lifecycle.
