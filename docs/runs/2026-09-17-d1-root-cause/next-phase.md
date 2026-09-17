# Next Phase: Post-v0.1.0 Directions

- **Run:** `2026-09-17-d1-root-cause`
- **Planner:** D6
- **Date:** 2026-09-17

## 1. What Was Completed

This run closed the last open correctness item in the codebase: the
nil-`interface{}` panic class in generated request builders is now fixed at the
spec level (`patch_spec.py` defect 4), the post-generation `patch_gen.py` is a
no-op, and `client/client.gen.go` was regenerated with typed params. With this
change:

- All 185/185 operations are implemented and stable.
- No known correctness defects remain open.
- `make check`, `make codegen-verify`, `make docs-check`, `make license-check` pass.
- v0.1.0 is released on GitHub and mirrored to Gitee.

The correctness and developer-experience backlog is empty. Remaining work is
forward-looking rather than remedial.

## 2. Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| **GitHub Discussions welcome post** | Low | Discussions is enabled with default categories; no announcement post yet. |
| **Public API stabilization** | Medium | No formal statement of what is frozen. Pre-1.0 rules exist in `RELEASING.md` but the exported surface has not been audited as a candidate 1.0. |
| **IBKR spec drift watch** | Medium | `codegen-verify` runs on a schedule/PR but nothing alerts when the upstream spec version advances past v2.39.0. |
| **Real-world examples** | Low | `examples/mock` exists; no end-to-end examples beyond the mock (portfolio, scanner, options chain). |
| **Performance baseline interpretation** | Low | Benchmarks + a CI regression gate exist; no targeted optimization pass has been attempted. |
| **`type: null` non-query params** | Low | Defect 4 covers inline **query** params. Inline `type: null` in other locations (if any) is not retyped; none are currently observed. |

## 3. Candidate Next Phases

### P1 — v1.0.0 Readiness & API Stabilization
**Objective:** Audit the exported surface of `pkg/ibkr` and `client/`, document
the stability contract, finalize the deprecation policy, and publish a v1.0.0
readiness checklist.
**Why now:** The feature set is complete and correctness defects are closed; the
natural next milestone is a stable public API.
**Effort:** M
**Dependencies:** None.
**Risks:** Renaming exported symbols to improve ergonomics would be a breaking
change; best done before 1.0, but needs care.

### P2 — Upstream Spec Drift Watch
**Objective:** Add a scheduled job (or script) that compares the live IBKR spec
version against the pinned v2.39.0 and opens an issue when it advances.
**Why now:** The SDK is only as current as its spec; silent drift is the main
long-term maintenance risk.
**Effort:** S
**Dependencies:** Existing `codegen-verify` workflow.
**Risks:** Upstream spec changes may require new patch rules; the job should
report, not auto-regenerate.

### P3 — Real-World Examples & Tutorials
**Objective:** Add non-mock examples (portfolio snapshot, market scanner run,
option chain fetch) plus a short tutorial doc linking them.
**Why now:** Lowers the adoption barrier; complements the existing mock example.
**Effort:** M
**Dependencies:** None (mock gateway available for tests).
**Risks:** Examples must not imply investment advice; align with `DISCLAIMER.md`.

### P4 — Community & Sustainment
**Objective:** Post a Discussions welcome/announcement, seed "good first issue"
labels, and refresh `CONTRIBUTING.md` with an architecture tour.
**Why now:** Discussions is live but empty; cheap adoption and sustainability win.
**Effort:** S
**Dependencies:** None.
**Risks:** Requires ongoing moderation.

## 4. Recommended Next Phase

**Recommended: P1 (v1.0.0 Readiness & API Stabilization).**

The project has reached the point where correctness and coverage are settled;
the highest-value next step is to decide and document what "stable" means, audit
the exported surface against that bar, and prepare a 1.0 checklist. This is
bounded, needs no new dependencies, and directly informs whether P3/P4 work is
worth doing under a stable API.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| V1 | Inventory and review all exported symbols in `pkg/ibkr`; flag ergonomics/naming issues | backend | — | review doc listing candidates and rationale |
| V2 | Document the 1.0 stability contract and deprecation policy in `RELEASING.md` | docs | V1 | policy section; `make docs-check` passes |
| V3 | Add a `docs/STABILITY.md` (or ADR) recording the frozen surface and exceptions | docs | V2 | doc linked from README/ROADMAP |
| V4 | Publish a v1.0.0 readiness checklist and resolve or defer each item | orchestrator | V3 | checklist in the run report |
| V5 | Release + close-out | release | V4 | pushed to both remotes; run indexed |

## 5. Open Questions for the Human

1. **Is v1.0.0 the goal now, or should the project stay at v0.x while the API
   settles through community use?**
2. **Breaking changes before 1.0:** are any exported names worth changing (e.g.
   manager accessor names) while still pre-1.0, or is the current surface
   considered final?
3. **Spec drift:** should the drift watch open an issue, a PR, or simply log to
   CI on a schedule?
4. **Examples scope:** which workflows matter most (portfolio, trading, market
   data, FA/model portfolios)?
5. **Community:** should the Discussions welcome post and "good first issue"
   seeding be combined with the stability work or run separately?
