# Design 05 — Streaming

Implementation-level design for the WebSocket layer. Protocol and usage are in
[../STREAMING.md](../STREAMING.md); this file covers internals.

## Components

```
wsConn (internal/)
 ├─ reader goroutine   : reads frames, routes by conid → dispatcher
 ├─ writer goroutine   : serializes outbound frames (subscribe/unsubscribe/ping)
 ├─ ping goroutine     : 30s ping, 10s pong deadline
 └─ reconnect loop     : backoff, re-subscribe

Subscription (public)
 ├─ Updates() <-chan Update
 ├─ Errors()  <-chan error
 └─ Close()
```

## Multiplexing

One connection, many subscriptions. Outbound frames carry a client-chosen `id`;
the connection tracks `id → pending request` to correlate acks/errors. Inbound
data frames are routed by `conid` to the matching subscription channels.

## Subscription lifecycle

1. `Subscribe(ctx, conids, fields)` validates limits, allocates a request id,
   registers the subscription, and sends the frame.
2. `Close()` sends `unsubscribe`, removes the subscription, and closes channels.
3. `ctx` cancellation is equivalent to `Close()`.

## Concurrency

- Writes are serialized by the writer goroutine; callers never write directly.
  (`coder/websocket` also permits concurrent writes, but serializing keeps
  ordering deterministic.)
- Channel sends use a select on `ctx.Done()` to avoid blocking after cancellation.
- Channels are closed exactly once (guarded by `sync.Once`).

## Buffer & drop policy

Per-subscription buffer default 256. On full buffer, drop the **oldest** update
and increment `dropped` (see [../STREAMING.md](../STREAMING.md)). The reader never
blocks on a slow consumer.

## Reconnect

- Triggered by read/write error or ping timeout.
- Backoff: 1,2,4,8,16,30s cap, with jitter.
- On success: re-issue all active subscriptions with fresh ids.
- Emits a reconnected notice on each subscription's `Errors()` channel; a gap is
  possible and is the caller's responsibility to reconcile (e.g. re-snapshot).

## Testing

- Mock gateway WebSocket hub (`internal/mockgateway`) behind `httptest`.
- Reconnect test: server drops once, asserts re-subscription.
- Overflow test: slow consumer, asserts oldest-dropped and counter.
- `goleak` after `Close`.
