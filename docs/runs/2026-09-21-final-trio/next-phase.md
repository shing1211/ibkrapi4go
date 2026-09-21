# Next Phase: Post-Final-Trio Directions

- **Run:** `2026-09-21-final-trio`
- **Date:** 2026-09-21

## 1. What Was Completed

This run completed three independent tasks:
- **v1.0.1 patch release** — codegen precision fixes (ConID int64, money as string,
  decodeJSON consistency) and WS system frame routing shipped as v1.0.1
- **Community seeding** — CONTRIBUTING.md now has a "Finding a First Issue"
  section; GitHub Discussions welcome post live
- **Examples** — `examples/mock/error-handling.go` (retry, error types, graceful
  degradation) and `examples/live/oauth2-flow.go` (OAuth2 token lifecycle) added

Both remotes at `d95c6fe`.

## 2. Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| Gitee v1.0.1 release entry | Low | Manual via Gitee web UI; no CLI/token |
| Pre-existing dangling links | Low | `docs/runs/2026-09-20-next-15-phases/plan.md` uses `../` paths |
| Example test coverage | Low | No `*_test.go` in `examples/` verifying compilation |
| Real-world example: options chain | Low | options-chain live example exists but could use a tutorial doc |

## 3. Candidate Next Phases

### P1 — Real-World Example Tutorial Doc
**Objective:** Write a short `docs/TUTORIAL.md` (or `examples/README.md` section) that
guides users through one workflow end-to-end using the mock gateway.
**Why now:** The examples exist but there's no narrative tutorial; users must
reverse-engineer from code.
**Effort:** S
**Dependencies:** None (mock gateway available).
**Risks:** None.

### P2 — Example Test Coverage
**Objective:** Add `examples/*_test.go` files that verify each example compiles
and runs against the mock gateway.
**Why now:** No CI guard ensuring examples stay runnable; a common source of
bitrot.
**Effort:** M
**Dependencies:** Mock gateway.
**Risks:** Tests may be flaky if gateway stateful.

### P3 — Fix Pre-Existing Dangling Links
**Objective:** Fix the `../` relative links in `docs/runs/2026-09-20-next-15-phases/plan.md`.
**Why now:** Pre-existing issue; `check_links.py` reports it on every run.
**Effort:** S
**Dependencies:** None.
**Risks:** None.

## 4. Recommended Next Phase

**Recommended: P3 (Fix Pre-Existing Dangling Links) — S effort.**

The dangling links in `docs/runs/2026-09-20-next-15-phases/plan.md` are reported
by `check_links.py` on every run. Quick win, no risk. Can be done in one session.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| D1 | Fix `../` relative links in `docs/runs/2026-09-20-next-15-phases/plan.md` | docs | — | `check_links.py` passes |
| D2 | Commit + push | release | D1 | Both remotes updated |

## 5. Open Questions

1. **Tutorial scope:** which workflow to document — portfolio snapshot, market
   scanner, or something else?
2. **Gitee release:** do you want to create the Gitee release entry manually?
