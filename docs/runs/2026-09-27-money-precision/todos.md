# Todos

- [x] Re-verify the blocked list - unchanged, both need credentials.
- [x] Scan the money-bearing test files for literals that a `float32` renders
      unchanged: 53 literals, 38 undetectable, 2 deliberately lossy.
- [x] Determine which undetectable literals actually matter. Banking and transfer
      money is `string`-typed, so a `float32` regression is a compile error there.
- [x] Isolate the real gap: 39 `json.Number` response money fields across the
      account summary, positions and ledger, with no lossy-value test.
- [x] Write precision tests for all three responses.
- [x] Reject values that survive the `float32` round trip unchanged - the guard
      caught `0.007` in my own first draft.
- [x] Mutation-verify all five conversions, and confirm each failed on the
      assertion rather than a build error.
- [x] Check whether quoted JSON strings decode into `json.Number`. They do; my
      recollection was wrong and the probe corrected it.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
