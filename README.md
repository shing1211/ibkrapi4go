# ibkr-sdk

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
</p>

> **⚠️ Under Active Development**
> This SDK is under active development against the [IBKR Web API v2.39](https://api.ibkr.com/gw/api/v3/api-docs).
> APIs and types are subject to change. Audit generated types against the official spec
> before relying on any field for production use.

> **Go-native. Type-safe. OpenAPI-driven.** The most complete and ergonomic Go SDK for
> [Interactive Brokers Web API](https://www.interactivebrokers.com/api/) — account management,
> portfolio, trading, market data, and real-time WebSocket streaming.

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Key Features](#key-features)
- [Authentication](#authentication)
- [Package Map](#package-map)
- [API Coverage](#api-coverage)
- [Examples](#examples)
- [Build \& Test](#build--test)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

---

## Install

```bash
go get github.com/shing1211/ibkrapi4go@latest
```

Requires **Go 1.26+** and a running [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/).

## Quick Start

### 1. Start Client Portal Gateway

Download and run the [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/).
By default it listens on `https://localhost:5000`.

### 2. Connect and authenticate

```go
package main

import (
    "context"
    "fmt"
    "log"

    ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
    // Create client connected to local Client Portal Gateway
    cli, err := ibkr.NewClient(
        ibkr.WithGatewayURL("https://localhost:5000"),
        ibkr.WithAPISecret("your-ibkr-api-secret"), // from IBKR settings
    )
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    ctx := context.Background()

    // Authenticate (SSO or token-based)
    if err := cli.Auth().SSO(ctx, "username", "password"); err != nil {
        log.Fatal(err)
    }

    // List all brokerage accounts
    accounts, err := cli.Account().List(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, acc := range accounts {
        fmt.Printf("Account: %s | %s\n", acc.ID, acc.Type)
    }

    // Get portfolio positions
    positions, err := cli.Portfolio().Positions(ctx, accounts[0].ID, nil)
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range positions {
        fmt.Printf("  %s %d @ %.2f\n", p.Contract.Symbol, p.Position, p.CostBasis)
    }
}
```

## Key Features

- **185 endpoints** across Account, Portfolio, Trading, Market Data, FA, and more
- **443 generated types** from official OpenAPI 3.0 spec
- **Session management** — automatic tickle heartbeat (60s interval), token refresh
- **WebSocket streaming** — channel-based market data, order updates, notifications (via `coder/websocket`)
- **Fluent API** — `cli.Account().List()`, `cli.Portfolio().Positions()`, `cli.Trade().SubmitOrder()`
- **FutuAPI-compatible design** — follows the same package layout as [futuapi4go](https://github.com/shing1211/futuapi4go)
- **Rate limiting** — enforces 10 req/sec per-endpoint with exponential backoff on 429
- **Auto-reconnect** — connection state machine with graceful shutdown
- **Paper trading support** — same API, paper account credentials

## Authentication

### SSO (Interactive Login)

```go
cli.Auth().SSO(ctx, "username", "password")
```

### API Key / Token

```go
cli, _ := ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithToken("your-access-token"),
)
```

### OAuth 2.0

```go
cli.Auth().OAuth2(ctx, clientID, clientSecret)
```

## Package Map

```
ibkr-sdk/
├── client/          # Generated OpenAPI types + HTTP client
├── pkg/
│   └── ibkr/        # Public SDK surface
│       ├── auth.go       # Authentication & session
│       ├── account.go    # Account management
│       ├── portfolio.go  # Portfolio & positions
│       ├── trade.go      # Order management
│       ├── marketdata.go # Market data & streaming
│       └── ws.go         # WebSocket client
├── internal/        # Private implementation
├── docs/            # Architecture & design docs
├── scripts/         # Build, codegen, test scripts
└── test/            # Integration tests
```

## API Coverage

| Category | Endpoints | Status |
|----------|-----------|--------|
| Account Management | 83 | ✅ |
| Trading | 55 | ✅ |
| Portfolio | 21 | ✅ |
| Market Data | 4 | ✅ |
| WebSocket | 1 | ✅ |
| **Total** | **185** | **✅** |

Full endpoint list: [docs/SPEC.md](./docs/SPEC.md)

## Examples

See the [examples](./examples/) directory:

- `account_list.go` — List all accounts
- `portfolio_positions.go` — Fetch positions
- `place_order.go` — Submit an order
- `market_data_snapshot.go` — Get real-time quote
- `ws_stream.go` — Subscribe to WebSocket market data

## Build & Test

```bash
# Install dependencies
go mod tidy

# Run unit tests
go test ./...

# Run integration tests (requires Client Portal Gateway)
IBKR_GATEWAY=https://localhost:5000 \
IBKR_USERNAME=user \
IBKR_PASSWORD=pass \
go test ./test/... -tags=integration

# Generate types from OpenAPI spec
./scripts/codegen.sh
```

## Architecture

See [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) for detailed design decisions.

## Troubleshooting

**`401 Unauthorized`**
→ Session expired. Call `cli.Auth().SSO()` again or check your token.

**`429 Too Many Requests`**
→ Rate limit hit. ibkr-sdk retries automatically with backoff.

**`Connection refused` on `localhost:5000`**
→ Client Portal Gateway is not running. Download from ibkr.com/api.

**Spec drift**
→ Run `./scripts/codegen.sh` to regenerate types from latest spec.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). All contributions welcome.

## License

Licensed under the Apache License 2.0. See [LICENSE](./LICENSE).
