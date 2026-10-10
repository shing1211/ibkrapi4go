# todos.md — 15-Phase Enhancement Run

- **Run:** `2026-09-20-next-15-phases`
- **Mode:** BUILD
- **Last updated:** 2026-09-20

Single source of truth for all tasks in this run. Update after every task state change.

## Phase tracking

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| S1 | Goroutine safety audit | tester | done | — | goleak on all async paths; no leaks |
| A1 | Interface segregation | architect | done | — | Interfaces for TokenSource, RoundTripper, SessionStateMachine |
| A2 | Typed builders | backend | done | — | OrderBuilder, ContractBuilder, TransferInstructionBuilder |
| S2 | Backpressure & deadline propagation | backend | done | S1 | ctx.Deadline honored everywhere; WithRequestTimeout added |
| E1 | CLI tool | devops | done | S1 | `ibkr` binary with accounts/positions/orders/stream commands |
| A3 | Middleware plugin API | architect | done | A1 | WithTransportMiddleware option; middleware chain test |
| E2 | Migration guide v0.x → v1.0 | docs | done | S1 | docs/MIGRATION.md; codemod script |
| S3 | Production resilience | backend | done | S1 | Per-endpoint timeout; circuit breaker tuning; panic recovery |
| S4 | Test quality & fault injection | tester | done | — | Fault-injection tests; mutagen coverage ≥80% |
| A4 | Unified `*Pager[T]` pagination | backend | done | — | Pager[T] type; AllModels/AllFYIs/Transactions use it |
| E3 | API reference site | docs | done | E1 | mkdocs-gen site; GitHub Pages; searchable |
| E4 | Release engineering automation | devops | done | — | Changelog from commits; semantic-version CI enforcement |
| E5 | Supply-chain security | security | done | — | SBOM; dependency pinning; secret scanning |
| S5 | Performance baseline | backend | done | S2 | HTTP/2 pooling; zero-allocation paths; baseline stored |
| A5 | Multi-client / shared transport | architect | done | A1 | TransportPool; shared rate limiter; thread-safe |

## Phase 5 artifacts

| ID | Artifact | Role | Status | Depends On |
|----|----------|------|--------|------------|
| D1 | docs-plan.md | docs | done | plan.md |
| D2 | release-plan.md | release | done | plan.md |
| D3 | next-phase.md | planner | done | plan.md |

## Run close-out

| ID | Artifact | Role | Status |
|----|----------|------|--------|
| R1 | report.md | orchestrator | done |
| R2 | index.md update | orchestrator | done |
| R3 | git commit + push | release | done |
