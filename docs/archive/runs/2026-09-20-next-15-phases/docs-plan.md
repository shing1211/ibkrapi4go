# docs-plan.md — Documentation Sync Plan

- **Run:** `2026-09-20-next-15-phases`
- **Date:** 2026-09-20
- **Mode:** BUILD

## Scope

Review and update all project Markdown files to reflect the current state after
the 15-phase plan is executed. Since this is a PLAN run (no code changes yet),
this document lists what WILL need updating when phases are executed.

## Files to review

| File | Action | Reason |
|------|--------|--------|
| `README.md` | Review | May reference the old CLI or API surface |
| `docs/ARCHITECTURE.md` | Update | A1/A2/A3/A5 introduce new architectural patterns; add to "Future" section |
| `docs/ROADMAP.md` | Update | Add S1–S5, E1–E5, A1–A5 as planned phases; mark P1 (v1.0 readiness) in progress |
| `docs/STABILITY.md` | Review | S3 may add new stability guarantees |
| `docs/RELEASING.md` | Update | E4/E5 add release automation and supply-chain steps |
| `docs/runs/index.md` | Update | Append this run's entry |
| `CHANGELOG.md` | Update | E4 adds changelog-from-commits automation |
| `CONTRIBUTING.md` | Review | May reference old PR process |
| `AGENTS.md` | Review | Hard rules remain; no changes expected |
| `examples/README.md` | Update | E1 adds CLI tool; may need new example |
| `docs/MOCK-GATEWAY.md` | Review | Unchanged unless A5 affects mock |
| `docs/adr/*.md` | Review | New ADRs may be needed for A1, A3, E4 |
| `docs/runs/2026-09-17-d1-root-cause/next-phase.md` | Update | Replace with reference to this run's next-phase.md |

## No change needed (with reason)

| File | Reason |
|------|--------|
| `docs/CODEGEN.md` | Codegen process unchanged |
| `docs/SESSIONS.md` | Session state machine unchanged |
| `docs/RATE-LIMITING.md` | Rate limiter unchanged |
| `docs/OBSERVABILITY.md` | Metrics/Observability unchanged |
| `docs/LOGGING.md` | Logging unchanged |
| `docs/ERRORS.md` | Error taxonomy unchanged |
| `docs/GLOSSARY.md` | Glossary unchanged |
| `docs/TESTING.md` | Testing strategy unchanged |

## ADR candidates

If the following phases land, new ADRs are required:

| Phase | New ADR |
|-------|---------|
| A1 Interface segregation | ADR 0018 — Interface boundaries in `internal/` |
| A3 Middleware plugin API | ADR 0019 — Transport middleware extension |
| E4 Release automation | ADR 0020 — Automated changelog and versioning |
| E5 Supply-chain security | ADR 0021 — SBOM and provenance policy |

## Execution note (BUILD mode)

When this run executes in BUILD mode, a `docs` sub-agent will be spawned to
perform the actual updates per the table above.
