# Next phase

## Worth doing

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

- **`TestWS_DialAndSubscribe` is no longer a flake** (fixed in v1.1.10). It slept a
  fixed 500ms and then asserted an update had arrived, but the mock gateway emits
  its scripted ticks synchronously while handling the subscribe frame, so the
  update was already on the wire before `Subscribe` returned. The sleep was not
  waiting for anything and the assertion could only fail on a slow machine.
  `fakeSink` now signals deliveries on a non-blocking channel and the test waits
  for the event with the context deadline as its bound. It runs in about 10ms
  instead of 500ms and additionally asserts the delivered `ConID`, which the old
  version never checked. Verified over 15 consecutive runs and 4 runs under
  `-race` with `GOMAXPROCS=2`.
- **The other `time.Sleep` sites in the suite are not the same defect**, and were
  left alone. `oauth_test.go` waits to prove a call did *not* return early, which
  cannot be expressed as a wait on an event; the circuit-breaker tests wait for a
  real cooldown; the fault-injection tests sleep to simulate latency.
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
