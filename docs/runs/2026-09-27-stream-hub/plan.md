# Plan: the stream hub's fan-out, and a test that could not be written

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `3573421` (`v1.1.20`)

## Objective

Eighth request to "plan and implement remaining blocking items", with the same two
facts as the previous run: the credential-blocked items cannot move, and the
substantive offline backlog is empty.

The one gap left that would actually tell us something was `stream.go` - 73%, with
`closeAll` and `Broadcast` uncovered. Both fan out across every connected client, so
a defect there is a defect every client sees.

## What the code does

Both take the same shape:

```go
h.mu.Lock()
conns := snapshot(h.conns)
h.mu.Unlock()

for _, sc := range conns { /* act */ }
```

Snapshot under the lock, act outside it. That ordering is the point: `sendFrame`
writes to a socket, and holding the hub mutex across the write would let one
unresponsive client stall every other client and every hub operation.

Two behaviours are silent rather than loud, and both are easy to read wrong:

- `sendFrame` returns **nil** for an already-closed connection, so it is skipped
  without error. `Broadcast` counts a connection when `sendFrame` returns nil, so
  its return value is an **attempt count, not a delivery count**.
- `closeAll` marks connections closed and calls `CloseNow`, but does **not** remove
  them. `CloseNow` unblocks `serveWS`, whose **defer** calls `remove` - so the hub
  drains as a consequence of closing, not inside `closeAll`.

## What this run does, and does not, deliver

Three behaviours are observable and are now tested. One is not, and the reason is
the more useful result - see the report.
