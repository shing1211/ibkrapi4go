# Design 08 — Concurrency and secrets

## Concurrency contract

| Type | Concurrent-safe? | Notes |
|------|:----------------:|-------|
| `Client` | ✅ | all state guarded |
| Managers (`Account`, `Portfolio`, `Trade`, `MarketData`) | ✅ | stateless over shared client |
| `Session` | ✅ | mutex-guarded state machine |
| `Subscription` | ✅ | channels + `sync.Once` close |
| `Limiter` | ✅ | mutex-guarded buckets |
| `*ibkr.Error` | ✅ | immutable after construction |

Rules:

- Public types are safe for concurrent use unless documented otherwise.
- No exported method panics on concurrent use.
- Background goroutines are owned by `Client`/`Session`/`wsConn` and stopped on
  `Close`.

## Goroutine ownership

| Goroutine | Started by | Stopped by |
|-----------|-----------|-----------|
| tickle loop | `Session.Initialize` | `Session.Close` / `Client.Close` |
| ws reader | first `Subscribe` | ws close |
| ws writer | first `Subscribe` | ws close |
| ws ping | ws connect | ws close |
| reconnect loop | ws disconnect | ws close or `Close` |

Every test that starts goroutines runs `goleak` to assert none survive `Close`.

## Cancellation

- Every public method takes `context.Context` first.
- Cancellation aborts in-flight I/O and pending rate-limit waits.
- `Client.Close` cancels the WebSocket's owned I/O context and signals session
  shutdown; background work must not retain a live socket reader.

## Secrets

- **Sources:** tokens come from the gateway session; no username/password is ever
  accepted. Config comes from options/env ([../CONFIG.md](../CONFIG.md)).
- **Storage:** tokens are held in memory only; never written to disk or logs.
- **Redaction:** the transport redacts `Authorization`, `Cookie`, and token-bearing
  headers from logs and from `*ibkr.Error.Message`.
- **TLS:** all requests use HTTPS. `WithInsecureSkipVerify(true)` is for
  `localhost` only; the SDK warns if enabled against a non-loopback host.

## Deadlock avoidance

- OAuth token invalidation uses generations; an in-flight result from an older
  generation is returned to its caller but never repopulates the cache.
- Never hold a mutex while performing I/O or blocking on a channel.
- Channel sends select on `ctx.Done()`.
- The ws reader never blocks on a consumer (bounded buffer + drop policy,
  [05-streaming.md](./05-streaming.md)).

## Race detection

CI runs `go test -race ./...`. Any data race is a release blocker.
