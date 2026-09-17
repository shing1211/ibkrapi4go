# Todos: Phase 9 — Correctness & Conformance Closure

Run: `docs/runs/2026-09-17-correctness-closure/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `blocked` · `review` · `done`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T1 | D1: enumerate all nil-interface{} request builder panics | backend | done | — | 5 ops need guard (3 SDK-reachable: GetContractInfo/6fields, GetConidsByExchange/1field, GetAllFyis/3fields; 2 structural: GetTradingSchedule2, ModifyFyiEmails) |
| T2 | D1: apply nil guard fix | backend | done | T1 | 12 nil guards; `make check` pass |
| T3 | D2: fix 8 REST wrapper mismatches | backend | done | T2 | 8 ops decode correctly; checks pass |
| T4 | Docs sync | docs | done | T3 | No stale refs; `make docs-check` pass |
| T5 | Release: commit + push both remotes | release | done | T4 | Both remotes at `adb3c1f`; no force-push |
| T6 | Next-phase planning | planner | done | T5 | `next-phase.md` |
| T7 | Close-out report + index | orchestrator | doing | T6 | `report.md` + index line |
| T7 | Close-out report + index | orchestrator | todo | T6 | `report.md` + index line |

## Notes
- Base commit: `eb34ca1`
- T1 first: enumerate all affected ops BEFORE touching any code.
- If `make tools` unavailable, T1/T2 may need to patch `client/client.gen.go`
  directly (exceptional, flagged in release note).
- D1: root cause is spec-patch produces `interface{}` with `omitempty` for
  optional non-pointer params; codegen template doesn't guard nil.
- D2: 8 ops have wrapper/decode mismatches; fix wrappers (not fixtures).
