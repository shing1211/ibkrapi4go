# Next Phase: Post-spec-drift-watch Directions

- **Run:** `2026-09-21-spec-drift-watch`
- **Date:** 2026-09-21

## 1. What Was Completed

Implemented a spec drift watch: `scripts/check_spec_version.py` (fetches live
spec, compares against pinned v2.40.0, exits 0/1/2) and
`.github/workflows/spec-drift.yml` (weekly Monday 9am UTC + manual dispatch,
opens GitHub issue on drift). Both remotes at `6e2be91`.

## 2. Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| `docs/STABILITY.md` Pre-1.0 rules block | Low | Lines 74–78 still reference v0.x rules; flagged in docs-reconciliation run |
| Pre-existing dangling links in run docs | Low | `docs/runs/2026-09-20-next-15-phases/plan.md` uses `../` paths |
| Community/Discussions welcome post | Low | GitHub Discussions is live but empty |
| OpenAPI spec v2.41.0+ watch | Medium | When IBKR releases a new spec, the workflow will open an issue automatically |

## 3. Candidate Next Phases

### P1 — Community Seeding
**Objective:** Post a Discussions welcome/announcement, seed "good first issue"
labels, and refresh `CONTRIBUTING.md` with an architecture tour.
**Why now:** Discussions is live but empty; a cheap adoption and sustainability
win. The spec drift watch is now running — the next maintenance risk is
community engagement.
**Effort:** S
**Dependencies:** None.
**Risks:** Requires ongoing moderation.

### P2 — Real-World Examples & Tutorials
**Objective:** Add non-mock examples (portfolio snapshot, market scanner run,
option chain fetch) plus a short tutorial doc.
**Why now:** Lowers adoption barrier; complements the existing mock example.
**Effort:** M
**Dependencies:** None (mock gateway available for tests).
**Risks:** Examples must not imply investment advice; align with `DISCLAIMER.md`.

### P3 — `docs/STABILITY.md` cleanup
**Objective:** Remove the Pre-1.0 rules block (lines 74–78) flagged during the
docs-reconciliation run.
**Why now:** Quick win; the block is stale and potentially confusing.
**Effort:** S
**Dependencies:** None.
**Risks:** None.

## 4. Recommended Next Phase

**Recommended: P3 (`docs/STABILITY.md` cleanup) — S effort.**

Quick, bounded, no risk. Removes the stale Pre-1.0 rules block that was flagged
during the docs-reconciliation run. Can be done in one sub-agent session.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| C1 | Remove stale Pre-1.0 rules block from `docs/STABILITY.md` | docs | — | Block removed; doc still coherent |
| C2 | Commit + push | release | C1 | Both remotes updated |

## 5. Open Questions

1. **Community post scope:** should the Discussions welcome post also cover the
   spec-drift workflow announcement?
2. **Real-world examples:** which workflow is most valuable to users — portfolio
   snapshot, market scanner, or options chain?
