# v0.3.0 — Ergonomics & Type Safety Breaking Release

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** In Progress

## Goal

Fix all high and medium priority naming/type-safety issues in a single breaking release while the user base is small.

## Task Breakdown

| ID | Task | Role | Size | Depends |
|----|------|------|------|---------|
| A1a | Remove `Get` prefix — ScannerManager, TradingAccountManager, ModelManager, SessionManager, WatchlistManager, AllocationManager, FYIManager | backend | M | — |
| A1b | Remove `Get` prefix — TradeManager (13 methods) | backend | M | — |
| A1c | Remove `Get` prefix — PerformanceManager, AlertManager, ForecastManager, PortfolioManager, RESTRequests | backend | M | — |
| B1 | Fix Go initialism casing (14 symbols) + `ReqAccessToken` abbreviation | backend | S | — |
| C1 | AccountID type consistency — 21 exported fields | backend | M | — |
| D1 | float32 → int64 for banking IDs (4 fields) + call sites | backend | S | — |
| E1 | ConID type consistency (4 fields) + call sites | backend | S | — |
| F1 | Remove dead `RESTInstructions` type | backend | S | — |
| G1 | Remove deprecated `ErrStream*` aliases | backend | S | — |
| H1 | Fix method/type name collisions (`GetContractInfo`, `GetAlertDetails`) | backend | S | A1b |
| I1 | Update all tests for renamed symbols | tester | M | A1a–H1 |
| I2 | Verify: `make check` + race tests | tester | S | I1 |
| I3 | CHANGELOG + commit + tag v0.3.0 + push | release | S | I2 |

## Verification

```bash
make check
python3 scripts/check_money.py
go test -race -count=1 ./...
```
