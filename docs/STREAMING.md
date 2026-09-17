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

| Code | Constant | Meaning | Code | Constant | Meaning |
|-----:|----------|---------|-----:|----------|---------|
| 31 | `FieldLastPrice` | Last price | 84 | `FieldBidPrice` | Bid |
| 86 | `FieldAskPrice` | Ask | 88 / 85 | `FieldBidSize`/`FieldAskSize` | Bid / ask size |
| 70 | `FieldHigh` | High | 71 | `FieldLow` | Low |
| 87 | `FieldVolume` | Volume | 82 / 83 | `FieldChange`/`FieldChangePercent` | Change / % |
| 55 | `FieldSymbol` | Symbol | | | |

Field codes are IBKR-defined; the SDK exposes them as typed constants but does
not reinterpret values.

## Channel API

```go
sub, err := cli.MarketData().Subscribe(ctx, []ibkr.ConID{265598},
    []ibkr.Field{ibkr.FieldLastPrice, ibkr.FieldBidPrice, ibkr.FieldAskPrice})
if err != nil { return err }
defer sub.Close()

for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case u, ok := <-sub.Updates():
        if !ok { return nil } // stream closed
        fmt.Printf("%d %s=%s\n", u.ConID, u.Field, u.Value)
    case err := <-sub.Errors():
        // connection-level event; use errors.Is(err, ibkr.ErrStreamReconnected)
        // or ibkr.ErrStreamDisconnected to distinguish reconnect notices.
        log.Println("stream error:", err)
    }
}
```

Design rules:

- `Subscribe` returns a `*Subscription` with `Updates()` and `Errors()` channels.
- One `Update` is emitted per field code in a frame; a conid's updates are
  delivered to every subscription that requested it.
- Channels are closed (not just abandoned) on close/error.
- `Subscription.Close()` is idempotent and unsubscribes server-side.
- `ctx` cancellation closes the subscription.
- `Subscription.Dropped()` reports updates dropped under backpressure.

## Connection management

- One WebSocket connection per `Client`, multiplexed across subscriptions.
- A reader goroutine dispatches frames to per-subscription channels; a writer
  goroutine serializes outbound frames (avoids concurrent-write issues).
- Heartbeat: ping every 30s; pong deadline 10s.

## Reconnect

On unexpected disconnect:

1. Emit `ErrStreamDisconnected` on every subscription's `Errors()` channel.
2. Backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s, with full jitter.
3. Re-establish the connection.
4. Re-send active subscriptions with new request ids.
5. Emit `ErrStreamReconnected` on each `Errors()` channel so callers can react.

Reconnect does **not** guarantee gap-free data; consumers needing continuity must
resubscribe to snapshots.

## Limits

IBKR limits conids/fields per subscription and total subscriptions per session.
The SDK enforces configurable ceilings and returns an `*Error` wrapping
`ErrStreamingLimit` when exceeded. Configure with `WithStreamingLimits`:

| Limit | Default |
|-------|--------:|
| `MaxConIDsPerRequest` | 100 |
| `MaxFieldsPerRequest` | 50 |
| `MaxSubscriptions` | 10 |
| `BufferSize` | 256 |
| `ReconnectBase` / `ReconnectMax` | 1s / 30s |

Exact IBKR ceilings are account/entitlement dependent.

## Backpressure

Per-subscription channels are buffered (default 256). On overflow the SDK:

1. Logs a warning and increments a dropped-update counter, then
2. Drops the **oldest** update (keeps the connection alive).

This is a deliberate choice: dropping stale quotes is preferable to blocking the
reader and stalling all subscriptions. Callers needing every tick should use a
larger buffer or persist server-side.

## Testing

- Mock gateway WebSocket hub ([MOCK-GATEWAY.md](./MOCK-GATEWAY.md)): subscribe →
  receive → unsubscribe.
- Reconnect test with a server that drops the connection once.
- Buffer-overflow test asserts the drop policy and counter.
- `goleak` asserts no goroutines survive `Close()`.
