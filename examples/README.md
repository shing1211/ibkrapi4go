# Examples

Runnable examples demonstrating the SDK. There are two groups: **mock** examples that
need no credentials, and **live** examples that require paper trading credentials.

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

### Mock gateway configuration

The mock is served on `:5001` by default. Override with `IBKR_GATEWAY_URL`:

```bash
IBKR_GATEWAY_URL=https://localhost:5000 go run ./examples/mock
```

## Live examples (paper trading)

Live examples require a running IBKR Client Portal Gateway and paper trading
credentials. Set these environment variables before running any live example:

| Variable | Description |
|----------|-------------|
| `IBKR_GATEWAY` | Gateway base URL (e.g. `https://localhost:5000`) |
| `IBKR_USERNAME` | Paper trading username |
| `IBKR_PASSWORD` | Paper trading password |

```bash
IBKR_GATEWAY=https://localhost:5000 \
IBKR_USERNAME=yourpaperusername \
IBKR_PASSWORD=yourpaperpassword \
  go run ./examples/live-portfolio
```

All live examples are **read-only** — no orders are submitted, no positions are
modified, and no transfers are initiated.

### Index (live — paper trading required)

- [`live-portfolio/main.go`](./live-portfolio/main.go) — Session.Initialize,
  Account.List, Portfolio.Positions/Ledger/Summary.
- [`options-chain/main.go`](./options-chain/main.go) — search for a stock by
  symbol, then fetch call and put strikes for a given expiry month.
- [`screener/main.go`](./screener/main.go) — fetch available scanner parameters
  (instruments, locations, scan types), then run a market scanner.
- [`multi-account/main.go`](./multi-account/main.go) — aggregate accounts and
  positions across multiple clients using `MultiClient`.
- [`live/oauth2-flow.go`](./live/oauth2-flow.go) — OAuth2 token acquisition and
  refresh for IB REST API.

## See also

- [`docs/MOCK-GATEWAY.md`](../docs/MOCK-GATEWAY.md) — mock gateway usage, flags,
  exposed URLs, and the Go test API.
- [`test/integration_test.go`](../test/integration_test.go) — integration tests
  using the same paper-trading env vars.
