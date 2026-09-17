# Todos: In-Repo Mock Gateway (185-op)

Run: `docs/runs/2026-09-17-mock-gateway/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `blocked` · `review` · `done`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T1 | Core server + session/auth + recorder + scenario + fixtures; migrate endtoend test | backend | done | — | endtoend green via package; `make check` + `test-race` pass |
| T2 | Full CPAPI 115 routes + CPAPI coverage guard | backend | done | T1 | 115/115 CPAPI routed; tests pass |
| T3 | IB REST `/gw/api/v1\|v2` + OAuth2 token/JWT | backend | done | T2 | 185/185 routed; OAuth grants validated; tests pass |
| T4 | WebSocket streaming mock; migrate `ws_test.go` | backend | done | T3 | Streaming tests green |
| T5 | `cmd/ibkr-mock-gateway` binary + examples wiring | devops | done | T4 | `go run` serves both surfaces |
| T6 | ADR 0014 + `docs/MOCK-GATEWAY.md` + ROADMAP/TESTING links | architect | done | T5 | ADR indexed; `make docs-check` pass |
| T7 | Full docs sync (all Markdown) | docs | done | T6 | Files-reviewed table; no stale refs |
| T7b | Fix stale mockgateway code comments (latency semantics) | reviewer | done | T7 | Comments match behavior; checks pass |
| T7c | Fix streaming `MaxSubscriptions` to count distinct conids (F1) | backend | done | T7b | Distinct-accounting matches client; tests pass |
| T8 | Release: commit + push GitHub & Gitee main | release | doing | T7c | Both remotes at new commit; no force-push |
| T9 | Next-phase planning | planner | todo | T8 | `next-phase.md` with 3–7 candidates |
| T10 | Close-out report + runs index | orchestrator | todo | T9 | `report.md` + index line |

## Discovered issues (deferred)
- **D1 — generated-client nil-`interface{}` panic.** Wrappers `TradeManager.GetConidsByExchange`, `TradeManager.GetContractInfo`, and `FYIManager.GetAllFYIs` panic before any HTTP call because the generated request builders pass a nil `interface{}` param (e.g. `params.AssetClass`) to `runtime.StyleParamWithOptions` without a nil guard (`client/client.gen.go:44409`). Root cause is spec-patch/codegen; fix belongs in `scripts/patch_spec.py` + regenerate (AGENTS.md rule 1). Blocked locally by missing `oapi-codegen` (`make tools`). → next-phase candidate.
- **D2 — REST wrapper/model mismatches.** Driving every REST wrapper against the mock surfaced 8 pre-existing wrapper decode issues (analogous to D1, not mock bugs): `TaxDocuments.Generate` double-reads a closed body; `Utilities.Enumerations`, `Utilities.ComplexAssetTransferBrokers`, `Utilities.RequiredForms`, `TaxVouchers.CreateRequests`, `TaxVouchers.ActiveCountries`, `TaxVouchers.AvailableYears`, `TaxVouchers.Dividends`. Fixtures use the spec/generated-model shape. → next-phase candidate (wrapper fixes, not codegen).

## Notes
- Base commit: `177e5be`
- T2–T4 are sequential to avoid concurrent edits to `server.go`.
- Per-task verification: `make check` + `make test-race`; docs tasks also
  `make docs-check` + `make license-check`.
- Never edit `client/*.gen.go`; no new dependencies (ADR 0004).
