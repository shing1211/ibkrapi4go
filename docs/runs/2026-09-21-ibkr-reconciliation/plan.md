# Reconciliation Plan — IBKR API Coverage and Schema Fidelity

- **Run:** `2026-09-21-ibkr-reconciliation`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Review, reconcile, and fix gaps between the ibkrapi4go SDK and the live IBKR API surface.

## Findings

| Area | Status |
|------|--------|
| IB REST API coverage | ✅ 98.6% wrapped (69/70 endpoints) |
| CPAPI REST coverage | ✅ All 115 accessible via managers |
| WebSocket market data | ✅ Fully implemented |
| WebSocket order updates / notifications | 🔴 Not consumed from wire |
| Money/decimal fields (ADR 0008) | ⚠️ float64 violations in generated code |
| Numeric ID precision | ⚠️ float32 for ConIDs in some generated types |
| decodeJSON helper consistency | ⚠️ Not applied uniformly |

## Tasks

| ID | Task | Size | Depends |
|----|------|------|---------|
| R1 | Fix WS dispatch — add non-market-data frame routing | L | — |
| R2 | Patch scripts/patch_spec.py — fix float32 ConID → int64 | M | — |
| R3 | Add float64 → string wrapper for AccountSummary money fields | M | — |
| R4 | Make decodeJSON consistent across all REST managers | S | — |
| R5 | Add wrappers for high-value unwrapped CPAPI endpoints | M | — |
| R6 | Update SPEC.md — reconcile endpoint counts | S | R1–R5 |
| R7 | Commit + push | S | R1–R6 |
