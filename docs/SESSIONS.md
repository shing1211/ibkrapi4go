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
- The tickle goroutine is owned by the `Session`; `Close` signals it and waits
  for it to exit, so no goroutine survives shutdown.
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

## Public surface

`Client.Session()` returns a `*SessionManager` (`pkg/ibkr/auth.go`):

```go
func (m *SessionManager) Initialize(ctx context.Context) error
func (m *SessionManager) Close(ctx context.Context) error
func (m *SessionManager) State() SessionState
func (m *SessionManager) Status(ctx context.Context) (*AuthStatus, error)
```

`State`/`AuthStatus` expose the `SessionState` values above. `Client.Close()` is
the usual shutdown entry point.

## Concurrency contract

- `Session` is safe for concurrent use; state is stored as `int32` and accessed
  via `sync/atomic`.
- `Initialize` is idempotent when already `AUTHENTICATED` (returns nil without
  calling the API).
- `Close` stops the tickle goroutine and sends `POST /v1/api/logout`. It is
  idempotent.
- Current state is observable via `SessionManager.State()` (returns `SessionState`).

## Timing

| Parameter | Default | Configurable |
|-----------|--------:|--------------|
| Tickle interval | 60s | `WithTickleInterval` |
| Request timeout | 15s | `WithRequestTimeout` |
| Consecutive failures before EXPIRED | 2 | internal |

## Shutdown sequence

`Client.Close()` (no context; uses an internal 3s logout budget):

1. Mark the client closed; further manager calls return `ErrClosed`.
2. Transition session state to `CLOSED`.
3. Signal the tickle goroutine to stop and wait for it to exit.
4. Best-effort `POST /v1/api/logout`.

Step 4 is best-effort; `Close` is idempotent and returns the first non-nil error.

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
