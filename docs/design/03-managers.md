# Design 03 — Managers

Managers are the public, domain-facing API. Each wraps a slice of the generated
client and returns stable public types.

## Managers (CPAPI)

| Manager | Scope | Methods (implemented) |
|---------|-------|-----------------------|
| `AccountManager` | accounts & summaries | 4 ops |
| `AlertManager` | alerts | 7 ops |
| `AllocationManager` | FA allocation | 9 ops |
| `FYIManager` | FYIs / notifications | 12 ops |
| `ForecastManager` | event contracts | 5 ops |
| `MarketDataManager` | quotes, history & streaming | 5 ops |
| `ModelManager` | model portfolios | 18 ops |
| `OAuthManager` | OAuth1 | 3 ops |
| `PerformanceManager` | PortfolioAnalyst | 5 ops |
| `PortfolioManager` | positions & ledger | 16 ops |
| `ScannerManager` | market scanner | 2 ops |
| `SessionManager` | session lifecycle & health | 7 ops |
| `TradeManager` | orders & contracts | 27 ops |
| `TradingAccountManager` | trading-account ops | 9 ops |
| `WatchlistManager` | watchlists | 4 ops |

One manager per row, and one count per row, on purpose. An earlier revision of this
table grouped managers into shared cells — `` | `AlertManager`, `ForecastManager`,
`ScannerManager` | … | 7, 5, and 2 ops respectively | `` — which reads better and
cannot be checked. The verifier matches a single manager name against a single `N
ops` cell, so every grouped row silently went unverified, and the drift it was
supposed to catch went unnoticed for nine runs. The counts below are enforced by
`make design-check`; a mismatch fails the build.

> **Method counts are verified against source by `make design-check`, not by hand.**
> The authoritative per-method list is the
> [godoc reference](https://pkg.go.dev/github.com/shing1211/ibkrapi4go/pkg/ibkr);
> this table carries counts only, so there is one number per manager to keep honest.

`ForecastManager` documents five operations: `ForecastCategories`,
`ForecastContract`, `ForecastMarkets`, `ForecastRules` and `ForecastSchedule`. The
`Get` prefix the OpenAPI operation ids carry (`getForecastCategories` and so on) is
dropped from the public method and kept in the `op` string, which is the same split
every other manager uses — `AlertManager.AllAlerts` runs under `Alert.GetAllAlerts`.

`ForecastCategories` returns `json.RawMessage` where the other four return typed
models. The category tree is an object keyed by category id, and the generated
response type flattens that map into a single struct, so a typed model built from it
would describe a shape IBKR does not send. `ScannerParameters` is passthrough for
the same reason.

The IB REST (`oauth2Bearer`) surface is exposed separately via `Client.REST()`
(`RESTSurface` and its `REST*` sub-managers); see
[08-concurrency.md](./08-concurrency.md) and
[../adr/0011](../adr/0011-oauth2-surface.md).

## Method conventions

- First parameter is always `context.Context`.
- Errors are `*ibkr.Error` (see [../ERRORS.md](../ERRORS.md)).
- Money/quantities are `string`/`json.Number` ([../adr/0008](../adr/0008-numeric-precision.md)).
- No method performs automatic writes on behalf of the caller except explicit
  order methods.

## Pagination

Positions and orders are paginated by IBKR. Managers expose an iterator rather
than a bare slice:

```go
it := cli.Portfolio().PositionsPaginated(ctx, accountID)
for it.Next(ctx) {
    p := it.Value()
    // ...
}
if err := it.Err(); err != nil { ... }
```

`Positions(ctx, accountID, nil)` fetches the first page for convenience.

## Request/response mapping

Managers never return generated types. They map generated responses to public
structs (see [04-generated-wrapping.md](./04-generated-wrapping.md)).

## Ids and types

- Account ids: `type AccountID string`.
- Contract ids: `type ConID int`.
- Field codes: `type Field string` with constants.

## Example

```go
accounts, err := cli.Account().List(ctx)
sum, err := cli.Account().Summary(ctx, accountID)
positions := cli.Portfolio().PositionsPaginated(ctx, accountID)
order, err := cli.Trade().Submit(ctx, accountID, req)
quote, err := cli.MarketData().Snapshot(ctx, conid, fields)
```

## Testing

Each manager has table-driven tests against the in-repo mock gateway
(`internal/mockgateway`) served via `httptest`, asserting the outbound request
(path, query, body) and the mapped response, including error paths and
pagination.
