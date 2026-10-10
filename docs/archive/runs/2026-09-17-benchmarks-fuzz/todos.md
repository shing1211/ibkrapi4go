# Todos: Benchmarks + Fuzz Tests

Run: `docs/runs/2026-09-17-benchmarks-fuzz/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `done`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T1 | Benchmarks: JSON encode/decode, HTTP RT, WS subscribe, session init | backend | done | — | 20 JSON benchmarks + HTTP/WS/Session benchmarks; stable numbers |
| T2 | Fuzz tests: unmarshal all public types, 185 op response decode | backend | done | T1 | 47 fuzz functions; no panics; corpus generated |
| T3 | CI: benchmark comparison gate in `ci.yml` | devops | done | T1 | ci.yml updated; benchmark.baseline + bench_compare.go created |
| T4 | Docs sync | docs | done | T2 | CHANGELOG.md, OBSERVABILITY.md, ROADMAP.md updated; `make docs-check` pass |
| T5 | Release: commit + push both remotes | release | done | T4 | both remotes at c97a027; pushed to origin + gitee |
| T6 | Next-phase planning | planner | doing | T5 | `next-phase.md` |
| T7 | Close-out report + index | orchestrator | todo | T6 | `report.md` + index line |

## Notes
- Base commit: `6c5023b`
- Use `internal/mockgateway` as deterministic backend (already at `eb34ca1`)
- No new dependencies; stdlib `testing.B`/`testing.F`
- Benchmark baseline stored in repo; CI compares against it
