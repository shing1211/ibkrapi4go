# 15-Phase Enhancement Plan: Stability, Ecosystem, Architectural

- **Run:** `2026-09-20-next-15-phases`
- **Date:** 2026-09-20
- **Mode:** BUILD (plan approved 2026-09-20)
- **Status:** Planned

## Context

Phases 0–8 are complete (185/185 operations implemented, v0.2.0 shipped). The
project is at v0.2.0 and entering P1 — v1.0.0 readiness. This plan defines the
next 15 enhancement phases across three non-functional dimensions: **Stability
Hardening**, **Ecosystem**, and **Architectural**. All phases are
pre-v1.0; breaking changes are acceptable before stabilization.

## Guiding constraints

- **No new dependencies** without an ADR (AGENTS.md rule 4).
- **No auto-retry order mutations** (ADR 0009).
- **Money/quantities as `string`** (ADR 0008).
- Generated code (`client/*.gen.go`) is never hand-edited.
- `make codegen-verify` must pass after any spec-driven change.

---

## Phase inventory

### Stability Hardening (S1–S5)

| # | Phase | Name | What | Effort | Depends |
|---|-------|------|------|--------|---------|
| S1 | Goroutine safety audit | Panic boundaries + goleak on all async paths | Verify `goleak` coverage on tickle goroutine, token refresh, WS reconnect, session init. Only `ws_test.go` has goleak today. | S | — |
| S2 | Backpressure & deadline propagation | Honor caller cancellation everywhere | Propagate `context.Deadline` through WS subscribe, tickle, token refresh. Add `WithRequestTimeout` per-endpoint override. | M | S1 |
| S3 | Production resilience | Bulkheads + graceful degradation | Per-endpoint timeouts, error budgets, circuit breaker tuning, panic recovery in goroutines, graceful degradation under gateway failures. | M | S1 |
| S4 | Test quality & fault injection | Reliability under failure | Mutation testing (`mutagen`), fault-injection in integration tests, coverage gap analysis, property-based testing for JSON round-trips. | L | — |
| S5 | Performance baseline | Profile and optimize hot paths | HTTP/2 connection pooling, request pipeline batching, zero-allocation JSON paths, memory profile baseline. Lower urgency — correctness above speed. | M | S2 |

### Ecosystem (E1–E5)

| # | Phase | Name | What | Effort | Depends |
|---|-------|------|------|--------|---------|
| E1 | CLI tool | `ibkr` binary | `ibkr accounts`, `ibkr positions`, `ibkr orders submit`, `ibkr stream` — composable commands, shell completions, config file support. | M | S1 |
| E2 | Migration guide v0.x → v1.0 | Breaking change catalog | Upgrade path for every symbol rename, codemod suggestions, compatibility shim strategy. Blocking v1.0 release. | S | S1 |
| E3 | API reference site | Branded docs site | mkdocs-gen from godoc, architecture diagrams, searchable decision log. Adoption driver — godoc alone insufficient for 180-symbol API. | M | E1 |
| E4 | Release engineering automation | Sustainable maintenance | Changelog from conventional commits, semantic-version enforcement in CI, auto-release note draft, Gitee mirror automation. | M | — |
| E5 | Supply-chain security | Trust & provenance | SBOM generation, dependency pinning audit, provenance attestations, secret scanning in CI, `go.mod` lock-file strategy. | M | — |

### Architectural (A1–A5)

| # | Phase | Name | What | Effort | Depends |
|---|-------|------|------|--------|---------|
| A1 | Interface segregation | Mockable `internal/` boundaries | `TokenSource`, `RoundTripper`, `SessionStateMachine` interfaces in `internal/` — unit testable without mock gateway or full transport stack. | M | — |
| A2 | Typed builders | `OrderBuilder`, `ContractBuilder`, `TransferInstructionBuilder` | Chainable, validated at construction. Complex objects require callers to set 10+ fields struct-by-struct today. | M | — |
| A3 | Middleware plugin API | User-injectable `http.RoundTripper` chain | Custom auth schemes, tracing, request/response logging as middleware — extension point between transport and user code. | L | A1 |
| A4 | Unified `*Pager[T]` pagination | Standard iterator for all list ops | Currently only `PositionIterator` exists. Extend to `AllModels`, `AllFYIs`, `Transactions`, `Subaccounts`, etc. | L | — |
| A5 | Multi-client / shared transport | Shared `*http.Client` + session across `Client` instances | Multi-account setups share connections and rate limiters; reduces gateway load. ARCHITECTURE.md "future" item. | M | A1 |

---

## Recommended sequencing

Phases are ordered by: correctness first, adoption impact, then architectural foundation.

| Order | Phase | Topic | Rationale |
|-------|-------|-------|-----------|
| 1 | S1 | Stability | Correctness hard requirement |
| 2 | A1 | Architectural | Testability foundation — unlocks unit testing without mock gateway |
| 3 | A2 | Architectural | DX + correctness for complex objects |
| 4 | S2 | Stability | Resource safety — honor caller cancellation |
| 5 | E1 | Ecosystem | Biggest DX win — single command to explore the API |
| 6 | A3 | Architectural | Extensibility — production users need middleware |
| 7 | E2 | Ecosystem | Blocking v1.0 — users need upgrade path |
| 8 | S3 | Stability | Bulkheads + graceful degradation |
| 9 | S4 | Stability | Reliability under failure |
| 10 | A4 | Architectural | API consistency —统一的 pagination story |
| 11 | E3 | Ecosystem | Adoption driver — searchable API reference |
| 12 | E4 | Ecosystem | Sustainable release process |
| 13 | E5 | Ecosystem | Trust — SBOM and provenance |
| 14 | S5 | Stability | Performance — lower urgency than correctness |
| 15 | A5 | Architectural | Multi-account convenience — lowest priority |

### Interdependencies (full graph)

```
S1
 │
 ├─► S2 ─► S3 ─► S5
 │            (S3 depends on S1 goroutine safety)
 │
A1 ─► A3
 │    │
 │    └─► A5 (A5 depends on A1)
 │
 └─► A2 ─► (no dependents)
 │
E1 ─► E3
 │
E2    (no deps — can run in parallel with S1)
```

---

## Exit criteria per phase

### S1 — Goroutine safety audit

- [ ] `goleak` asserted in tests for: tickle goroutine, token refresh, WS reconnect, session init.
- [ ] No `go.uber.org/goleak` errors in `go test ./...` for all packages.
- [ ] Panic recovery added to any goroutine lacking it.

### S2 — Backpressure & deadline propagation

- [ ] All `select` statements on `ctx.Done()` in WS subscribe path honor deadline.
- [ ] `WithRequestTimeout` option added per-endpoint override.
- [ ] Tickling stops immediately when context is cancelled.
- [ ] Token refresh goroutine exits on context cancellation.

### S3 — Production resilience

- [ ] Per-endpoint timeout configurable via `WithEndpointTimeout`.
- [ ] Circuit breaker trips cleanly under sustained 5xx load.
- [ ] Panic recovery wraps all goroutine entry points in `internal/`.
- [ ] Error budget tracking for sustained degraded mode.

### S4 — Test quality & fault injection

- [ ] `mutagen` mutation coverage ≥ 80% on `internal/` packages.
- [ ] Integration tests simulate: gateway timeout (504), rate limit (429), auth failure (401), server error (500).
- [ ] Property-based tests for all public JSON decode paths.

### S5 — Performance baseline

- [ ] Benchmark baseline stored in `benchmark.baseline`.
- [ ] HTTP/2 multiplexing enabled and validated.
- [ ] ≥1 zero-allocation JSON path identified and applied.
- [ ] CI regression gate fails at >10% ns/op regression.

### A1 — Interface segregation

- [ ] `TokenSource` interface: `FetchToken(ctx) (string, error)`.
- [ ] `RoundTripper` interface wraps `http.RoundTripper`.
- [ ] `SessionStateMachine` interface: `State()`, `Initialize(ctx)`, `Close(ctx)`.
- [ ] All `internal/` components depend on interfaces, not concrete types.
- [ ] `internal/` packages unit-testable with `fakeClient` implementations.

### A2 — Typed builders

- [ ] `OrderBuilder` with chainable methods: `Side`, `Type`, `Quantity`, `LimitPrice`, `StopPrice`, `TimeInForce`, `Account`, `Build()`.
- [ ] `ContractBuilder` for combo positions with leg composition.
- [ ] `TransferInstructionBuilder` with field validation at `Build()`.
- [ ] Builders return `*Order` (or error) fully populated with defaults.
- [ ] Builder methods validate at call time, not at `Build()`.

### A3 — Middleware plugin API

- [ ] `WithTransportMiddleware(f func(http.RoundTripper) http.RoundTripper)` option on `NewClient`.
- [ ] Middleware chain visible in transport stack trace.
- [ ] Example: `otelmiddleware`, `customAuth` demonstrated in `examples/`.

### A4 — Unified `*Pager[T]` pagination

- [ ] `Pager[T]` type in `pkg/ibkr/pagination.go` alongside `PositionIterator`.
- [ ] `AllModels`, `AllFYIs`, `Transactions`, `Subaccounts` return `*Pager[T]`.
- [ ] `Next`, `Value`, `Err` methods consistent with `PositionIterator`.
- [ ] Backward compatible — existing slice-returning methods deprecated, not removed.

### A5 — Multi-client / shared transport

- [ ] `NewSharedTransport()` returning a `*TransportPool` with shared `*http.Client`, session, rate limiter.
- [ ] Multiple `Client` instances share one transport pool.
- [ ] `TransportPool.Close()` gracefully tears down all shared resources.
- [ ] Rate limiter shared across all clients in pool.

### E1 — CLI tool

- [ ] `ibkr` binary supports: `accounts`, `positions`, `orders list`, `orders submit`, `stream`, `portfolio`.
- [ ] Shell completions (bash, zsh, fish) generated.
- [ ] `ibkr help` self-documents.
- [ ] `ibkr config` manages gateway URL, credentials, defaults.
- [ ] Builds cleanly on Linux, macOS, Windows.

### E2 — Migration guide v0.x → v1.0

- [ ] Document at `docs/MIGRATION.md` covering every symbol renamed since v0.1.0.
- [ ] Codemod script (`scripts/codemod.sh`) handles ≥80% of mechanical renames.
- [ ] Compatibility shims for: `Get*` prefix removal, `SessionValidation`/`SessionToken` renames.
- [ ] `make docs-check` passes with migration doc in tree.

### E3 — API reference site

- [ ] Generated from godoc via `mkdocs-gen` or equivalent (no new deps).
- [ ] Architecture diagram (layered `pkg/ibkr → internal → client`).
- [ ] Decision log browser over `docs/adr/`.
- [ ] Hosted on GitHub Pages from `docs/` folder.
- [ ] Searchable across all manager methods and types.

### E4 — Release engineering automation

- [ ] Changelog generated from conventional commits (`git-changelog` or equivalent).
- [ ] Semantic version enforced in CI: PR titles must follow conventional commits.
- [ ] Release note draft auto-generated on tag.
- [ ] Gitee mirror push automated in CI on tag.

### E5 — Supply-chain security

- [ ] SBOM generated on every release (`syft` or equivalent, no new deps).
- [ ] `go.mod` pinned to specific patch versions (not `latest`).
- [ ] Secret scanning in CI (`trufflehog` or `git-secrets`).
- [ ] Provenance attestation via GitHub Actions (`attest`).
- [ ] `go vet ./...` and `gosec` pass in CI.

---

## Open questions

| # | Question | Options | Recommendation |
|---|----------|---------|----------------|
| Q1 | Should S5 (performance) use a real paper trading account or stay mock-only? | Real account / Mock only | Mock only (no deps, no credentials) |
| Q2 | CLI — interactive mode or pure command-line? | Interactive shell / Pure flags | Pure flags + config file (scriptable) |
| Q3 | A2 builders — validate eagerly (at each method) or lazily (at Build)? | Eager / Lazy | Eager — fail fast on first invalid field |
| Q4 | A5 transport pool — thread-safe by default or explicit `Sync()` call? | Thread-safe / Explicit sync | Thread-safe (`sync.Mutex` internally) |
| Q5 | E4 changelog — generate from git history or require PR authors to write changelog entries? | Git history / PR entries / Both | Both (git history as draft, PR author refines) |

---

## References

- [ROADMAP.md](../../../ROADMAP.md) — phases 0–8 complete
- [STABILITY.md](../../../STABILITY.md) — public API definition and stability levels
- [RELEASING.md](../../../RELEASING.md) — versioning and release process
- [ARCHITECTURE.md](../../../ARCHITECTURE.md) — current layering and "future" items
- [ADR 0015](../../../adr/0015-stability.md) — stability contract
- [ADR 0008](../../../adr/0008-numeric-precision.md) — money/quantities as string
- [ADR 0009](../../../adr/0009-no-auto-retry-orders.md) — no auto-retry orders
