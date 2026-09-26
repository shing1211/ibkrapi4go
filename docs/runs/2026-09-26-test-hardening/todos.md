# Test Hardening - Todos

| # | Item | Status | Note |
|---|------|--------|------|
| 0 | Re-measure coverage with CI flags | done | 43.3%; 6275 stmts; 62.8 per point |
| 1 | Annotate the stale `-race` claim in the ws-shutdown report | done | Environment property, not a project limitation |
| 1 | Remove the false "replacement-blind" item (3 locations) | done | `check_design` catches replacements; real gap is doc breadth |
| 1 | Correct the stale 37.5% and goleak-framing claims | done | 43.3% / package-exit detection already existed |
| 1 | Restate P2 honestly | done | 2 of 9 design docs verified, not strictness |
| 1 | Mark the ratchet question answered | done | Decision: ratchet, in slices |
| 1 | Confirm one canonical floor value | done | `ci.yml:76` only; run records are history, not competing counts |
| 2 | Reproduce the `TestWS_Resilience` flake | pending | Read the leaked goroutine before choosing a fix |
| 2 | Fix the cause in `Close`/loop shutdown | pending | Bounded settle only if no real leak exists |
| 3 | Per-test goleak: `internal/ws_test.go` | pending | Attribution |
| 3 | Per-test goleak: `internal/session_test.go` | pending | 0 cleanup sites, highest risk |
| 3 | Per-test goleak: `pkg/ibkr/ws_test.go` | pending | Attribution |
| 3 | Per-test goleak: `pkg/ibkr/managers_e2e_test.go` | pending | Attribution |
| 3 | Fix any leaks the new assertions surface | pending | Fix, do not `IgnoreTopFunction` |
| 4 | Slice 1: `restrictions.go` | pending | 234 stmts |
| 4 | Slice 1: `rest_utilities.go` | pending | 119 stmts |
| 4 | Slice 1: `notifications.go` | pending | 112 stmts |
| 4 | Slice 1: `trading_accounts.go` | pending | 83 stmts |
| 4 | Slice 1: raise floor to measured value | pending | Never above measured |
| 4 | `cmd/ibkr` meaningful parts | pending | Order validation, flag handling; `main()` stays uncovered |

## Slice 1 Budget

548 statements across four managers is about 8.7 points if fully covered. Partial
coverage is expected, so the floor will be set to whatever is actually measured,
not to 8.7 points of optimism.
