# todos.md — P2 OTel First-Class Support

- **Run:** `2026-09-21-otel-first-class`
- **Mode:** BUILD

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| O1 | Create `contrib/otel/go.mod` | backend | done | — | Module compiles, depends on SDK + OTel |
| O2 | Implement `OTelMetrics` bridge | backend | done | O1 | Counter/Histogram/Gauge → OTel instruments |
| O3 | Unit tests for OTelMetrics | tester | done | O2 | All 3 methods tested; concurrent-safe |
| O4 | Example `contrib/otel/examples/otel/main.go` | docs | done | O2 | Compiles and demonstrates usage |
| O5 | Update OBSERVABILITY.md | docs | done | O2 | References contrib package |
| O6 | Commit + push | release | done | O3,O4,O5 | `696c8f4` — both remotes updated |
