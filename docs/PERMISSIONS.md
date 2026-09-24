# Permissions & Entitlements

Access to IBKR data and trading is governed by the **permissions** attached to
your account, not by the SDK. The SDK surfaces what the gateway returns; it does
not grant, cache, or bypass entitlements. This page explains the common cases so
you can interpret the symptoms correctly.

See also: [GATEWAY-SETUP.md](./GATEWAY-SETUP.md) · [AUTH.md](./AUTH.md) ·
[STREAMING.md](./STREAMING.md) · [ERRORS.md](./ERRORS.md).

## Trading permissions

Placing orders requires the relevant **trading permission** for the product and
market (stocks, options, futures, specific exchanges, short selling, etc.).
Without it, order endpoints fail or the order is rejected by IBKR.

- The permission lives on the account at IBKR; grant it in Account Management.
- Read-only or paper accounts can browse data and simulate orders but not trade.
- The SDK does not pre-validate permissions; it returns the gateway's error
  (see [ERRORS.md](./ERRORS.md)). Treat order rejections as authoritative.

## Market-data entitlements

Real-time market data requires a **market-data subscription** per exchange/data
vendor. Entitlements are per account and per data source (e.g. US equities,
options, futures).

- Without a subscription, quotes may be **delayed** or unavailable.
- Subscriptions are billed by IBKR; the SDK never purchases them.
- Sharing/consolidation flags come back in the data (see *Delayed data* below).

## Delayed data

When you are not entitled to real-time data, the gateway may return **delayed**
values rather than an error. The SDK exposes this explicitly on market-data
types via field `6509` (`MarketDataStatus`):

| Availability | Meaning | `IsDelayed` | `IsFrozen` | `IsNotSubscribed` |
|--------------|---------|:-----------:|:----------:|:-----------------:|
| `R` | Real-time | no | no | no |
| `D` | Delayed | yes | no | no |
| `Z` | Frozen | no | yes | no |
| `Y` | Frozen-delayed | yes | yes | no |
| `N` | Not subscribed | no | no | yes |
| `i` | Incomplete | no | no | no |
| `v` | VDR-exempt | no | no | no |

- `Snapshot.Status` carries this for REST snapshots.
- `Update.Status` carries it for streamed field `6509`.

Check `status.IsDelayed` / `status.IsNotSubscribed` before treating a value as
live. Example:

```go
snaps, err := cli.MarketData().Snapshot(ctx, []ibkr.ConID{265598}, fields)
if err != nil { return err }
for _, s := range snaps {
    if s.Status != nil && s.Status.IsDelayed {
        log.Printf("conid %d: delayed data (%s)", s.ConID, s.Status.Availability)
    }
}
```

Never rely on quotes for execution decisions without confirming they are
real-time.

## Read-only accounts

Read-only access (for example an advisor or a view-only login) can read
accounts, positions, and market data but cannot submit, modify, or cancel
orders. The SDK surfaces the gateway's rejection; it does not attempt to work
around it. See [ERRORS.md](./ERRORS.md).

## Subscription limits

Independent of entitlements, IBKR limits conids/fields per request and total
subscriptions per session. The SDK enforces configurable ceilings and returns an
error wrapping `ErrStreamingLimit` when exceeded — see the limits table in
[STREAMING.md](./STREAMING.md). These limits are account/entitlement dependent;
raise the SDK's `WithStreamingLimits` only up to what your session allows.

## Troubleshooting

| Symptom | Likely cause | Where to look |
|---------|--------------|---------------|
| Quotes are stale but present | Delayed entitlement | `MarketDataStatus.IsDelayed` |
| `N` availability / no quotes | Not subscribed to the venue | IBKR market-data subscriptions |
| Order rejected immediately | Missing trading permission or read-only account | IBKR Account Management |
| `ErrStreamingLimit` | Too many conids/fields/subscriptions | [STREAMING.md](./STREAMING.md) |
| Intermittent data gaps | Entitlement or data-vendor issue | IBKR client services |

Permissions are managed at IBKR; the SDK only reports what the gateway returns.
