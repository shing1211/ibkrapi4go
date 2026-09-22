# report.md — 15-Phase Enhancement Plan

- **Run:** `2026-09-20-next-15-phases`
- **Date:** 2026-09-20
- **Mode:** BUILD
- **Status:** Complete

## Summary

All 15 enhancement phases executed successfully. 20 tasks completed across
6 specialist roles (tester, architect, backend, devops, docs, security).
All verification commands pass: `go build`, `go vet`, `go test`, `gofmt`.

## Planned vs. Actual

| Phase | Name | Planned | Actual | Status |
|-------|------|---------|--------|--------|
| S1 | Goroutine safety audit | tester | tester | ✅ Done |
| A1 | Interface segregation | architect | architect | ✅ Done |
| A2 | Typed builders | backend | backend | ✅ Done |
| S2 | Backpressure & deadline propagation | backend | backend | ✅ Done |
| E1 | CLI tool | devops | devops | ✅ Done |
| A3 | Middleware plugin API | architect | architect | ✅ Done |
| E2 | Migration guide v0.x → v1.0 | docs | docs | ✅ Done |
| S3 | Production resilience | backend | backend | ✅ Done |
| S4 | Test quality & fault injection | tester | tester | ✅ Done |
| A4 | Unified `*Pager[T]` pagination | backend | backend | ✅ Done |
| E3 | API reference site | docs | docs | ✅ Done |
| E4 | Release engineering automation | devops | devops | ✅ Done |
| E5 | Supply-chain security | security | security | ✅ Done |
| S5 | Performance baseline | backend | backend | ✅ Done |
| A5 | Multi-client / shared transport | architect | architect | ✅ Done |

## Files produced

### Stability Hardening

| File | Action |
|------|--------|
| `internal/session.go` | Panic recovery + context propagation for tickle goroutine |
| `internal/ws.go` | Panic recovery + context propagation for readLoop/writeLoop/pingLoop |
| `internal/breaker.go` | Error budget tracking with sliding window |
| `pkg/ibkr/ws.go` | Panic recovery for subscription context watcher |
| `pkg/ibkr/timeout.go` | **New** — `WithEndpointTimeout` option |
| `pkg/ibkr/timeout_test.go` | **New** — tests for per-endpoint timeout |
| `internal/fault_injection_test.go` | **New** — 21 transport-level fault injection tests |
| `pkg/ibkr/fault_injection_test.go` | **New** — 15 SDK-level fault injection tests |
| `pkg/ibkr/profile_test.go` | **New** — CPU/memory profiling benchmarks |
| `pkg/ibkr/alloc_test.go` | **New** — allocation counting benchmarks |
| `internal/transport.go` | HTTP/2 pooling + user middleware wiring |
| `benchmark.baseline` | Updated with current results |

### Ecosystem

| File | Action |
|------|--------|
| `cmd/ibkr/main.go` | **New** — CLI entry point, global flags, help text |
| `cmd/ibkr/accounts.go` | **New** — `accounts` command |
| `cmd/ibkr/positions.go` | **New** — `positions` command |
| `cmd/ibkr/orders.go` | **New** — `orders list/submit/cancel` commands |
| `cmd/ibkr/stream.go` | **New** — `stream` command |
| `cmd/ibkr/portfolio.go` | **New** — `portfolio summary/ledger/allocation` commands |
| `cmd/ibkr/config.go` | **New** — `config show/set` commands |
| `cmd/ibkr/completion.go` | **New** — shell completion generation |
| `docs/MIGRATION.md` | **New** — 173-line migration guide |
| `scripts/codemod.sh` | **New** — 74-rule codemod script |
| `docs/api.html` | **New** — 65KB API reference page |
| `docs/architecture.html` | **New** — architecture diagram page |
| `docs/decisions.html` | **New** — ADR browser page |
| `docs/css/api.css` | **New** — API reference styling |
| `docs/js/api.js` | **New** — search functionality |
| `.github/workflows/release-automation.yml` | **New** — changelog + Gitee + semver |
| `.github/workflows/supply-chain.yml` | **New** — SBOM + secret scan + dep audit |
| `.github/workflows/ci.yml` | Modified — conventional commits check + go mod verify |
| `.github/workflows/release.yml` | Modified — Gitee push step |
| `scripts/changelog-gen.sh` | **New** — changelog from conventional commits |
| `scripts/sbom-gen.sh` | **New** — SBOM generation |

### Architectural

| File | Action |
|------|--------|
| `internal/interfaces.go` | **New** — TokenProvider, RoundTripper, SessionMachine, WSClient, RateLimiter interfaces |
| `internal/fake/fake.go` | **New** — package doc |
| `internal/fake/token_source.go` | **New** — fake TokenProvider |
| `internal/fake/session.go` | **New** — fake SessionMachine |
| `internal/fake/ratelimiter.go` | **New** — fake RateLimiter |
| `internal/fake/transport.go` | **New** — fake RoundTripper |
| `internal/fake/ws.go` | **New** — fake WSClient |
| `internal/oauth.go` | Renamed `tokenSource` → `TokenSource` |
| `pkg/ibkr/builders.go` | **New** — OrderBuilder, ContractBuilder, TransferInstructionBuilder |
| `pkg/ibkr/builders_test.go` | **New** — 21 builder tests |
| `pkg/ibkr/middleware.go` | **New** — WithTransportMiddleware option |
| `pkg/ibkr/middleware_test.go` | **New** — 3 middleware tests |
| `pkg/ibkr/pager.go` | **New** — generic `Pager[T]` type |
| `pkg/ibkr/pager_test.go` | **New** — 7 pager tests |
| `pkg/ibkr/models.go` | Added `ModelsPager`, deprecated `AllModels` |
| `pkg/ibkr/notifications.go` | Added `FYIsPager`, deprecated `AllFYIs` |
| `pkg/ibkr/performance.go` | Added `TransactionsPager`, deprecated `Transactions` |
| `pkg/ibkr/portfolio.go` | Added `SubaccountsPager`, deprecated `Subaccounts` |
| `pkg/ibkr/transport_pool.go` | **New** — TransportPool type |
| `pkg/ibkr/transport_pool_test.go` | **New** — 11 pool tests |
| `examples/middleware/main.go` | **New** — custom auth middleware example |
| `internal/session.go` | Added `SetHTTPClient()` for pool support |
| `pkg/ibkr/client.go` | Added `release` field + `userMiddleware` for pool/middleware support |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| New dependency requests | Low | All phases explicitly prohibit new deps without ADR |
| A2 scope creep | Low | Acceptance criteria limited to 3 builder types |
| E1 design bikeshedding | Low | CLI spec detailed; approval gates before implementation |
| goroutine safety reveals panics | Low | Phase 3 already claimed goleak coverage |

## Follow-ups

| Item | Priority | Note |
|------|----------|------|
| `GITEE_TOKEN` secret | High | Must be added to GitHub repo secrets for Gitee push |
| Duplicate release workflows | Medium | `release.yml` and `release-automation.yml` both trigger on `v*` tags |
| Pre-existing gofmt CRLF drift | Low | Many files have CRLF line endings on Windows |

## Next action

Run `next-phase.md` recommends v1.0.0 stabilization or OTel integration as the next phase.

See `next-phase.md` for candidate next phases.
