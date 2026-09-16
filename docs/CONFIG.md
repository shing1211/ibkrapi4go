# Configuration

How the SDK is configured, and the order of precedence.

## Precedence

Highest wins:

1. **Functional options** passed to `NewClient` / managers.
2. **Environment variables** (see table).
3. **Defaults** baked into the SDK.

Environment variables are read once at construction. Options always override.

## Options

```go
ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithInsecureSkipVerify(true),          // local self-signed cert
    ibkr.WithRequestTimeout(15*time.Second),
    ibkr.WithTickleInterval(60*time.Second),
    ibkr.WithLogger(slog.Default()),
    ibkr.WithHTTPClient(custom *http.Client),
    ibkr.WithUserAgent("my-app/1.0"),
    ibkr.WithStreamingLimits(ibkr.StreamingLimits{
        MaxConIDsPerRequest: 100,
        MaxSubscriptions:    10,
        BufferSize:          256,
    }), // see STREAMING.md
    ibkr.WithRateLimit(10, 20),                 // per-endpoint rps, burst
    ibkr.WithGlobalRateLimit(50),               // client-wide rps
    ibkr.WithRetryPolicy(ibkr.DefaultRetryPolicy()), // safe methods only
    ibkr.WithTelemetry(myTracer),               // optional request hooks
    ibkr.WithCircuitBreaker(5, 30*time.Second), // optional, off by default
)
```

Rate limiting defaults to 10 req/s per endpoint and 50 req/s client-wide
([RATE-LIMITING.md](./RATE-LIMITING.md)). Retries apply to safe methods only and
default to 3 attempts ([ERRORS.md](./ERRORS.md)); `WithRetryPolicy` with
`MaxAttempts: 1` disables them. The circuit breaker is disabled by default.

Invalid configuration (for example a malformed gateway URL) is reported as a
`*ibkr.ConfigError`.

## Environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `IBKR_GATEWAY_URL` | Gateway base URL | `https://localhost:5000` |
| `IBKR_INSECURE_SKIP_VERIFY` | Skip TLS verify (localhost only) | `false` |
| `IBKR_TIMEOUT` | Request timeout (duration) | `15s` |
| `IBKR_TICKLE_INTERVAL` | Tickle heartbeat interval | `60s` |
| `IBKR_RATE_LIMIT` | Per-endpoint requests/second | `10` |
| `IBKR_GLOBAL_RATE_LIMIT` | Client-wide requests/second | `50` |
| `IBKR_LOG_LEVEL` | `debug`/`info`/`warn`/`error` | `info` |
| `IBKR_USER_AGENT` | Outbound User-Agent | module version |

Integration tests additionally read `IBKR_USERNAME` / `IBKR_PASSWORD` for the
human login step; these are **never** read by the SDK itself.

## TLS

The local gateway uses a self-signed certificate. `WithInsecureSkipVerify(true)`
(or `IBKR_INSECURE_SKIP_VERIFY=true`) disables verification and **must only be
used against `localhost`**. The SDK logs a warning when it is enabled against a
non-loopback host.

## Logging

The SDK accepts a `*slog.Logger`. When unset, it discards output. Tokens,
cookies, and `Authorization` headers are always redacted. See
[../SECURITY.md](../SECURITY.md).

## Timeouts

| Scope | Default | Notes |
|-------|--------:|-------|
| Per request | 15s | includes connection + response headers |
| Response body read | 30s | streaming reads use the caller's context |
| Dial / TLS handshake | 10s | via the transport |
| Logout (on Close) | 3s | best-effort |

Per-call overrides use `context.WithTimeout` at the call site. The SDK applies
`WithRequestTimeout` as a middleware when the caller's context has no deadline;
a caller-supplied deadline always wins.

## Validation

Configuration errors (bad URL, negative limits, non-loopback insecure TLS) are
returned from `NewClient` as a `*ibkr.ConfigError`, never as a panic.
