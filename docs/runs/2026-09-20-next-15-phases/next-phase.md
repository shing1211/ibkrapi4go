# next-phase.md — What Comes After the 15 Phases

- **Run:** `2026-09-20-next-15-phases`
- **Date:** 2026-09-20
- **Mode:** BUILD

## What this run completes

This run plans 15 enhancement phases across three non-functional dimensions:

- **Stability Hardening (S1–S5):** goroutine safety, backpressure, resilience,
  fault-injection testing, performance baseline
- **Ecosystem (E1–E5):** CLI tool, migration guide, API reference site, release
  automation, supply-chain security
- **Architectural (A1–A5):** interface segregation, typed builders, middleware
  plugin, unified pagers, multi-client transport

These phases address the highest-value non-functional gaps identified in the
v0.2.0 codebase. Completion of all 15 is the prerequisite for v1.0.0
stabilization.

## Gaps and tech debt observed

| Item | Severity | Note |
|------|----------|------|
| Internal package coupling | Medium | `internal/` packages are concrete; testing requires full stack |
| No order builder ergonomics | Medium | 10+ field structs set manually today |
| No pagination beyond `PositionIterator` | Low | 10+ list ops return raw slices |
| No first-class OpenTelemetry | Low | Telemetry hooks exist but no OTel SDK integration |
| No real-world benchmark baseline | Low | Mock benchmarks only; real I/O untested |
| Multi-account requires separate `Client` | Low | No shared transport across accounts |

## Candidate next phases (beyond 15)

### P2 — OTel First-Class Support

Add native OpenTelemetry tracing and metrics as a first-class `Option`
(`WithOTelTracer`, `WithOTelMeter`). Currently the telemetry hooks are
interface-only with no official SDK implementation.

**Why now:** Users want distributed traces; the hook infrastructure exists but
no official integration. **Effort:** M. **Depends on:** A1 (interface
segregation).

### P3 — Real-World Example Suite

Add runnable examples against live paper trading: `examples/live-portfolio`,
`examples/options-chain`, `examples/screener`. The mock examples exist but
don't exercise real market data.

**Why now:** Reduces adoption barrier; paper trading is safe. **Effort:** M.
**Depends on:** E2 (migration guide — ensures examples work across versions).

### P4 — Community & Sustainment

Seed GitHub Discussions, label "good first issue", refresh `CONTRIBUTING.md`
with architecture tour, add `FUNDING.yml` details.

**Why now:** Discussions is live but empty; cheap win. **Effort:** S.
**Depends on:** None.

### P5 — v1.0.0 Final Stabilization

After all 15 phases complete: final API audit, CHANGELOG sweep, tag `v1.0.0`,
push to both remotes.

**Why now:** The natural conclusion of the stabilization work. **Effort:** S.
**Depends on:** All 15 phases.

### P6 — Multi-gateway / Session Pool

Allow one `Client` to manage sessions across multiple IBKR accounts or
gateways simultaneously — e.g., aggregate positions across a family of accounts.

**Why now:** ARCHITECTURE.md lists it as a future item; A5 lays groundwork.
**Effort:** L. **Depends on:** A5 (multi-client transport).

## Recommended next phase

**Recommended: v1.0.0 Final Stabilization (P5)** after all 15 complete — or
alternatively, **OTel First-Class Support (P2)** in parallel with the 15-phase
execution (P2 depends on A1 which is early in the sequence).

## Open questions for the human

1. Should OTel integration (P2) be a standalone phase or folded into one of the
   15 (e.g., E4 as observability in CI)?
2. Real-world examples (P3) — should they require paper trading credentials or
   use the mock with real-looking data?
3. Community (P4) — should this be done before v1.0.0 or after?
