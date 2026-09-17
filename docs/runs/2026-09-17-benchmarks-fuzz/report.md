# Report: Phase — Benchmarks + Fuzz Tests

- **Run:** `docs/runs/2026-09-17-benchmarks-fuzz/`
- **Mode:** BUILD
- **Base commit:** `6c5023b`
- **Release commit:** `c97a027` — `test(bench+fuzz): add benchmarks, fuzz tests, and CI regression gate`
- **Remotes:** `origin/main` + `gitee/main` both at `c97a027`
- **Status:** complete

## Shipped

| Task | Deliverable | Status |
|------|-------------|--------|
| T1 | Benchmarks: JSON encode/decode (20 types), HTTP RT (mock), WS subscribe/unsubscribe, session init | done |
| T2 | Fuzz tests: 47 `Fuzz*` functions covering all public JSON types; 185 op fixture decode fuzzing; ADR 008 money/quantity fuzzer | done |
| T3 | CI benchmark gate: `benchmark.baseline`, `scripts/bench_compare.go` (pure stdlib), `.github/workflows/ci.yml` benchmarks job | done |
| T4 | Docs sync: CHANGELOG.md, OBSERVABILITY.md (Benchmarks + Fuzz sections), ROADMAP.md (backlog update) | done |
| T5 | Release: committed and pushed to both remotes | done |
| T6 | Next-phase planning: CI Quality Gates bundle recommended | done |

### Key numbers
- **20** JSON encode/decode benchmarks across all major public types
- **6** HTTP/WS/session benchmarks (p50/p95/p99 latency reported)
- **47** `Fuzz*` functions
- **0** panics found across all fuzz inputs
- **315** baseline benchmark entries in `benchmark.baseline`
- **>10%** regression threshold in CI gate

### Verification (final)
`make check` · `make test-race` · `make docs-check` · `make license-check` — all pass.

## Next recommended phase
**CI Quality Gates + Developer Experience**: coverage badge, pre-commit CI gate,
GitHub Discussions, `FUNDING.yml`. See `next-phase.md`.
