# Todos

- [x] Re-state that the credential-blocked items cannot move from here, rather than
      re-deriving it a third time.
- [x] Inventory what is left of the `json.Number` class: 5 fields in `marketdata.go`,
      3 in `contract.go`.
- [x] `number_precision_test.go` - bar OHLCV, strike slices, contract multiplier.
- [x] Reuse `float32Loses` so the new file rejects undetectable values too.
- [x] Mutation-verify all five conversions, each on the assertion.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## The class is now closed

- [x] account summary, positions, ledger - 1.1.18 / 1.1.19
- [x] tax voucher dividends - 1.1.9
- [x] historical bars, option strikes, contract multiplier - this run

No `json.Number` field reaching a caller as a string is left without a
`float32`-changing assertion.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
