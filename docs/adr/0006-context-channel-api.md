# 0006 — Context-aware, channel-based API; use `coder/websocket`

- Status: Accepted
- Date: 2026-09-16

## Context

Real-time data is central to an IBKR client. Go's concurrency primitives map well
to streaming: updates as values on channels, cancellation via `context.Context`.

Candidate WebSocket libraries:

- `gorilla/websocket` — most widely used, but archived (2022), weaker
  `context.Context` story, requires a pong callback before `Ping`, concurrent
  writes must be serialized by the caller.
- `coder/websocket` (formerly `nhooyr.io/websocket`) — actively maintained,
  first-class `context.Context`, safe concurrent writes, zero dependencies.
- `gobwas/ws` / `lxzan/gws` — high raw throughput, lower-level/event-driven;
  performance is not the bottleneck for a client talking to IBKR.

## Decision

- Use `github.com/coder/websocket`.
- Expose streaming as channel-based `Subscription` values with `Updates()` and
  `Errors()` channels, cancelled by context, per [../STREAMING.md](../STREAMING.md).

## Consequences

- Idiomatic consumer code (`for select`).
- No manual concurrent-write locking.
- Raw throughput is lower than `gws`/`gobwas`, which is irrelevant here.
- A bounded buffer with a documented drop policy is required to avoid head-of-line
  blocking across subscriptions.
