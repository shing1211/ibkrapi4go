# todos.md — Blueprint Hardening & Enhancement

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Mode:** BUILD
- **Base commit:** `96d7d75`

Statuses: `todo` · `doing` · `blocked` · `review` · `done`

## Phase A — Correctness & Security (P0)

| ID | Task | Role | Status | Depends On | Size | Acceptance |
|----|------|------|--------|-----------|------|-----------|
 | A1 | Fix broken build (`examples/mock` duplicate `main`/`defaultGatewayURL`) | backend | done | — | S | `go build ./...` exits 0 |
| A2 | Fix `check_money.py` regex + scan `client/`/`internal/` | backend | done | — | M | detects scalar/map/pointer float money |
| A3 | Fix money precision leaks (`rest_banking.go`, `rest.go`, `CashBalance`) | backend | done | A2 | M | money/qty never cast to float32 |
| A4 | Fix timeout body-cancellation (`internal/transport.go:138`) | backend | done | — | S | large body read succeeds |
| A5 | Bound success-response sizes | backend | done | — | M | oversized response → typed error |
| A6 | Harden redaction (bearer/`sess`/csrf/form secrets) | security | done | — | S | secrets masked on all paths |
| A7 | Sanitize `Error.Message` at construction | security | done | — | S | user-visible message redacted |

## Phase B — CI/CD & Release

| ID | Task | Role | Status | Depends On | Size | Acceptance |
|----|------|------|--------|-----------|------|-----------|
 | B1 | Run `golangci-lint` in CI + enable `gosec` | devops | done | A1 | S | CI fails on lint |
| B2 | Consolidate duplicate release workflows | devops | done | — | S | single `v*` release path |
| B3 | GoReleaser multi-arch + checksums + provenance; fix SLSA claim | devops | B2 | M | binaries + checksums published |
| B4 | Coverage threshold + `-coverpkg` | devops | done | A1 | S | CI fails below threshold |
| B5 | OS/Go-version test matrix | devops | done | A1 | S | matrix runs |
| B6 | DCO enforcement + dependency-review | devops | done | — | S | unsigned commit blocked |
| B7 | Real secret scanner + CycloneDX SBOM | devops | done | B6 | M | standard SBOM artifact |
| E9 | Enforce GoDoc in CI | devops | done | B1 | S | undocumented export fails CI |

## Phase C — Reliability & Observability

| ID | Task | Role | Status | Depends On | Size | Acceptance |
|----|------|------|--------|-----------|------|-----------|
 | C1 | `internal/ws.go` unit tests | tester | done | A1 | M | ws.go covered |
| C2 | Fix flaky `TestWS_SystemUpdates`; `sts` on subscribe | tester | done | C1 | S | stable ×20 |
| C3 | Resilience tests (storm/heartbeat/cancel-write/dup/out-of-order) | tester | done | C1 | M | pass under `-race` |
| C4 | Injectable clock/dialer | backend | done | C1 | M | fake clock in tests |
| C5 | Missing metrics (retry/pacing/heartbeat/dropped/queue/durations) | backend | done | — | M | metrics emitted + tested |
| C6 | Composite health/readiness probe | backend | done | C5 | M | structured `Health(ctx)` |
| C7 | Real OTel tracing bridge (`contrib/otel`) | backend | done | C5 | L | REST + WS spans |
| C8 | Fuzz in CI + WS/REST-error targets | tester | done | A1 | S/M | `make fuzz` + CI job |

## Phase D — Feature Gaps

| ID | Task | Role | Status | Depends On | Size | Acceptance |
|----|------|------|--------|-----------|------|-----------|
| D1 | Order state machine + duplicate-submission protection | architect | done | C1 | L | explicit transitions; dup blocked |
| D2 | Bracket/OCA/conditional/multi-leg + TIF validation | backend | done | D1 | L | new flows tested; non-breaking |
| D3 | Typed order/execution/portfolio WS events | backend | done | C1 | M/L | typed events + tests |
| D4 | WS sequence/gap detection | backend | done | D3 | M | gap signal emitted |
| D5 | Delayed-data flags + market-data permissions | backend | done | — | M | fields on `Snapshot`/`Update` |
| D6 | `ClientOrderID` round-trip | backend | done | — | S | field populated |
| D7 | Implement account/portfolio streaming (typed WS events) | backend | done | D3 | M/L | typed account/portfolio events |

## Phase E — Docs/DX

| ID | Task | Role | Status | Depends On | Size | Acceptance |
|----|------|------|--------|-----------|------|-----------|
| E1 | Fix version/spec drift (doc.go, README×6, SPEC, ROADMAP) | docs | done | — | S | current release / spec v2.40.0 |
| E2 | Fix wrong error docs | docs | done | — | S | ERRORS.md matches code |
| E3 | Fix `scripts/check_design/main.go` | backend | review | — | S/M | fails on real drift |
| E4 | Remove leaking `CreateSessionRaw` | backend | done | — | S | no generated type in public sig |
| E5 | Add `docs/GATEWAY-SETUP.md` + `docs/PERMISSIONS.md` | docs | done | — | S | linked from README |
| E6 | Fix misleading live examples | docs | done | — | S | examples accurate |
| E7 | Add cancellation + reconciliation examples | docs | done | — | S | entries compile |
| E8 | Document streaming (incl. account/portfolio from D7) | docs | done | D7 | S | STREAMING.md covers new capability |
| E10 | README ASCII architecture diagram | docs | done | — | S | diagram + translations |

## Close-out

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|-----------|-----------|
| Z1 | Docs sync pass | docs | done | all waves | docs reflect changes |
| Z2 | Release (commit + push both remotes; sync gitee) | release | done | Z1 | both remotes at same commit |
| Z3 | Run report + index + next-phase | orchestrator | done | Z2 | artifacts written |
