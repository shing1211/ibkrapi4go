# Report: In-Repo Mock Gateway (185-op)

- **Run:** `docs/runs/2026-09-17-mock-gateway/`
- **Mode:** BUILD
- **Base commit:** `177e5be`
- **Release commit:** `f9cf1fc` — `feat(mockgateway): add in-repo 185-op mock gateway, CLI, examples, docs`
- **Remotes:** `origin/main` + `gitee/main` both at `f9cf1fc`
- **Status:** complete

## Shipped

| Task | Deliverable | Status |
|------|-------------|--------|
| T1 | `internal/mockgateway` core: server, options, fixtures, recorder, scenario, session/auth; migrated `pkg/ibkr/endtoend_test.go` | done |
| T2 | All 115 CPAPI routes + fixtures; SPEC-derived CPAPI coverage guard | done |
| T3 | All 70 IB REST routes (`/gw/api/v1\|v2`) + `/oauth2/api/v1/token` (client_credentials, refresh_token, private_key_jwt); guard extended to 185/185 | done |
| T4 | WebSocket streaming hub at `/v1/api/ws`; migrated `pkg/ibkr/ws_test.go` | done |
| T5 | `cmd/ibkr-mock-gateway` binary (single port, all surfaces) + `examples/mock` + `make mock-gateway` | done |
| T6 | `docs/adr/0014-mock-gateway.md` (Accepted) + `docs/MOCK-GATEWAY.md`; ROADMAP/TESTING/index links | done |
| T7 | Full docs sync (62 files reviewed; 17 updated) incl. README + 5 translations in lockstep | done |
| T7b | Corrected stale `internal/mockgateway` comments (latency semantics, recorder params, OAuth claims, streaming limits) | done |
| T7c | Fixed `MaxSubscriptions` to count distinct conids (F1) | done |
| T8 | Release: single conventional commit, pushed to both remotes | done |
| T9 | `next-phase.md` (6 candidates + recommended Phase 9) | done |

### Key outcomes
- **185/185 route coverage** with a guard (`internal/mockgateway/coverage_test.go`) that parses `docs/SPEC.md` and fails on any missing operation.
- **Zero new dependencies** — reuses `github.com/coder/websocket`; stdlib for HTTP.
- **One port** serves CPAPI, IB REST, OAuth2 token, and WebSocket.
- Deterministic fixtures (money/quantities as strings per ADR 0008), scriptable faults, request recorder, and race/goleak-clean streaming.
- `client/client.gen.go` untouched.

## Deferred / discovered

| ID | Item | Disposition |
|----|------|-------------|
| D1 | Generated-client panic on nil optional `interface{}` params (`GetConidsByExchange`, `GetContractInfo`, `GetAllFYIs`); root cause in spec patch/codegen; blocked by missing `oapi-codegen` (`make tools`) | → next-phase (Phase 9, C1) |
| D2 | 8 REST wrapper/model decode mismatches (`TaxDocuments.Generate` double-read; `Utilities.*`; `TaxVouchers.*`) | → next-phase (Phase 9, C2) |
| F1 | Mock `MaxSubscriptions` counted additions not distinct conids | **fixed** (T7c) |
| — | `docs/runs/2026-09-17-full-api-coverage/` never closed out | noted in `next-phase.md` (Phase 9, T9.4) |

## Risks / notes
- Mock fixtures are **synthetic** and shape-level; the guard asserts route presence, not payload conformance.
- Mock auth is shape/flow-level, not cryptographic.
- Latency faults are **max**, not additive; the recorder captures requests before routing (no `Params`).
- `make codegen-verify` cannot run locally without `make tools` (pre-existing environment gap).

## Verification (final)
`make check` · `make test-race` · `make docs-check` · `make license-check` — all pass.
Coverage guard: CPAPI 115/115, IB REST 70/70, total 185/185.

## Follow-ups
See `next-phase.md` — recommended **Phase 9: Correctness & conformance closure**
(D1 codegen fix, D2 wrapper fixes, fixture shape guard, run close-out + index,
integration scaffold).

## Actuals vs plan
All planned tasks T1–T10 completed. Two tasks were added mid-run: **T7b**
(comment corrections) and **T7c** (streaming-limit fix), both discovered by
sub-agents during verification of T6/T7. No planned scope was dropped.
