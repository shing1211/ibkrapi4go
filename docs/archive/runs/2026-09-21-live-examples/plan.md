# P3 — Real-World Example Suite

- **Run:** `2026-09-21-live-examples`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Add 3 runnable read-only examples against live paper trading credentials, complementing the existing 6 mock-gateway examples.

## Examples

| ID | Path | Size | What it demonstrates |
|----|------|------|----------------------|
| L1 | `examples/live-portfolio/` | S | Session.Initialize → Account.List → Portfolio.Positions/Ledger/Summary |
| L2 | `examples/options-chain/` | M | `ContractSymbolsFromBody("AAPL")` → `SecDefInfos()` → show strikes × expiries |
| L3 | `examples/screener/` | M | `ScannerParameters()` → inspect live filters → `ScannerResults()` |

## Design

- Read-only only — no orders, no modifications, no transfers
- Same env vars as `test/integration_test.go`: `IBKR_GATEWAY`, `IBKR_USERNAME`, `IBKR_PASSWORD`
- Examples live in root module (no separate `go.mod`)
- No new dependencies

## Task breakdown

| ID | Task | Size | Status |
|----|------|------|--------|
| L1 | `examples/live-portfolio/main.go` | S | todo |
| L2 | `examples/options-chain/main.go` | M | todo |
| L3 | `examples/screener/main.go` | M | todo |
| L4 | Update `examples/README.md` | S | todo |
| L5 | Commit + push | S | todo |
