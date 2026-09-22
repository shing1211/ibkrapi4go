# Design 01 — Transport

The transport is the single path through which every HTTP request flows. It is an
`http.RoundTripper` chain so it composes with `net/http` and is testable.

## Chain

```
request
  └─ requestID → userAgent → auth → logging/telemetry → Instrument
       → circuitBreaker → retry (safe methods only) → rateLimit
       → timeout → errorDecode → UserMiddleware → http.Transport.Do
response / *ibkr.Error
```

Order matters: auth runs before retry so every attempt reuses the same
credential; retry runs before rate limiting so each attempt is paced; error
decoding runs before `UserMiddleware` (innermost custom layer); `Instrument`
(metrics) sits between logging and the circuit breaker. Assembled by
`internal.NewClientTransport`, configured via `TransportConfig`.

## Interfaces

```go
type RoundTripper interface {
    RoundTrip(*http.Request) (*http.Response, error)
}

type Middleware func(next RoundTripper) RoundTripper
```

## Responsibilities

| Layer | Does |
|-------|------|
| requestID | attaches `X-request-id` (lowercase, UUID) and stores it in context for error correlation |
| userAgent | sets `User-Agent` (`ibkrapi4go/<version>`) if unset |
| auth | injects `Authorization: Bearer <token>` from the credential provider |
| rateLimit (per-endpoint) | token bucket keyed by `METHOD templatedPath` |
| rateLimit (global) | client-wide token bucket |
| retry | exponential backoff + jitter; honoring `Retry-After`; safe methods only |
| errorDecode | maps status + IBKR envelope to `*ibkr.Error` |

## Rules

- Never modify the caller's request body for retries without buffering it first
  (safe methods have no body; unsafe methods are not retried, which sidesteps this).
- Always honor the request context; cancellation must abort in-flight I/O.
- Redact `Authorization`, `Cookie`, and token-bearing headers from all logs/errors.
- The generated client is given this `http.Client`; it does not build its own.

## Testing

- `httptest.Server` captures requests; assert headers, ids, and ordering.
- Rate limiter uses an injectable clock.
- Retry tests use a server that fails N times then succeeds, asserting attempt
  count and that unsafe methods are single-attempt.
