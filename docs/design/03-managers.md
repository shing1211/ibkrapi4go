# Design 03 — Managers

Managers are the public, domain-facing API. Each wraps a slice of the generated
client and returns stable public types.

## Managers (v1, CPAPI)

| Manager | Scope | Representative calls |
|---------|-------|----------------------|
| `AccountManager` | accounts & summaries | `List`, `Summary`, `BalanceSummary`, `MarginSummary`, `PnL` |
| `PortfolioManager` | positions & ledger | `Accounts`, `Positions`, `Position`, `Ledger`, `Allocation`, `Invalidate` |
| `TradeManager` | orders & contracts | `Submit`, `Modify`, `Cancel`, `OpenOrders`, `OrderStatus`, `Trades`, `SearchContracts`, `ContractInfo` |
| `MarketDataManager` | quotes & history | `Snapshot`, `History`, `Unsubscribe`, `UnsubscribeAll`, `Subscribe` |

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

Each manager has table-driven tests against `httptest` fixtures, asserting the
outbound request (path, query, body) and the mapped response, including error
paths and pagination.
