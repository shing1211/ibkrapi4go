# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-pre--alpha-red?style=flat-square" alt="Status">
</p>

> **⚠️ Unofficial & pre-alpha.** ibkrapi4go is a community Go SDK for the
> Interactive Brokers Web API. It is **not affiliated with Interactive Brokers**.
> It is under active development: the public packages are not yet implemented.
> See [DISCLAIMER.md](./DISCLAIMER.md) and [docs/ROADMAP.md](./docs/ROADMAP.md).

> **Go-native. Type-safe. OpenAPI-driven.** An idiomatic Go client for the
> Interactive Brokers Web API — account management, portfolio, trading, market
> data, and real-time WebSocket streaming.

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

## Table of Contents

- [Status](#status)
- [The Two APIs](#the-two-apis)
- [Install](#install)
- [Planned Usage](#planned-usage)
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
| OpenAPI codegen validation | ✅ Verified (see [docs/CODEGEN.md](./docs/CODEGEN.md)) |
| `client/` generated code | 🚧 Generated on demand (`make codegen`) |
| `pkg/ibkr` public API | 🚧 Not implemented |
| `internal/` implementation | 🚧 Not implemented |
| Tests / examples | 🚧 Not implemented |

This repository currently contains **documentation and codegen tooling**, plus a
verified codegen recipe. No public SDK code exists yet. See
[docs/ROADMAP.md](./docs/ROADMAP.md) for the build plan.

## The Two APIs

The IBKR OpenAPI spec (v2.39.0) actually describes **two API surfaces with two
different authentication schemes**. They are not interchangeable:

| Surface | Base path | Operations | Auth |
|---------|-----------|-----------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | 70 | `oauth2Bearer` |
| **Total** | | **185** | |

**v1 of this SDK targets CPAPI (`ssoBearer`) only.** The `oauth2Bearer` surface
is deferred to a later phase. See [ADR 0001](./docs/adr/0001-two-api-surfaces.md)
and [ADR 0005](./docs/adr/0005-v1-scope.md).

## Install

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

Requires **Go 1.26+** and a running [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/).

## Planned Usage

> The API below is the **target design** and is **not yet implemented**. It is
> shown to communicate the intended ergonomics. It will not compile today.

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
├── pkg/ibkr/        # Public SDK surface (planned)
├── internal/        # Private implementation (planned)
├── docs/            # Design, reference, ADRs
├── scripts/         # Codegen + validation
└── specs/           # Cached OpenAPI spec (gitignored)
```

## Repository Docs

| Doc | Contents |
|-----|----------|
| [docs/SPEC.md](./docs/SPEC.md) | Canonical endpoint index (surface + auth) |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Layering, composition, middleware |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased plan with exit criteria |
| [docs/AUTH.md](./docs/AUTH.md) | The two auth models |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | Spec fetch, patch, generate, verify |
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
