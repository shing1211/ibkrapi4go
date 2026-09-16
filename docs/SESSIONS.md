# Sessions

The IBKR session is a state machine. Callers never manage it directly; the SDK
runs a tickle heartbeat and surfaces state transitions.

## State machine

```
        NewClient / NewSession
                 │
                 ▼
          ┌─────────────┐
          │ DISCONNECTED│
          └──────┬──────┘
   Initialize()  │
                 ▼
          ┌─────────────┐   auth error
          │ INITIALIZING├──────────────┐
          └──────┬──────┘              │
                 │ ssodh/init ok       │
                 ▼                     │
          ┌─────────────┐              │
          │AUTHENTICATED│◀─────────────┘
          └──────┬──────┘
     tickle 401  │  Logout()/Close()
        ┌────────┴────────┐
        ▼                 ▼
  ┌───────────┐    ┌───────────┐
  │  EXPIRED  │    │  CLOSED   │
  └─────┬─────┘    └───────────┘
        │ Initialize() again
        └──────▶ INITIALIZING
```

## Heartbeat (tickle)

- Endpoint: `POST /v1/api/tickle`.
- Interval: 60 seconds, started on entering `AUTHENTICATED`.
- The tickle goroutine is owned by the `Session`; it exits when the session
  leaves `AUTHENTICATED` or the client closes.
- Consecutive tickle failures (default 2) transition to `EXPIRED`.
- A successful tickle refreshes the session token held in memory.

```go
type SessionState int
const (
    StateDisconnected SessionState = iota
    StateInitializing
    StateAuthenticated
    StateExpired
    StateClosed
)
```

## Concurrency contract

- `Session` is safe for concurrent use; state is guarded by a mutex.
- `Initialize` is idempotent when already `AUTHENTICATED` (returns nil).
- `Close` stops the tickle goroutine and sends `POST /v1/api/logout`. It is
  idempotent.
- All state changes are observable via `Status(ctx)`.

## Timing

| Parameter | Default | Configurable |
|-----------|--------:|--------------|
| Tickle interval | 60s | `WithTickleInterval` |
| Tickle timeout | 10s | `WithRequestTimeout` |
| Consecutive failures before EXPIRED | 2 | internal |

## Shutdown sequence

`Client.Close()`:

1. Cancel the client context (stops in-flight requests).
2. Stop the tickle goroutine.
3. Best-effort `POST /v1/api/logout` with a short timeout.
4. Close the WebSocket connection if open.
5. Release the rate limiter.

Steps 3–4 are best-effort; `Close` returns the first non-nil error.

## Testing

The session state machine is unit-testable with an `httptest` server
(see [TESTING.md](./TESTING.md)). Required cases:

- happy path DISCONNECTED → AUTHENTICATED
- tickle failure → EXPIRED
- re-initialize from EXPIRED
- `Close` idempotency
- goroutine leak check (`goleak`) after `Close`
