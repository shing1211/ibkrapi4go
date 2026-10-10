# Todos

- [x] Prototype the literal-scanning rule. It flags 27, almost all false
      positives, and misses the one real case.
- [x] Prototype the type-keyed rule. Right invariant, but 16 false negatives from
      table-driven assertions.
- [x] Withdraw the gate recommendation rather than ship an unsound lint.
- [x] Close the real remaining gap: 7 of 28 monetary `json.Number` fields on the
      account summary and ledger had no lossy-value assertion.
- [x] Mutation-verify each of the 7 individually, confirming an assertion failure
      rather than a build error.
- [x] Record the failure mode in the test file, so the set reads as exhaustive.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
