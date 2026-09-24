# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.40-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-stable-brightgreen?style=flat-square" alt="Status">
  <a href="https://codecov.io/gh/shing1211/ibkrapi4go"><img src="https://codecov.io/gh/shing1211/ibkrapi4go/branch/main/graph/badge.svg" alt="Coverage"></a>
  <a href="https://shing1211.github.io/ibkrapi4go/"><img src="https://img.shields.io/badge/Docs-GitHub%20Pages-97CAFF?style=flat-square&logo=github" alt="Docs"></a>
</p>

> **⚠️ Unofficial.** ibkrapi4go is a community Go SDK for the
> Interactive Brokers Web API. It is **not affiliated with Interactive Brokers**.
> All 185 API operations are implemented (115 CPAPI + 70 IB REST).
> See [DISCLAIMER.md](./DISCLAIMER.md) and [docs/ROADMAP.md](./docs/ROADMAP.md).

> **Go-native. Type-safe. OpenAPI-driven.** An idiomatic Go client for the
> Interactive Brokers Web API — account management, portfolio, trading, market
> data, and real-time WebSocket streaming.

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

## Table of Contents

- [Status](#status)
- [The Two APIs](#the-two-apis)
- [Install](#install)
- [Usage](#usage)
- [Authentication](#authentication)
- [Package Layout](#package-layout)
- [Repository Docs](#repository-docs)
- [Build & Test](#build--test)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

---

## Status

| Item | State |
|------|-------|
| Planning & documentation | ✅ Complete |
| OpenAPI codegen validation | ✅ Verified ([docs/CODEGEN.md](./docs/CODEGEN.md)) |
| `client/` generated code | ✅ Committed (generated) |
| `pkg/ibkr` public API | ✅ Implemented (185/185 operations) |
| `internal/` implementation | ✅ Implemented |
| Tests / examples | ✅ Implemented |
| Mock gateway (185/185 ops) | ✅ Shipped ([docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md)) |
| Benchmarks + fuzz tests | ✅ Shipped ([docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)) |
| Metrics + logging | ✅ Shipped ([docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)) |
| CI quality gates (coverage, pre-commit) | ✅ Shipped |
| Docs website + Discussions | ✅ Shipped ([docs/ROADMAP.md](./docs/ROADMAP.md)) |
| Release | ✅ v1.0.6 (GitHub + Gitee) |

This repository contains the **generated OpenAPI client**, documentation, and
codegen tooling. All 185 API operations are implemented across both CPAPI and IB REST
surfaces. See [docs/ROADMAP.md](./docs/ROADMAP.md) for the build plan.

## The Two APIs

The IBKR OpenAPI spec (v2.40.0) actually describes **two API surfaces with two
different authentication schemes**. They are not interchangeable:

| Surface | Base path | Operations | Auth |
|---------|-----------|-----------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | 70 | `oauth2Bearer` |
| **Total** | | **185** | |

v1 originally scoped CPAPI (`ssoBearer`) only; the `oauth2Bearer` surface shipped
in Phases 5-6. See [ADR 0001](./docs/adr/0001-two-api-surfaces.md) and
[ADR 0011](./docs/adr/0011-oauth2-surface.md).

## Install

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

Requires **Go 1.26+** and a running [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/).

## Usage

A minimal Client Portal API (CPAPI) example:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	ctx := context.Background()

	// The Client Portal Gateway is authenticated interactively in a browser.
	// The SDK talks to an already-authenticated local gateway.
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // local gateway uses a self-signed cert
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatal(err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, acc := range accounts {
		fmt.Fprintf(os.Stdout, "account: %s\n", acc.AccountID)
	}
}
```

## Authentication

The Client Portal Gateway is authenticated **interactively** (browser login +
2FA). The SDK does not accept a username and password, and the IBKR spec defines
no password grant. The SDK talks to an already-authenticated gateway and manages
the resulting session token.

See [docs/AUTH.md](./docs/AUTH.md) and [docs/SESSIONS.md](./docs/SESSIONS.md).

## Package Layout

```
ibkrapi4go/
├── client/          # Generated OpenAPI types + HTTP client (DO NOT EDIT)
├── pkg/ibkr/        # Public SDK surface
├── internal/        # Private implementation
├── cmd/             # Standalone binaries (ibkr-mock-gateway)
├── examples/        # Runnable examples (mock)
├── docs/            # Design, reference, ADRs
├── scripts/         # Codegen + validation
└── specs/           # Cached OpenAPI spec (gitignored)
```

## How it fits together

```text
NewClient(options...)
   │
   ▼
SessionManager.Initialize(ctx)   → tickle goroutine starts
   │
   ▼
Manager calls (Account, Portfolio, Trade, MarketData)
   │
   ▼
Client.Close()                   → tickle stops, logout, ws closed
```

Every request flows through a transport chain before it reaches the gateway:

```text
Request
  → request ID + User-Agent
  → auth header injection (bearer)
  → logging + telemetry hooks
  → circuit breaker (optional)
  → retry (safe methods only; honors Retry-After)
  → per-endpoint rate limiter
  → global rate limiter
  → per-request timeout (when the caller sets none)
  → HTTP call
  → error parsing (IBKR envelope → *ibkr.Error)
Response
```

See [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) for the full layering.

## Repository Docs

| Doc | Contents |
|-----|----------|
| [docs/SPEC.md](./docs/SPEC.md) | Canonical endpoint index (surface + auth) |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Layering, composition, middleware |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased plan with exit criteria |
| [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) | In-repo mock gateway for tests, examples, and a standalone binary |
| [docs/GATEWAY-SETUP.md](./docs/GATEWAY-SETUP.md) | Running and authenticating the Client Portal Gateway |
| [docs/PERMISSIONS.md](./docs/PERMISSIONS.md) | Trading permissions, market-data entitlements, delayed data |
| [docs/AUTH.md](./docs/AUTH.md) | The two auth models |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | Spec fetch, patch, generate, verify |
| [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) | Metrics, benchmarks, fuzz tests, logging |
| [docs/TESTING.md](./docs/TESTING.md) | Testing strategy and tiers |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | Terminology |
| [docs/adr/](./docs/adr/) | Architecture Decision Records |
| [docs/design/](./docs/design/) | Per-module design contracts |

## Build & Test

```bash
# Show available targets
make help

# Format, vet, test
make check

# Regenerate client/ from the OpenAPI spec
make codegen

# Verify codegen is reproducible (drift check)
make codegen-verify
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). All commits must be DCO-signed
(`git commit -s`). By contributing you agree to the
[Code of Conduct](./CODE_OF_CONDUCT.md). Translations follow
[TRANSLATING.md](./TRANSLATING.md).

## Security

See [SECURITY.md](./SECURITY.md) and [docs/design/08-concurrency.md](./docs/design/08-concurrency.md)
for the concurrency and secrets model.

## License

[Apache License 2.0](./LICENSE). See [NOTICE](./NOTICE) and
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).
