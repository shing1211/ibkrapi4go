# WebSocket Shutdown Reliability

- **Run:** `docs/runs/2026-09-24-ws-shutdown/`
- **Base commit:** `945182e`
- **Mode:** BUILD
- **Status:** complete

## Objective

Remove the WebSocket shutdown hang that blocked reliable Windows test runs while
preserving sequence-gap detection, reconnect/resubscribe behavior, and the
public API.

## Root cause

The D4 sequence-gap change added `lastUpdated` to `WSConn` but did not initialize
it. The first market-data frame containing `_updated` therefore panicked while
holding `WSConn.mu`; the goroutine recovery left the mutex locked. `Close` then
blocked in `ActiveSubscriptions`.

Explicit close also used a synchronous graceful WebSocket close while the reader
was blocked in `Read`, and reconnect could publish a newly dialed connection
after shutdown had begun.

## Tasks

| ID | Objective | Status | Acceptance |
|----|-----------|--------|------------|
| W1 | Reproduce the sequence and close failures | done | Focused tests fail before the fix and pass after it |
| W2 | Fix sequence tracking, shutdown, and reconnect publication | done | No locked-mutex panic; close interrupts I/O; late connections close |
| W3 | Add Windows-oriented regression coverage | done | Silent-peer, sequence, public close, and repeated-run tests pass |
| W4 | Sync docs, release metadata, and mirrors | done | Changelog/run docs updated; release mirrors synchronized |
| W5 | Make `money-check` precise and green | done | Struct-aware scanner passes with a seeded-field self-test |

## Constraints

- No public API signature changes.
- No generated-client edits.
- No new dependencies.
- No order mutation retry behavior changes.

## Verification

- `go test ./... -count=1 -timeout=15m` — pass
- Focused WebSocket tests repeated 20 times — pass
- `go vet ./...` — pass
- `go test -race` could not run locally because `CGO_ENABLED=0` and no C compiler
  is installed; CI remains the authoritative race check
