# P2 — OTel First-Class Support

- **Run:** `2026-09-21-otel-first-class`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Ship a first-class `contrib/otel` package that bridges `ibkr.Metrics` → OpenTelemetry, with a convenience constructor. The OTel bridge is documented in OBSERVABILITY.md as an example but not shipped as code.

## Constraint

ADR 0004 forbids new runtime deps in core. Solution: separate `contrib/otel` module with its own `go.mod`.

## Task breakdown

| ID | Task | Role | Size | Depends |
|----|------|------|------|---------|
| O1 | Create `contrib/otel/go.mod` | backend | S | — |
| O2 | Implement `OTelMetrics` bridge | backend | M | O1 |
| O3 | Unit tests for OTelMetrics | tester | M | O2 |
| O4 | Example `examples/otel/main.go` | docs | S | O2 |
| O5 | Update OBSERVABILITY.md | docs | S | O2 |
| O6 | Commit + push | release | S | O3,O4,O5 |
