# Next Phase: Post-v1.0.0 Directions

- **Run:** `2026-09-21-docs-reconciliation`
- **Date:** 2026-09-21

## 1. What Was Completed

This run reconciled all markdown documentation against the actual codebase.
The repo had advanced to v1.0.0 (via intermediate v0.2.0, v0.3.0 releases) but
documentation still referenced v0.1.0–v0.2.0. Fixed 11 files: version constants,
status badges, release rows, alpha/pre-alpha claims, and the CHANGELOG compare
link. Both remotes are at `8af8404`.

## 2. Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| Pre-existing dangling links in run docs | Low | `docs/runs/2026-09-20-next-15-phases/plan.md` uses `../` relative paths that break when docs are served from a subdir path. Not blocking. |
| `docs/STABILITY.md` Pre-1.0 rules block | Low | Lines 74–78 still reference v0.x rules; should be removed or marked historical. |
| Upstream spec drift watch | Medium | Spec is pinned at v2.40.0; no automated alert when IBKR releases v2.41.0+. |
| Community/Discussions | Low | GitHub Discussions is live but empty; no welcome post. |

## 3. Candidate Next Phases

### P1 — Spec Drift Watch
**Objective:** Add a scheduled job or script that compares the live IBKR spec
version against the pinned v2.40.0 and opens an issue when it advances.
**Why now:** The SDK is only as current as its spec; silent drift is the main
long-term maintenance risk.
**Effort:** S
**Dependencies:** Existing `codegen-verify` workflow.
**Risks:** Upstream spec changes may require new patch rules; the job should
report, not auto-regenerate.

### P2 — Community Seeding
**Objective:** Post a Discussions welcome/announcement, seed "good first issue"
labels, and refresh `CONTRIBUTING.md`.
**Why now:** Discussions is live but empty; a cheap sustainability win.
**Effort:** S
**Dependencies:** None.
**Risks:** Requires ongoing moderation.

### P3 — Real-World Examples & Tutorials
**Objective:** Add non-mock examples (portfolio snapshot, market scanner run,
option chain fetch) plus a short tutorial doc.
**Why now:** Lowers adoption barrier; complements the mock example.
**Effort:** M
**Dependencies:** None (mock gateway available).
**Risks:** Must not imply investment advice; align with `DISCLAIMER.md`.

### P4 — v1.0.1 Patch Release
**Objective:** If any regressions or spec misalignment were introduced by the
v1.0.0 release, address them in a v1.0.1 patch.
**Why now:** v1.0.0 was a major breaking-change release; a quick patch addresses
any early adopter feedback.
**Effort:** S
**Dependencies:** None.
**Risks:** May not be needed if no issues are reported.

## 4. Recommended Next Phase

**Recommended: P1 (Spec Drift Watch).**

The SDK is feature-complete and stable at v1.0.0. The highest-value next step
is establishing a watch that keeps the spec current. This is a small, bounded
script that prevents silent bitrot.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| S1 | Script `scripts/check_spec_version.py` to fetch live spec and compare | backend | — | script runs; exits 0 if current, 1 if drifted |
| S2 | Wire into CI as a scheduled job (no new deps) | devops | S1 | job visible in GitHub Actions |
| S3 | Commit + push | release | S2 | pushed to both remotes |

## 5. Open Questions

1. **Spec drift action:** should the job open a GitHub issue automatically,
   or just log/fail CI?
2. **Community:** should the Discussions welcome post happen before or after
   the spec watch work?
