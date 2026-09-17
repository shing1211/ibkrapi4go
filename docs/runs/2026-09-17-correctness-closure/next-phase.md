# Next Phase: Correctness & Conformance Closure — Post-Mortem & Planning

**Run:** `2026-09-17-correctness-closure`
**Planner:** T6
**Date:** 2026-09-17

---

## 1. What Was Completed This Phase

Phase 9 (**Correctness & Conformance Closure**) resolved two classes of correctness bugs discovered during the mock-gateway development run:

- **D1 — Nil-`interface{}` panics in generated request builders.** 12 nil guards were applied to `client/client.gen.go` for five operations (`GetContractInfo`/6 fields, `GetConidsByExchange`/1 field, `GetAllFyis`/3 fields, `GetTradingSchedule2`/1 field, `ModifyFyiEmails`/1 field). Root cause: the spec-patch pipeline produces `interface{}` with `omitempty` for optional non-pointer params, but the `oapi-codegen` request-builder template does not guard nil for non-pointer interface types. A one-time toolchain-gap exception was recorded in `eb34ca1`; future fixes must go through `scripts/patch_spec.py` + `make codegen`.
- **D2 — 8 REST wrapper/decode mismatches.** Fixed in `pkg/ibkr/rest.go` and `pkg/ibkr/rest_utilities.go`: `TaxDocuments.Generate`, `TaxVouchers.CreateRequests`, `ActiveCountries`, `AvailableYears`, `Dividends`, `Utilities.Enumerations`, `ComplexAssetTransferBrokers`, and `RequiredForms`. Result committed at `adb3c1f`.

The repo now has **185/185 operations** covered, unified `slog` logging with redaction, a dependency-free OTel-compatible metrics layer, and an in-repo mock gateway (`cmd/ibkr-mock-gateway`, `examples/mock`). All phase-exit checks pass (`make check`, `make test-race`, `make docs-check`, `make license-check`).

---

## 2. Gaps, Tech Debt, and Deferred Items

| Item | Severity | Notes |
|------|----------|-------|
| **Benchmarks** | Medium | No `*_test.go` benchmark suite exists. HTTP latency, WebSocket throughput, and OAuth token-acquisition overhead are unmeasured. The mock gateway provides a stable target. |
| **Fuzz tests** | Medium | No fuzz coverage for JSON decode/encode round-trips, especially for `interface{}`-typed fields and edge-case `null` handling. |
| **Coverage badge** | Low | `make check` does not emit coverage; no badge in README. CI (`.github/workflows/ci.yml`) could be extended. |
| **Pre-commit hooks** | Low | No `pre-commit` configuration. Gofmt/vet/lint failures could be caught before push. |
| **Docs website** | Low | No hosted docs site. `godoc.org` / `pkg.go.dev` render the current docs, but a dedicated site would improve discoverability. |
| **GitHub Discussions** | Low | Not enabled on the repo; no community Q&A channel. |
| **FUNDING.yml** | Low | Not present; no GitHub Sponsors or sponsor button. |
| **D1 root cause (spec-patch)** | Low | `scripts/patch_spec.py` still produces `interface{}`+`omitempty` for optional non-pointer params. The fix was a one-time gen.go patch; the underlying pipeline defect is unfixed. |
| **CI already exists** | — | `.github/workflows/ci.yml` is already present and active; not a gap. |

---

## 3. Candidate Next-Phase Items

### C1: Benchmark Suite
**Objective:** Measure baseline performance for HTTP round-trips, WebSocket subscribe/unsubscribe, OAuth2 token acquisition, and rate-limiter overhead.
**Why now:** With the mock gateway and all 185 ops implemented, a stable, reproducible baseline can be established before any optimization work.
**Effort:** M
**Dependencies:** `internal/mockgateway` (already in-repo), `cmd/ibkr-mock-gateway`.
**Risks:** Benchmarks that become stale as the API evolves; need to integrate into CI or `make check` to stay current.

### C2: Fuzz Tests for JSON Round-Trips
**Objective:** Add `go-fuzz` coverage for all model types, focusing on `interface{}`-typed fields, `null` handling, and unexpected schema drift between spec and generated models.
**Why now:** D1 and D2 both stemmed from schema/codegen mismatches. Fuzzing is the most cost-effective way to catch similar issues early.
**Effort:** M
**Dependencies:** `client/client.gen.go` (stable), mock gateway fixtures.
**Risks:** Fuzz corpus maintenance; may require determinism guarantees from fixtures that are already scriptable in the mock gateway.

### C3: Coverage Badge + Extended CI Coverage Report
**Objective:** Emit `coverage.out` from `make check` (via `go test -coverprofile`) and display a badge in README.md backed by an artifact or coveralls-style service.
**Why now:** Quick win for project visibility; the CI workflow already exists and can be extended with a `coverage` job.
**Effort:** S
**Dependencies:** `.github/workflows/ci.yml`, README.
**Risks:** Low. May require choosing a coverage aggregation service if not self-hosting.

### C4: Pre-Commit Hooks
**Objective:** Add `pre-commit` configuration running `gofmt`, `go vet`, `make license-check`, and `make docs-check` on commit.
**Why now:** Catches style and doc-link errors before they reach CI; especially useful with multiple contributors.
**Effort:** S
**Dependencies:** None (no new dependencies by ADR 0004; can use `go run` wrappers or existing `make` targets).
**Risks:** False positives may frustrate contributors; need escape hatch (`--no-verify`).

### C5: Docs Website
**Objective:** Deploy a GitHub Pages site from `docs/` using a static generator (e.g., `mkdocs-material` or `docusaurus`), rendering godoc, design docs, and ADR.
**Why now:** The project has a rich `docs/` tree that is under-discoverable. A site would improve onboarding and API discoverability.
**Effort:** L
**Dependencies:** GitHub Pages setup, choice of static generator (adds a dependency — requires an ADR per ADR 0004).
**Risks:** Dependency addition; site drift from source docs; hosting/maintenance burden.

### C6: GitHub Discussions + FUNDING.yml
**Objective:** Enable GitHub Discussions for Q&A and add `FUNDING.yml` for GitHub Sponsors / sponsor button.
**Why now:** Low-effort community and sustainability signals.
**Effort:** S
**Dependencies:** None.
**Risks:** Low. Discussions may require moderation.

---

## 4. Recommended Next Phase

**Recommended focus: C1 (Benchmarks) + C2 (Fuzz Tests) — "Correctness & Performance Baseline"**

These two items share the mock gateway as infrastructure, both strengthen correctness guarantees beyond what the D1/D2 fixes achieved, and neither requires new dependencies. The work is bounded (M effort each) and produces artifacts (benchmark results, fuzz corpus) that integrate naturally into CI.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| T1 | Set up `benchmarks/` directory with `go test -bench` suite | backend | — | `BenchmarkHTTP_Static` and `BenchmarkWS_Subscribe` run against mock gateway; `make bench` target exists |
| T2 | Benchmark OAuth2 token acquisition + rate-limiter overhead | backend | T1 | Baseline numbers recorded in `docs/BENCHMARKS.md` |
| T3 | Add `go-fuzz` targets for all model types in `client/` | backend | — | Fuzz harness runs; corpus starts populating |
| T4 | Integrate fuzz + benchmarks into CI coverage job | ci | T1, T3 | `go test -fuzz` runs in CI; benchmark delta is computed |
| T5 | Docs: create `docs/BENCHMARKS.md`, update `SPEC.md` counts | docs | T2 | `make docs-check` passes |
| T6 | Close-out: commit, push, `next-phase.md` for next run | release | T4, T5 | both remotes at new commit |

---

## 5. Open Questions for the Human

1. **Benchmarks first or fuzz tests first?** C1 and C2 are independent but share the mock gateway. Should they be one phase or two sequential phases?
2. **Coverage badge service?** If no external service is desired, should a `coverage/` artifact be uploaded to the GitHub Actions run and linked from the README as a raw file?
3. **Docs website dependency?** Adding a static-site generator requires an ADR (ADR 0004). Does the project want to go down this path, or stay with `godoc.org`/`pkg.go.dev`?
4. **FUNDING.yml specifics?** Is there a GitHub Sponsors account to link, or should this be deferred until the project has a more active community?
5. **D1 root cause (spec-patch pipeline)?** Should a fix for the `scripts/patch_spec.py` `interface{}`+`omitempty` root cause be scheduled as a prerequisite to the next codegen run?
