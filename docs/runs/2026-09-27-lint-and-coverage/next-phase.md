# Next phase

## Worth doing

### `TestWS_DialAndSubscribe` is a wall-clock flake

It asserts "at least one update after 500ms" and misses under parallel load; it
passes 4/4 in isolation. It is not safe to simply widen the window - the window is
the assertion. The right fix is to wait on a condition with a deadline and treat
the deadline as the failure, rather than sleeping a fixed interval and hoping, and
to make sure the mock actually guarantees a delivery rather than leaving it to
timing. Until then this test will keep costing a red build at random.

### `contextcheck` is disabled for the WebSocket layer, not satisfied

The rule is off for `internal/ws.go` and `pkg/ibkr/ws.go` because satisfying it
means adding a `ctx` parameter to the public `Subscription.Close()`. That is a
breaking change and belongs in a minor-version discussion, not a patch. The
questions to settle first: is `Close()` ever called with a context that should
govern it, or is contextless teardown genuinely the right API for a handle that
has to work from `defer`?

### `cmd/ibkr` is only partly testable

`run()`, `parseGlobalFlags()` and `runCompletion()` are parameterised, but
`runAccounts`, `runOrders`, `runPortfolio`, `runStream`, `runPositions` and
`runConfig` still read `os.Args` and write to `os.Stdout` directly, and all of them
construct a live client. Making them testable means threading args and writers
through each, and injecting a client factory so the argument validation can be
tested without a gateway. That is mechanical but touches all six files.

## Settled, no action

- **The 193-vs-191 fixture count is not a gap.** 193 SPEC rows are 192 unique
  operation IDs (`getTradingSchedule` appears under two methods); 191 have fixture
  bodies because `submitModelPortfolioOrder` is deliberately unrouted, pinned by
  `TestModelOrderRouteCollision`.
- **The lint gate is real and green.** Every exclusion in `.golangci.yml` carries a
  comment explaining why, so a future change that flips a linter off is visible in
  review.

## Still open from earlier runs

- **D15** (`listTaxDocumentsAvailable` no longer sends `year=`) has not been
  confirmed against a live gateway. One call with the parameter omitted would close
  it.
- **Mutating model-endpoint dispatch** needs an FA-enabled paper account.
