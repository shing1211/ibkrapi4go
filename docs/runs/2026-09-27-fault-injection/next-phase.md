# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** The client half is already
  covered: `rest_reports_e2e_test.go:296` asserts the parameter is absent from the
  wire entirely, not merely empty. What remains is the live call. Read-only, so the
  risk is low and the obstacle is purely that it cannot be done offline.

- **`submitModelPortfolioOrder` collision.** The SDK implements and tests the
  operation. An FA-enabled paper account would confirm the gateway exposes it at the
  same path as `submitNewOrder`, which is the assumption the mock's unrouted
  decision rests on. `TestEveryOpIDIsRoutedOrExplained` enforces that decision, so
  changing it requires deleting the exception deliberately.

## Available next, no credentials needed

- **`internal/mockgateway` is at 86.6%**, and the remainder is now small and
  identified: `stream.go` 73.1% (`closeAll`, `Broadcast`, the `handleSubscribe`
  family, and `serveWS`'s own branches) and `server.go` 79.7% (`Operations`,
  `waitCtx`/`sleepCtx` edge cases, the remaining `ServeHTTP` paths).
  `stream.go`'s `closeAll` and `Broadcast` are the interesting ones: they govern what
  a client still receives after its own subscription is cancelled.

- **`options.go` at 25%** - the seven functional options. Each is a one-line setter,
  so coverage here is nearly free, and it is the cheapest remaining gap in the
  module. Worth doing for completeness, not for insight.

- **`cmd/ibkr` at 33%** is the largest untouched block, but every non-help path
  builds a live client. Testing it properly means standing up a gateway per
  subcommand, which is a design change rather than a test. Only worth it if the CLI
  is a supported interface.

- **Coverage floor policy, now a real decision.** 66.3% against a 62% floor. The
  floor has moved 58 -> 60 -> 62 in three releases, which makes it a function of
  release timing rather than of the code. Either pin it and let coverage rise, or
  commit to ratcheting on every batch - but the current churn is not a policy.

## Process notes carried forward

- **A test that asserts a precedence that does not exist is worse than no test.**
  `TestServer_SessionExpiredFaultTakesPrecedence` claimed the session-expired switch
  outranks the unauthenticated response; it does not, and correcting the test
  revealed the switch is independent of `authRequired` - a real property the
  original name had obscured.
- **Check which path the dependents actually use before testing a whole API.**
  The SDK injects via `SetPolicy`, not `Set`/`SetGlobal`. Testing the declarative
  API alone would have left the path every retry test depends on untested, and
  would have looked thorough.
- **The headline percentage has now under-reported three findings in a row.** The
  value of a run is the behaviour pinned, not the number moved; judge this work by
  what is now caught, not by the delta.
