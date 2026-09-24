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
        // connection-level event; use errors.Is(err, ibkr.ErrWSReconnected)
        // or ibkr.ErrWSDisconnected to distinguish reconnect notices.
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

## Account and portfolio streaming

The same multiplexed connection also streams account values and portfolio
positions. Subscribe through the account/portfolio managers:

```go
acct, err := cli.Account().SubscribeAccount(ctx, nil)
if err != nil { return err }
defer acct.Close()

port, err := cli.Portfolio().SubscribePortfolio(ctx, nil)
if err != nil { return err }
defer port.Close()

for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case e := <-acct.AccountUpdates():
        fmt.Printf("account %s net=%s cash=%s\n", e.Account, e.NetLiquidity, e.Cash)
    case p := <-port.PortfolioUpdates():
        fmt.Printf("position %d qty=%s avg=%s mv=%s pnl=%s\n",
            p.Conid, p.Position, p.AvgCost, p.MarketValue, p.UnrealizedPNL)
    case err := <-acct.Errors():
        log.Println("stream error:", err)
    }
}
```

`SubscribeAccount` sends the gateway `account` method; `SubscribePortfolio`
sends `portfolio`. Both return a `*Subscription` that shares the connection,
reconnect, and backpressure behavior described below, and both honor the same
`MaxSubscriptions`/`MaxFieldsPerRequest` limits and `ctx` cancellation. Closing
the subscription unsubscribes server-side.

### Typed events

| Channel | Frame | Struct | Fields |
|---------|-------|--------|--------|
| `AccountUpdates()` | `acq` | `AccountUpdateEvent` | `Account`, `NetLiquidity`, `Cash`, `Equity`, `MaintMargin`, `Received` |
| `PortfolioUpdates()` | `pos` | `PortfolioEvent` | `Conid`, `Position`, `AvgCost`, `MarketValue`, `UnrealizedPNL`, `Received` |

A `pos` frame may carry a single position object or an array; each position is
emitted as a separate `PortfolioEvent`. Money and quantities are decimal strings
(ADR 0008).

Order status (`sor`), notifications (`ntf`), and user messages (`usr`) continue
to arrive on every subscription's `SystemUpdates()` channel alongside the typed
account/portfolio channels.

## Connection management

- One WebSocket connection per `Client`, multiplexed across subscriptions.
- A reader goroutine dispatches frames to per-subscription channels; a writer
  goroutine serializes outbound frames (avoids concurrent-write issues).
- Heartbeat: ping every 30s; pong deadline 10s.
- `Client.Close` cancels the connection's owned I/O context and force-closes the
  socket, so shutdown does not wait on a blocked read or close handshake.

## Reconnect

On unexpected disconnect:

1. Emit `ErrWSDisconnected` on every subscription's `Errors()` channel.
2. Backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s, with full jitter.
3. Re-establish the connection.
4. Re-send active subscriptions with new request ids.
5. Emit `ErrWSReconnected` on each `Errors()` channel so callers can react.

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
  receive → unsubscribe for market data, account (`acq`), and portfolio (`pos`).
- Reconnect test with a server that drops the connection once (market data and
  account streams).
- Buffer-overflow test asserts the drop policy and counter.
- `goleak` asserts no goroutines survive `Close()`.
