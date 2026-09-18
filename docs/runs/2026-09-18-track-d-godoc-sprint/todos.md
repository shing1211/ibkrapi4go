# Track D — Godoc Sprint

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T1 | `rest_banking.go` — 26 symbols | docs | done | — | `go vet ./...` passes |
| T2 | `rest_utilities.go` — 10 symbols | docs | done | — | `go vet ./...` passes |
| T3 | `rest_balances.go` — 4 symbols | docs | done | — | `go vet ./...` passes |
| T4 | `restrictions.go` — 4 symbols | docs | done | — | `go vet ./...` passes |
| T5 | `rest_requests.go` — 4 symbols | docs | done | — | `go vet ./...` passes |
| T6 | `rest.go` (TaxVouchers) — 9 symbols | docs | done | — | `go vet ./...` passes |
| T7 | `rest_accounts.go` (types) — 8 symbols | docs | done | — | `go vet ./...` passes |
| T8 | Verify + commit + push | release | doing | T1–T7 | `make check` green, both remotes |
