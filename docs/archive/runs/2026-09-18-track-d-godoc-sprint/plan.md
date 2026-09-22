# Track D — Godoc Sprint

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** In Progress

## Goal

Add missing godoc comments to all exported symbols in `pkg/ibkr/` that currently lack them. Multi-line doc blocks with usage examples for complex types. Skip 4 wrapper types (`LoginMessagesWrapper`, `AccountStatusBulkWrapper`, `Au10TixWrapper`, `RegistrationTasksWrapper`) — add later if Track E doesn't unexport them.

## Scope — 65 symbols across 7 files

| File | Types | Methods | Total |
|------|:-----:|:-------:|:-----:|
| `rest_banking.go` | 13 | 13 | 26 |
| `rest_utilities.go` | 3 | 7 | 10 |
| `rest_balances.go` | 2 | 2 | 4 |
| `restrictions.go` | 1 | 3 | 4 |
| `rest_requests.go` | 2 | 2 | 4 |
| `rest.go` (TaxVouchers) | 3 | 6 | 9 |
| `rest_accounts.go` (types) | 8 | 0 | 8 |
| **TOTAL** | **32** | **33** | **65** |

## Task Breakdown

| ID | Task | Role | Size | Acceptance |
|----|------|------|------|------------|
| T1 | `rest_banking.go` — 26 symbols | docs | M | `go vet ./...` passes |
| T2 | `rest_utilities.go` — 10 symbols | docs | S | `go vet ./...` passes |
| T3 | `rest_balances.go` — 4 symbols | docs | S | `go vet ./...` passes |
| T4 | `restrictions.go` — 4 symbols | docs | S | `go vet ./...` passes |
| T5 | `rest_requests.go` — 4 symbols | docs | S | `go vet ./...` passes |
| T6 | `rest.go` (TaxVouchers section) — 9 symbols | docs | S | `go vet ./...` passes |
| T7 | `rest_accounts.go` (types only, skip wrappers) — 8 symbols | docs | S | `go vet ./...` passes |
| T8 | Verify + commit + push | release | S | `make check` green, both remotes updated |

## Verification

```bash
go vet ./...
go build ./...
make check
go test -race -count=1 ./pkg/ibkr/
```
