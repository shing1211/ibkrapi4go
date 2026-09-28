# Todos

- [x] Re-verify D15 has no offline half left - the client-side omission of `year`
      is already asserted at `rest_reports_e2e_test.go:296`.
- [x] Determine which injection path the SDK's retry tests actually depend on:
      `SetPolicy` + `applyScenario`, not `Scenario.Set`/`SetGlobal`.
- [x] `scenario_test.go` - `FaultFor` precedence (policy, per-op, global), every
      setter/getter, the zero-value `Scenario`, and all four injection modes.
- [x] `recorder_test.go` - deep-copy per field, nil guards, slice copy, reset,
      concurrent access, and the pre-routing contract.
- [x] `stream_parse_test.go` - field list, conid list and frame parsing across the
      JSON and legacy text wire shapes.
- [x] Mutation-verify 15 mutations across all three areas.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account. The offline half is already covered.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
