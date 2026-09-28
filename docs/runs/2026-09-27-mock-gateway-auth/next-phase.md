# Next phase

## Closed this run

- **WebSocket `contextcheck`.** No longer on the blocked list. The 7 findings were
  read rather than assumed: the gauge sites use the connection context
  deliberately, and `Close` loses no cancellation because `WSConn.send` enqueues
  rather than writing to the socket. `Subscription.Close(ctx)` would be a public API
  change that buys nothing. The exclusion stays, with the verified reasoning recorded
  in `.golangci.yml`.

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** Settled in code and spec;
  unverified against a real gateway. Read-only endpoint.
- **`submitModelPortfolioOrder` collision.** The SDK implements and tests the
  operation; what an FA-enabled paper account would confirm is that the real
  gateway also exposes it at the same path as `submitNewOrder`. Until then the mock
  gateway must keep it unrouted, and `TestEveryOpIDIsRoutedOrExplained` enforces
  that, so routing it requires deleting the exception on purpose.

## Available next, no credentials needed

- **`internal/mockgateway` is now 75.5%, and the remaining third is findable.**
  `stream.go` is 66.0% (51 uncovered blocks) and `server.go` is 62.2% (45 blocks).
  The stream path is the better target: it is where framing, subscription teardown
  and field selection live, and the SDK's streaming tests depend on it the same way
  the auth tests depended on the auth store.

- **`scenario.go` at 30.2%** - fault injection. The fault paths are what make the
  retry and error-classification tests meaningful, so an untested scenario builder
  quietly weakens the SDK's retry tests. Same argument as the auth store.

- **`recorder.go` at 40.6%** - the request recorder is how tests assert *what was
  sent*. A recorder bug could make a request assertion pass vacuously.

- **`cmd/ibkr` is at 33.0%.** Now testable but thin. `run` builds a live client for
  every non-help path, so the remaining gap is mostly the gateway-facing halves.
  Worth doing only if the CLI is a supported interface rather than a demo.

- **Coverage floor policy.** 65.4% against 62%. Keep ~3 points of headroom and
  ratchet only when a batch lands, or pin the floor and let coverage rise - the
  previous two runs raised it twice in three releases, which makes the number a
  function of release timing rather than of the code.

## Process notes carried forward

- **Read the linter's actual complaints before accepting a note about them.** The
  WebSocket item had carried "breaking change, belongs in a minor version" through
  three runs without anyone running the linter. It was never a change at all.
- **Coverage can be concentrated on the wrong half of a component.** The mock's route
  tables were at 100% and its auth decisions at 4%. A number that is high on
  aggregate can still say nothing about the part that matters.
- **Two "the number is wrong" defects in consecutive runs, both found by measuring.**
  The opId coverage gap and the `coverpkg` omission were each invisible to review
  and obvious to a script. Measure the number before publishing it.
- **A mutation that survives the first pass is telling you the test is
  unobservable, not that the code is fine.** `clear()` keeping the default token
  looked harmless because it is behaviourally identical in that case; the fix was a
  test that sets a non-default token, not a suppression.
