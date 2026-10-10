# Examples

Runnable examples demonstrating the SDK. There are two groups: **mock** examples
that need no gateway, and **live** examples that require a running, authenticated
IBKR Client Portal Gateway.

## Mock examples

Run the mock gateway in one terminal:

```bash
go run ./cmd/ibkr-mock-gateway
```

Then run an example in another terminal:

```bash
go run ./examples/mock
```

### Index (mock — no credentials required)

- [`mock/main.go`](./mock/main.go) — connect to the mock, initialize a session,
  and list accounts.
- [`portfolio/main.go`](./portfolio/main.go) — query accounts, positions, ledger,
  and portfolio summary.
- [`marketdata-streaming/main.go`](./marketdata-streaming/main.go) — subscribe to
  real-time market-data fields and print updates.
- [`orders/main.go`](./orders/main.go) — WhatIf dry-run, submit an order, confirm
  it, and list open orders.
- [`models/main.go`](./models/main.go) — list model portfolios and their
  positions.
- [`middleware/main.go`](./middleware/main.go) — inject custom HTTP middleware via
  `WithTransportMiddleware`.
- [`mock-error-handling/main.go`](./mock-error-handling/main.go) — retry, error types, and
  graceful degradation patterns.
- [`mock/cancel-order/main.go`](./mock/cancel-order/main.go) — submit a resting
  order, then cancel it.
- [`mock/reconcile-open-orders/main.go`](./mock/reconcile-open-orders/main.go) —
  reconcile the open-order list against per-order status and flag drift.

### Mock gateway configuration

The mock is served on `:5001` by default. Override with `IBKR_GATEWAY_URL`:

```bash
IBKR_GATEWAY_URL=https://localhost:5000 go run ./examples/mock
```

## Live examples

Live examples require a running IBKR Client Portal Gateway that you have
authenticated in a browser. There is no username/password login — the SDK never
drives the login flow (see [`docs/GATEWAY-SETUP.md`](../docs/GATEWAY-SETUP.md)
and [`docs/AUTH.md`](../docs/AUTH.md)). The gateway URL is optional and defaults
to `https://localhost:5000`:

| Variable | Description |
|----------|-------------|
| `IBKR_GATEWAY` | Gateway base URL (default `https://localhost:5000`) |

```bash
IBKR_GATEWAY=https://localhost:5000 go run ./examples/live-portfolio
```

`live/oauth2-flow.go` is the exception: it targets the hosted IB REST surface and
needs OAuth2 credentials (`IBKR_CLIENT_ID`, `IBKR_CLIENT_SECRET`, and optionally
`IBKR_CLIENT_REFRESH_TOKEN`).

All live examples are **read-only** — no orders are submitted, no positions are
modified, and no transfers are initiated.

### Index (live — running gateway required)

- [`live-portfolio/main.go`](./live-portfolio/main.go) — Session.Initialize,
  Account.List, Portfolio.Positions/Ledger/Summary.
- [`options-chain/main.go`](./options-chain/main.go) — search for a stock by
  symbol, then fetch call and put strikes for a given expiry month.
- [`screener/main.go`](./screener/main.go) — fetch available scanner parameters
  (instruments, locations, scan types), then run a market scanner.
- [`multi-account/main.go`](./multi-account/main.go) — aggregate accounts and
  positions across multiple clients using `MultiClient`.
- [`live/oauth2-flow.go`](./live/oauth2-flow.go) — OAuth2 acquisition,
  automatic/explicit refresh, invalidation, and error handling for the IB REST
  API; token values are never printed.

## See also

- [`docs/MOCK-GATEWAY.md`](../docs/MOCK-GATEWAY.md) — mock gateway usage, flags,
  exposed URLs, and the Go test API.
- [`test/integration_test.go`](../test/integration_test.go) — integration tests
  using the same paper-trading env vars.
