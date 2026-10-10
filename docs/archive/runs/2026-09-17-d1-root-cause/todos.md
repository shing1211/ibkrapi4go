# Todos: D1 Root-Cause Fix — Retype Null Query Params

Run: `docs/runs/2026-09-17-d1-root-cause/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `done` · `cancelled`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| D1 | Enumerate all `type: null` query params; confirm typeless→`interface{}` | codegen | done | — | 22 params across 7 operations; reproduced panic path |
| D2 | Add `patch_spec.py` defect 4; retire `patch_gen.py` to no-op | codegen | done | D1 | `null_type_patched=22`; codegen scripts skip `patch_gen.py` |
| D3 | Regenerate `client/client.gen.go`; adapt `contract.go` + `notifications.go` | backend | done | D2 | build clean; 0 nil-guard comments; enum types emitted |
| D4 | Verify build/vet/tests/codegen-verify/docs/license | qa | done | D3 | all checks pass |
| D5 | Docs sync + release (commit `b94e677`, push both remotes) | docs + release | done | D4 | both remotes at `b94e677` |
| D6 | Run docs + index + next-phase | orchestrator | done | D5 | `report.md`, index line, `next-phase.md` |

## Notes

- Base commit: `5857f03`
- The 22 params are all plain string values; `type: string` is semantically correct,
  not a workaround.
- `right` (`GetContractInfo`) and `assetClass` (`GetTradingSchedule`) additionally
  carry `enum` constraints, so codegen now emits typed string enums with `Valid()`.
- No public SDK signature changes; only two internal literal constructions changed.
- `make tools` (oapi-codegen v2.8.0) was required to regenerate; it is pinned in
  the `Makefile`.
