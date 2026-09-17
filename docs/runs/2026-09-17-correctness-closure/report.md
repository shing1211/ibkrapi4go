# Report: Phase 9 — Correctness & Conformance Closure

- **Run:** `docs/runs/2026-09-17-correctness-closure/`
- **Mode:** BUILD
- **Base commit:** `eb34ca1`
- **Release commit:** `adb3c1f` — `fix: correct nil-interface{} panics and 8 REST wrapper/decode mismatches`
- **Remotes:** `origin/main` + `gitee/main` both at `adb3c1f`
- **Status:** complete

## Shipped

| Task | Deliverable | Status |
|------|-------------|--------|
| T1 | Enumerated all nil-interface{} builder panics: 5 ops needing guard (3 SDK-reachable: GetContractInfo/6fields, GetConidsByExchange/1field, GetAllFyis/3fields; 2 structural: GetTradingSchedule2, ModifyFyiEmails) | done |
| T2 | Applied 12 nil guard blocks in `client/client.gen.go` (direct patch; toolchain gap) | done |
| T3 | Fixed 8 REST wrapper/decode mismatches (`rest.go` ×5, `rest_utilities.go` ×3) | done |
| T4 | Docs sync: CHANGELOG.md, AGENTS.md (toolchain-gap note) | done |
| T5 | Release: committed and pushed to both remotes | done |
| T6 | `next-phase.md` created | done |

### Key outcomes
- **D1 fixed:** 12 nil guards across 5 operations — `GetContractInfo` (6 fields), `GetConidsByExchange` (1), `GetAllFyis` (3), `GetTradingSchedule2` (1), `ModifyFyiEmails` (1).
- **D2 fixed:** 8 REST wrapper decode shapes corrected: `TaxDocuments.Generate`, `TaxVouchers.{CreateRequests,ActiveCountries,AvailableYears,Dividends}`, `Utilities.{Enumerations,ComplexAssetTransferBrokers,RequiredForms}`.
- **Toolchain note:** `patch_spec.py` cannot fix D1 (operates on spec JSON, not codegen templates); a proper root-fix would require modifying the `oapi-codegen` request-builder template. Direct gen.go patching was the pragmatic path; flagged in release note and AGENTS.md.

## Verification (final)
`make check` · `make test-race` · `make docs-check` · `make license-check` — all pass.
Coverage guard: still 185/185.

## Actuals vs plan
All planned tasks (T1–T6) completed. No planned scope dropped.
