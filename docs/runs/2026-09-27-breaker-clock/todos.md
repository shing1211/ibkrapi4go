# Todos

- [x] Verify the "backlog is empty" claim instead of repeating it.
- [x] Find the 0.0% package earlier runs skipped as "not worth it".
- [x] Establish that `internal/fake` has no importer, no test, and no external
      reachability.
- [x] Establish *why*: `internal.Clock` is a struct with unexported fields and no
      exported constructor, so `fake.Clock` cannot be passed to `SetClock`.
- [x] Write the deterministic breaker tests the seam was built for, driving the
      clock in-package where it can actually be constructed.
- [x] Pin the error budget's real semantics, including the time component it does
      not have.
- [x] Mutation-verify, and record the one case that cannot be detected.
- [x] Full verification sweep.
- [x] Surface `internal/fake` as a decision rather than delete it.
- [ ] Commit, tag, push - **held for explicit approval**.

## Awaiting a decision

- [ ] `internal/fake` - delete, or rewire. See the report; the two options are not
      equivalent and the changelog records it as an intended feature.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
