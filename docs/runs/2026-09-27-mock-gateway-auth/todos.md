# Todos

- [x] Re-verify the three carried-forward blocked items rather than asserting them.
- [x] Run `contextcheck` with the WebSocket exclusions removed and read the actual
      7 findings.
- [x] Determine whether `Close(ctx)` is required, or whether the exclusion is a
      documented false positive. It is the latter: `send` enqueues, it does not
      write to the socket, so no caller context is lost.
- [x] Sharpen the `.golangci.yml` comment to say which reason applies where.
- [x] Measure `internal/mockgateway` coverage per file to find the real gap.
- [x] `session_test.go` - session store, auth gate, `isSessionOp` allowlist,
      logout revoking access, end-to-end enforcement through the handler.
- [x] `oauth_test.go` - grant validation, JWT assertion shape, token lifecycle,
      bearer extraction, end-to-end token issuance to a bearer route.
- [x] Mutation-verify all 8 auth mutations, including the one the first pass missed.
- [x] Widen the coverage metric to `./internal/...` so the shipped mock gateway
      counts, and re-baseline the floor to 62% against 65.4%.
- [x] Full verification sweep.
- [ ] Commit, tag, push - **held for explicit approval**.

## Still blocked, unchanged

- [ ] D15 live verification - real account.
- [ ] `submitModelPortfolioOrder` collision confirmed on a real gateway - FA paper
      account. The SDK side is implemented and tested; only the gateway behaviour
      is unverified.
- [ ] `Subscription.Close(ctx)` - **no longer recommended**. The investigation found
      no lost cancellation, so this is closed rather than blocked.
