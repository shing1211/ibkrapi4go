# todos.md — Reconciliation

- **Run:** `2026-09-21-ibkr-reconciliation`
- **Mode:** BUILD

| ID | Task | Size | Status | Depends | Notes |
|----|------|------|--------|---------|-------|
| R1 | Fix WS dispatch — add non-market-data frame routing | L | todo | — | architect role |
| R2 | Patch scripts/patch_spec.py — ConID float32 → int64 | M | todo | — | backend role |
| R3 | Add float64 → string wrapper for AccountSummary | M | todo | — | backend role |
| R4 | Make decodeJSON consistent in rest_accounts.go | S | todo | — | backend role |
| R5 | Add wrappers for unwrapped CPAPI endpoints | M | todo | — | backend role |
| R6 | Update SPEC.md endpoint counts | S | todo | R1–R5 | docs role |
| R7 | Commit + push | S | todo | R1–R6 | release role |
