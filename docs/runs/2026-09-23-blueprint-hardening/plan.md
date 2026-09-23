# Plan — Blueprint Hardening & Enhancement

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Date:** 2026-09-23
- **Mode:** BUILD
- **Base commit:** `96d7d75`

## Goal

Harden and enhance the existing `ibkrapi4go` Go SDK against the production-grade
blueprint, using the blueprint as a **capability checklist**, not a rewrite
mandate. Fix P0 correctness/security defects, make CI gates real, raise test and
observability coverage, close feature gaps, and bring docs to truth.

## Locked Scope Decisions

1. **Money:** keep **ADR 0008** (`string` / `json.Number`). Do **not** add
   `github.com/shopspring/decimal` (would reverse a ratified ADR and break v1.x).
2. **Layout:** keep `pkg/ibkr` + `internal/` + `client/`. Do **not** restructure
   into `/pkg/domain`, `/pkg/services`, `/pkg/transport`, `/pkg/errors`.
3. **Phase D features:** all in scope.
4. **E8 → D7:** **implement** account/portfolio streaming (user decision,
   2026-09-23). E8 becomes the documentation of the shipped feature.

## Audit Findings (baseline at `96d7d75`)

| Area | Status | Key evidence |
|------|--------|--------------|
| Build | **BROKEN** | `examples/mock/main.go:30,32` + `error-handling.go:31,33` duplicate `main`/`defaultGatewayURL`; `go build ./...` exits 1 |
| Money precision | **BROKEN** | `scripts/check_money.py` regex misses scalar/pointer/map/array; `rest_banking.go` casts transfer Amount/Qty to `float32` on outgoing requests; `rest.go` tax vouchers `*float32` |
| Transport timeout | **DEFECT** | `internal/transport.go:138-152` `defer cancel()` fires before body read → `context.Canceled` on large success bodies |
| Response size | **GAP** | success bodies unbounded (`response.go:42`, 193 `io.ReadAll` in `client.gen.go`) |
| Redaction | **PARTIAL** | `internal/transport.go:179-181` header-regex only; misses bearer/`sess`/csrf/form secrets; `Error.Message` unredacted |
| Coverage | **LOW** | `pkg/ibkr` 30.8%, `internal` 60.1%, `internal/mockgateway` 58.2% (target 85%+) |
| CI gates | **PARTIAL** | `.golangci.yml` exists but never runs; `gosec` absent; no coverage threshold; Ubuntu-only |
| Release | **OVERLAP** | `release.yml` + `release-automation.yml` both fire on `v*`; no GoReleaser/cross-compile/checksums |
| Observability | **PARTIAL** | `slog` + metrics exist; **no distributed tracing**; missing retry/pacing/heartbeat/queue-depth/order-duration metrics; no health/readiness probe |
| Docs | **PARTIAL** | version/spec drift (`doc.go` v1.0.0 vs tag v1.0.1; spec v2.39 vs v2.40); wrong error docs; `check_design` is a no-op |
| Feature gaps | **MISSING** | order state machine, duplicate-submission protection, bracket/OCA/multi-leg, typed WS events, WS gap detection, delayed-data flags, ClientOrderID round-trip |

## Phase A — Correctness & Security (P0)

| ID | Objective | Role | Deps | Size | Acceptance | Verification |
|----|-----------|------|------|------|-----------|--------------|
| A1 | Fix broken build — separate `examples/mock/error-handling.go` so it does not redeclare `main`/`defaultGatewayURL` | backend | — | S | `go build ./...` exits 0 | `go build ./...` |
| A2 | Fix `check_money.py` regex; scan `client/` + `internal/`; cover scalar/pointer/map/array | backend | — | M | detects injected `Money float64` scalar | `python scripts/check_money.py` |
| A3 | Fix money precision leaks: `rest_banking.go` outgoing Amount/Qty/TransferPrice (hand-build decimal JSON like `trade.go`), `rest.go` tax vouchers, `CashBalance` rounding | backend | A2 | M | no `float32` cast on money/qty paths; round-trip exact | `make money-check` + new tests |
| A4 | Fix timeout middleware cancelling context before body read (`transport.go:138`) | backend | — | S | large slow body reads succeed | new `internal/transport` test |
| A5 | Bound success-response bodies (`http.MaxBytesReader`/`LimitReader`) | backend | — | M | oversized response returns typed error | new test |
| A6 | Harden redaction: bearer tokens, `sess=`, `x-csrf-token`, form `client_secret`/`refresh_token` | security | — | S | secrets masked in all log paths | new redaction tests |
| A7 | Sanitize `Error.Message` at construction (docs promise it) | security | — | S | `Error.Message` shown to user is redacted | new test |

## Phase B — CI/CD & Release (P1)

| ID | Objective | Role | Deps | Size | Acceptance |
|----|-----------|------|------|------|-----------|
| B1 | Run `golangci-lint` in CI; enable `gosec` in `.golangci.yml` | devops | A1 | S | CI job fails on lint error |
| B2 | Consolidate `release.yml` + `release-automation.yml` (one tag path) | devops | — | S | single release workflow on `v*` |
| B3 | GoReleaser multi-arch + checksums + provenance; correct false SLSA claim in CHANGELOG | devops | B2 | M | release publishes binaries + checksums |
| B4 | Coverage threshold + `-coverpkg` | devops | A1 | S | CI fails below threshold |
| B5 | Linux/macOS/Windows × Go-version matrix | devops | A1 | S | matrix runs green |
| B6 | DCO enforcement action + `dependency-review` | devops | — | S | unsigned commit/PR blocked |
| B7 | Real secret scanner + CycloneDX SBOM | devops | B6 | M | standard SBOM artifact on release |
| E9 | Enforce GoDoc in CI (`revive` exported → error) | devops | B1 | S | undocumented export fails CI |

## Phase C — Reliability & Observability (P1/P2)

| ID | Objective | Role | Deps | Size | Acceptance |
|----|-----------|------|------|------|-----------|
| C1 | `internal/ws.go` unit tests (dispatch, parseSystemFrame, backoff, framing) | tester | A1 | M | `internal/ws.go` > 0% and covered paths |
| C2 | Fix flaky `TestWS_SystemUpdates`; emit `sts` on subscribe | tester | C1 | S | test stable across 20 runs |
| C3 | Resilience tests: reconnect storm, heartbeat timeout, cancel-write, dup/out-of-order | tester | C1 | M | new tests pass under `-race` |
| C4 | Injectable clock/dialer (remove real sleeps in tests) | backend | C1 | M | timing tests use fake clock |
| C5 | Missing metrics: retry, 429/pacing, heartbeat, dropped, queue depth, order durations | backend | — | M | metrics emitted + tested |
| C6 | Composite health/readiness probe (transport/session/MD-auth/trading) | backend | C5 | M | `Health(ctx)` returns structured status |
| C7 | Real OTel **tracing** bridge in `contrib/otel` | backend | C5 | L | spans for REST + WS lifecycle |
| C8 | Run fuzz in CI + WS-frame/REST-error fuzz targets | tester | A1 | S/M | `make fuzz` + CI smoke job |

## Phase D — Feature Gaps (P2, large)

| ID | Objective | Role | Deps | Size | Acceptance |
|----|-----------|------|------|------|-----------|
| D1 | Order state machine + duplicate-submission protection (cOID registry) | architect | C1 | L | explicit `OrderState` transitions; dup submit blocked |
| D2 | Bracket/OCA/conditional/multi-leg orders + TIF validation (additive to `OrderRequest`) | backend | D1 | L | new order flows tested; non-breaking |
| D3 | Typed order/execution/portfolio WS events (replace raw `[]byte`) | backend | C1 | M/L | typed event types + tests |
| D4 | WS sequence/gap detection | backend | D3 | M | gap signal emitted on missing seq |
| D5 | Delayed-data flags + market-data permission surfacing | backend | — | M | fields exposed on `Snapshot`/`Update` |
| D6 | `ClientOrderID` round-trip (parse back onto `Order`/`OrderStatus`) | backend | — | S | field populated from responses |
| D7 | Implement account/portfolio streaming — typed WS events for account, order, execution, and portfolio updates | backend | D3 | M/L | stream delivers typed account/portfolio events |

## Phase E — Docs/DX

| ID | Objective | Role | Deps | Size | Acceptance |
|----|-----------|------|------|------|-----------|
| E1 | Fix version/spec drift (`doc.go`, README×6, `docs/SPEC.md`, `docs/ROADMAP.md`) | docs | — | S | docs say v1.0.1 / spec v2.40.0 |
| E2 | Fix wrong error docs (`ErrStreamDisconnected` removed but documented) | docs | — | S | ERRORS.md matches `errors.go` |
| E3 | Fix `scripts/check_design/main.go` (currently self-referential no-op) | backend | — | S/M | tool fails when design doc drifts from code |
| E4 | Remove leaking generated type `CreateSessionRaw` (`rest_sso.go:160`) | backend | — | S | no generated type in public signature |
| E5 | Add `docs/GATEWAY-SETUP.md` + `docs/PERMISSIONS.md` | docs | — | S | new docs linked from README |
| E6 | Fix misleading live examples (creds unused; `forceRefreshToken` no-op) | docs | — | S | examples accurate |
| E7 | Add cancellation + reconciliation examples | docs | — | S | `examples/` entries compile/run |
| E8 | Document streaming (incl. new account/portfolio streaming from D7) | docs | D7 | S | `docs/STREAMING.md` covers new capability |
| E10 | README ASCII architecture diagram | docs | — | S | diagram in README + translations |

## Order of Work (waves)

```
Wave 0 (critical path): A1 → A2 → A3, plus A4–A7
Wave 1: B1–B7, E9
Wave 2: C1–C8, E1–E4
Wave 3: E5–E8, E10
Wave 4: D1 → D2; D3 → D4 → D7; D5, D6
```

## Verification (every wave)

`go build ./...` · `go vet ./...` · `go test -race ./...` · `make check` ·
`make codegen-verify` · `make docs-check` · `python scripts/check_links.py` ·
`python scripts/check_i18n.py`.

Phase A additionally: money-check must fail on a seeded `Money float64` before A2
and pass after A3.

## Risks

1. **A3 changes wire payloads** for transfers — correctness fix, but verify with
   round-trip tests; behavior change for current callers.
2. **D1/D2 touch order flow** — keep additive/non-breaking for v1.x; any breaking
   change needs a v2 or explicit approval.
3. **D3/D4 both modify `internal/ws.go`** — serialize.
4. **38 tasks is multi-session** — checkpoint after each wave; keep `todos.md`
   current.
5. **`gitee/main` is 2 commits behind** (`223d333` vs `96d7d75`) — sync before
   release steps.

## Open Questions

1. **D2 scope:** are bracket/OCA fields additive-only, or may we break the
   `OrderRequest` shape (would require a v2 bump)?
2. **Run folder location:** `docs/runs/` was archived; this run recreates it —
   confirm new runs belong under `docs/runs/` going forward (not `docs/archive/runs/`).

Resolved 2026-09-23: account/portfolio streaming is **implemented** (D7), not
deferred.
