# Plan: Benchmarks + Fuzz Tests

- **Run:** `2026-09-17-benchmarks-fuzz`
- **Mode:** BUILD
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `6c5023b`

## Goal
Establish a performance and correctness baseline using `testing.B` benchmarks and
`testing.F` fuzz tests, powered by the existing `internal/mockgateway` (already
committed in run `mock-gateway/eb34ca1`). No network, no real credentials needed.

## Scope
- `pkg/ibkr/` benchmarks: JSON encode/decode, all public response types
- `internal/` benchmarks: HTTP round-trip latency (mock), WebSocket subscribe/unsubscribe,
  session init
- Fuzz tests: JSON unmarshal into all public `pkg/ibkr` types; response body decode
  across all 185 op response shapes using the mock
- Optional CI gate: benchmark comparison in `.github/workflows/ci.yml`

## Assumptions
- `testing.B`/`testing.F` in-tree is acceptable (standard Go).
- The mock gateway is already committed and stable (`eb34ca1`).
- No new dependencies; fuzzing uses stdlib only.
- Benchmark results are compared to a baseline stored in the repo.

## Task breakdown
| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| T1 | Benchmarks: JSON encode/decode, HTTP RT, WS subscribe, session init | backend | — | `Bench*` pass; stable numbers |
| T2 | Fuzz tests: unmarshal all public types, 185 op response decode | backend | T1 | `Fuzz*` pass; corpus generated |
| T3 | CI: benchmark comparison gate in `ci.yml` | devops | T1 | ci.yml updated; regression detection works |
| T4 | Docs sync: CHANGELOG, any stale refs | docs | T2 | `make docs-check` pass |
| T5 | Release: commit + push both remotes | release | T4 | both remotes at new commit |
| T6 | Next-phase planning | planner | T5 | `next-phase.md` |
| T7 | Close-out report + index | orchestrator | T6 | `report.md` + index line |

## Verification (global)
`make check` · `make test-race` · `make docs-check` · `make license-check`
