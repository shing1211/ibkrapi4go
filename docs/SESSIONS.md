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

- `Session` is safe for concurrent use; state is stored as `int32` and accessed
  via `sync/atomic`.
- `Initialize` is idempotent when already `AUTHENTICATED` (returns nil without
  calling the API).
- `Close` stops the tickle goroutine and sends `POST /v1/api/logout`. It is
  idempotent.
- Current state is observable via `Session.State()` (returns `SessionState`).

## Timing

| Parameter | Default | Configurable |
|-----------|--------:|--------------|
| Tickle interval | 60s | `SessionConfig.TickleInterval` |
| Request timeout | 30s | `SessionConfig.RequestTimeout` |
| Consecutive failures before EXPIRED | 2 | internal |

## Shutdown sequence

`Client.Close()`:

1. Transition state to `CLOSED`.
2. Signal tickle goroutine to stop; wait for it to exit.
3. Best-effort `POST /v1/api/logout` with the session's request timeout.

Step 3 is best-effort; `Close` returns the first non-nil error.

## Testing

The session state machine is unit-tested with a `fakeAPI` struct that
implements the `api` interface, bypassing the network. See
[TESTING.md](./TESTING.md). Required cases:

- happy path DISCONNECTED → AUTHENTICATED
- tickle failure → EXPIRED
- re-initialize from EXPIRED
- `Close` idempotency
- goroutine leak check (`goleak`) after `Close` — session tests use a 50ms
  post-TestMain sleep to allow the scheduler to reap exited tickle goroutines
