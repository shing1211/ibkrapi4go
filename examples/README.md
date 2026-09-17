# Examples

Runnable examples that target the in-repo mock gateway by default, so they need
no IBKR account and no network access to the real API.

## Run the mock and an example

In one terminal, start the mock:

```bash
go run ./cmd/ibkr-mock-gateway
```

The binary prints the base URLs it serves and the environment variables to use.
Every surface is served on one port (default `:5001`):

| Surface | Path | SDK configuration |
|---------|------|-------------------|
| CPAPI | `/v1/api/*` | `IBKR_GATEWAY_URL` / `ibkr.WithGatewayURL` |
| CPAPI WebSocket | `/v1/api/ws` | derived from `IBKR_GATEWAY_URL` |
| IB REST | `/gw/api/v1/*`, `/gw/api/v2/*` | `IBKR_REST_GATEWAY_URL` / `ibkr.WithRESTGateway` |
| OAuth2 token | `/oauth2/api/v1/token` | OAuth2 client options |

In another terminal, run an example:

```bash
go run ./examples/mock
```

## Index

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

## Pointing at a different gateway

Both examples read `IBKR_GATEWAY_URL`, so an already-running mock (or the real
local Client Portal Gateway) can be used:

```bash
IBKR_GATEWAY_URL=https://localhost:5000 go run ./examples/mock
```

When the gateway uses a self-signed certificate, also set
`IBKR_INSECURE_SKIP_VERIFY=true`.

## See also

- [`docs/MOCK-GATEWAY.md`](../docs/MOCK-GATEWAY.md) — mock gateway usage, flags,
  exposed URLs, and the Go test API.
