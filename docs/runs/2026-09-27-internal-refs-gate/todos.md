# Todos

- [x] Confirm the class is decidable with no heuristics, unlike the withdrawn gate.
- [x] Write `scripts/check_internal_refs.py` with a self-test.
- [x] Wire into `make internal-refs-check`, `make check`, CI, and AGENTS.md.
- [x] Verify against the real tree: unreferenced package fails and is named.
- [x] Verify a test-only importer satisfies the rule - the rule is not "no new
      packages".
- [x] Verify removal restores green.
- [x] Two real bugs in the checker itself, both found by that verification.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
- [ ] `internal/fake` delete-or-rewire - maintainer decision, now enforced as a
      visible `ALLOWED` entry rather than left implicit.
