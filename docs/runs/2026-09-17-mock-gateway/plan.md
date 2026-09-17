# Plan: In-Repo Mock Gateway (185-op)

- **Run:** `2026-09-17-mock-gateway`
- **Mode:** BUILD (plan approved 2026-09-17)
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `177e5be` (dependency-free metrics layer)

## Goal
Build a dependency-free, in-repo mock IBKR gateway that serves both API
surfaces (CPAPI `/v1/api/*` and IB REST `/gw/api/v1|v2` + `/oauth2/api/v1/token`)
with full 185-operation route coverage, scriptable fault injection, request
recording, and a WebSocket streaming mock — usable from tests, examples, and a
standalone `cmd/ibkr-mock-gateway` binary.

## Scope
- New package `internal/mockgateway` (no `testing` import; usable by `cmd`).
- New binary `cmd/ibkr-mock-gateway`.
- Migrate `pkg/ibkr/endtoend_test.go` inline gateway and `pkg/ibkr/ws_test.go`
  WS server to consume the package.
- Coverage guard parsing `docs/SPEC.md` (canonical 185 ops).
- ADR `docs/adr/0014-mock-gateway.md` and `docs/MOCK-GATEWAY.md`.

### Out of scope
- Cryptographic conformance (mock validates assertion shape only).
- Real IBKR spec redistribution; fixtures are synthetic.
- Public (non-`internal`) mock API surface.

## Assumptions
- `docs/SPEC.md` remains the canonical op list (AGENTS.md rule 6).
- Existing deps suffice: `net/http`, `net/http/httptest`,
  `github.com/coder/websocket`, `golang.org/x/time` (no new deps, ADR 0004).
- Auth fidelity is shape/flow level: session `Authorization` header + cookie
  jar for CPAPI; bearer + refresh for IB REST.

## Approach (alternatives)
1. **Public `mockgateway/` package.** Pros: reusable by SDK consumers. Cons:
   expands public API surface; maintenance burden; conflicts with minimalism.
2. **Test-only helper in `pkg/ibkr/*_test.go`.** Pros: smallest change. Cons:
   not reusable by `cmd`/examples; cannot build a standalone binary.
3. **`internal/mockgateway` + `cmd/ibkr-mock-gateway`** (chosen). Pros:
   reusable across tests/examples, no public surface expansion, supports a
   standalone binary, stdlib-only. Cons: none material.

## Task breakdown
| ID | Objective | Role | Inputs | Outputs | Depends | Acceptance | Verify | Size |
|----|-----------|------|--------|---------|---------|------------|--------|------|
| T1 | Core server: options, fixtures, recorder, scenario, session/auth; migrate endtoend test | backend | plan, client wiring, `endtoend_test.go` | `internal/mockgateway/{server,options,fixtures,recorder,scenario,session}.go`; migrated `endtoend_test.go` | — | `endtoend_test.go` green via package; no behavior loss; `make check`/`test-race` pass | `make check`, `make test-race` | L |
| T2 | Full CPAPI route coverage (115) + CPAPI coverage guard | backend | T1, `docs/SPEC.md`, generated client paths | `routes_cpapi.go`; `coverage_test.go` (CPAPI assertions) | T1 | Guard finds 0 missing CPAPI ops; fixtures valid JSON; tests pass | `make check`, `make test-race` | L |
| T3 | IB REST routes (`/gw/api/v1|v2`) + OAuth2 token/JWT | backend | T2, SPEC IB REST rows, `internal/oauth.go` | `routes_rest.go`, `oauth.go`; guard extended to 185 | T2 | Guard finds 0 missing ops total; `private_key_jwt`/refresh shape validated; tests pass | `make check`, `make test-race` | L |
| T4 | WebSocket streaming mock; migrate `ws_test.go` | backend | T3, `internal/ws.go`, `pkg/ibkr/ws_test.go` | `stream.go`; migrated `ws_test.go` | T3 | Streaming tests green; subscribe/unsubscribe + scripted ticks work | `make check`, `make test-race` | M |
| T5 | `cmd/ibkr-mock-gateway` binary + examples wiring | devops | T4 | `cmd/ibkr-mock-gateway/main.go`; examples using it | T4 | Binary serves both surfaces; `go run` works; flags documented | `go build ./...`, `make check` | M |
| T6 | ADR 0014 + `docs/MOCK-GATEWAY.md` + ROADMAP/TESTING links | architect | T5 | `docs/adr/0014-mock-gateway.md`, `docs/MOCK-GATEWAY.md`, index updates | T5 | ADR indexed; links resolve | `make docs-check`, `make license-check` | S |
| T7 | Full docs sync (all Markdown) | docs | T6 | updated docs set | T6 | Summary table of files reviewed; no stale refs; `make docs-check` | `make docs-check`, `make license-check` | M |
| T8 | Release: stage/commit/push both remotes | release | T7 | commits on `origin/main` + `gitee/main` | T7 | Both remotes show new commit; no force-push | `git log`, `git push` | S |
| T9 | Next-phase planning | planner | plan, report, docs | `next-phase.md` | T8 | 3–7 candidates + recommended phase | — | S |
| T10 | Close-out report + runs index | orchestrator | all | `report.md`, `docs/runs/index.md` | T9 | Artifacts complete | — | S |

## Order of work
T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10.
T2–T4 are file-disjoint but share `server.go` registration; run sequentially to
avoid working-tree conflicts between sub-agents.

## Risks
- **Fixture breadth (185 ops):** minimal-but-valid bodies; guard ensures route
  presence even when a body is trivial.
- **Path normalization drift:** normalize dynamic segments to `{}` in both the
  guard and the mux.
- **WS upgrade under `httptest`:** use `InsecureSkipVerify` (as current
  `ws_test.go` does).
- **Sub-agent scope creep:** briefs must pin exact files; verification gates
  each task.

## Verification (global)
`make check` · `make test-race` · `make docs-check` · `make license-check`,
plus the 185-op coverage guard. SPDX header on every new file;
`client/*.gen.go` untouched; no new dependencies.
