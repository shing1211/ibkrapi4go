# Streaming (WebSocket)

Real-time data arrives over the WebSocket endpoint `GET /v1/api/ws` on the
Client Portal Gateway. The SDK wraps it in a channel-based, context-aware API
backed by [`coder/websocket`](https://github.com/coder/websocket).

## Why `coder/websocket`

See [ADR 0006](./adr/0006-context-channel-api.md). Summary: first-class
`context.Context`, safe concurrent writes, zero dependencies. `gorilla/websocket`
was rejected (archived, weaker context story).

## Protocol

Connect, then send JSON subscribe frames:

```json
{ "id": 1, "method": "subscribe", "params": { "conids": [265598], "fields": ["31","84","86"] } }
```

Unsubscribe:

```json
{ "id": 2, "method": "unsubscribe", "params": { "conids": [265598] } }
```

`id` is a monotonically increasing request id chosen by the client. Updates are
JSON objects keyed by `conid` and field code.

### Field codes (common)

| Code | Meaning | Code | Meaning |
|-----:|---------|-----:|---------|
| 31 | Last price | 84 | Bid |
| 86 | Ask | 83 / 88 | Bid / ask size |
| 70 | High | 71 | Low |
| 87 | Volume | 6509 | Company name |

Field codes are IBKR-defined; the SDK exposes them as typed constants but does
not reinterpret values.

## Channel API (planned)

```go
sub, err := cli.MarketData().Subscribe(ctx, []ibkr.ConID{265598}, ibkr.Fields{ibkr.FieldLast, ibkr.FieldBid, ibkr.FieldAsk})
if err != nil { return err }
defer sub.Close()

for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case u, ok := <-sub.Updates():
        if !ok { return nil } // stream closed
        fmt.Printf("%d %s\n", u.ConID, u.Field)
    case err := <-sub.Errors():
        log.Println("stream error:", err)
    }
}
```

Design rules:

- `Subscribe` returns a `*Subscription` with `Updates()` and `Errors()` channels.
- Channels are closed (not just abandoned) on close/error.
- `Subscription.Close()` is idempotent and unsubscribes server-side.
- `ctx` cancellation closes the subscription.

## Connection management

- One WebSocket connection per `Client`, multiplexed across subscriptions.
- A reader goroutine dispatches frames to per-subscription channels; a writer
  goroutine serializes outbound frames (avoids concurrent-write issues).
- Heartbeat: ping every 30s; pong deadline 10s.

## Reconnect

On unexpected disconnect:

1. Backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s, with jitter.
2. Re-establish the connection.
3. Re-send active subscriptions with new request ids.
4. Emit a notice on `Errors()` so callers can react.

Reconnect does **not** guarantee gap-free data; consumers needing continuity must
resubscribe to snapshots.

## Limits

IBKR limits conids/fields per subscription and total subscriptions per session.
The SDK batches large requests and returns a typed error when a limit is
exceeded. Exact ceilings are account/entitlement dependent and are configurable
via `WithStreamingLimits`.

## Backpressure

Per-subscription channels are buffered (default 256). On overflow the SDK:

1. Logs a warning and increments a dropped-update counter, then
2. Drops the **oldest** update (keeps the connection alive).

This is a deliberate choice: dropping stale quotes is preferable to blocking the
reader and stalling all subscriptions. Callers needing every tick should use a
larger buffer or persist server-side.

## Testing

- Local WebSocket server: subscribe → receive → unsubscribe.
- Reconnect test with a server that drops the connection once.
- Buffer-overflow test asserts the drop policy and counter.
- `goleak` asserts no goroutines survive `Close()`.
