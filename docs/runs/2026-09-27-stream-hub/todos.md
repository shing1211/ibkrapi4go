# Todos

- [x] Read `Broadcast`, `closeAll`, `sendFrame` and the connection handler to
      establish the actual semantics rather than infer them.
- [x] Pin the fan-out count, and that each client actually receives the frame.
- [x] Pin the silent closed-connection skip in `sendFrame`.
- [x] Pin that `closeAll` drains the hub.
- [x] Establish that the lock discipline cannot be tested soundly here, after three
      attempts, two of which passed with the lock held.
- [x] Remove the two vacuous tests rather than ship them.
- [x] Record the untestable requirement on `Broadcast` and `closeAll` as a review
      rule, explicitly weaker than a test.
- [x] Mutation-verify the three surviving tests, four mutations.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account.
