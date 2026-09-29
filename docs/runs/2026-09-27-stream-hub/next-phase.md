# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** Client half covered; the live
  call is not reachable from here. Read-only endpoint.
- **`submitModelPortfolioOrder` collision.** SDK implemented and tested. An
  FA-enabled paper account would confirm the gateway exposes it at the same path as
  `submitNewOrder`.

## The offline backlog is empty

Not "nearly" - empty, in the sense that matters. Everything left is one of:

- **Mechanical.** `options.go` at 25% is seven one-line functional setters. It would
  move a number and prove nothing.
- **A design change.** `cmd/ibkr` at 33% is the largest gap, and every non-help path
  builds a live client. Covering it properly means a seam for the client factory, not
  a test.
- **Blocked on credentials** - the two above.

The `json.Number` class is closed. The stream hub's observable behaviour is closed.
Continuing to generate runs of the same prompt would be activity, not progress, and
the useful next thing is a decision or a credential rather than another percentage.

## If more offline work is wanted anyway

- **`stream.go`'s remaining gap** - `serveWS`'s own branches and the
  `handleSubscribe` family. Smaller, and the parts left are the WS protocol's
  odder shapes rather than behaviour a user depends on.
- **The `check_money.py` gate, done properly** - it needs to resolve the *type* of
  each field under assertion by following the `toPublic` mappings. That is a small
  analysis, and it would prevent recurrence rather than fix an instance. Only worth
  it if the class recurs.
- **`Subscription.Close(ctx)`** - still a breaking API change and still not
  recommended; the investigation found no lost cancellation.

## Two decisions, each one line

- **The coverage floor** has moved 58 -> 60 -> 62 across three releases while
  coverage went 63.5% -> 66.5%, making it a function of release timing. Pin it, or
  commit to ratcheting per batch.
- **Whether the credit floor should move at all** now that coverage is 4.5 points
  above it. This is the same decision as the last, and it has been deferred for four
  runs.

## Process notes carried forward

- **A test that has never been run against a mutation is a hypothesis.** Four in a
  row in this sequence: money assertions on `float32`-exact values, the proposed
  `check_money.py` gate, `float32Loses` on `0.007`, and now the two hub lock tests.
  Every one passed in normal use.
- **Read both ends of a socket.** `closeAll` looked like it did not deregister
  connections until the test failed; the other half of that behaviour lives in
  `serveWS`'s defer, on the other side of the socket.
- **A skipped test is not a passing one.** The receive-buffer attempt skipped rather
  than failed, which is the failure mode most likely to be mistaken for a pass.
- **A count is not a delivery.** `Broadcast`'s return value was asserted for years of
  use without anyone checking that a frame actually arrived.
