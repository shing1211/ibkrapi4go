# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** The spec marks `year` optional
  and the wrapper omits it when unset. Settled in code; unverified against a real
  gateway. Read-only endpoint, so the risk is low and the only obstacle is that it
  cannot be done offline.

- **The `submitModelPortfolioOrder` path collision.** The SDK implements and tests
  the operation. What is unverified is the gateway side: that a real
  FA-enabled paper account also exposes it at the same path as `submitNewOrder`.
  Until that is confirmed, the mock gateway must keep the model operation
  unrouted, since a route that could never be selected would serve the wrong
  fixture. `TestEveryOpIDIsRoutedOrExplained` now enforces that decision, so
  routing it requires deleting the exception deliberately rather than by accident.

## Blocked on a maintainer decision

- **WebSocket `contextcheck`.** `Subscription.Close()` takes no context by design;
  it is public and commonly called from a `defer`. Adding a `ctx` parameter is a
  breaking change and belongs in a minor version, with an ADR. Not a lint tweak.

## Available next, no credentials needed

- **Coverage headroom.** 63.5% against a 60% floor. The next lever is
  `internal/mockgateway/` exercised directly rather than only through `pkg/ibkr`
  tests - it is the substrate everything else depends on.

- **The remaining `if err != nil` branches** in `trading_accounts.go` and
  `notifications.go` are the only gap in those two files. Previously judged to
  assert the language rather than behaviour. If that judgement is revisited, the
  useful version is a table test over the error path, not one assertion per branch.

- **A SPEC-to-SDK method map.** Deliberately declined this run: a 192-entry table
  duplicating `docs/SPEC.md` would drift independently of the spec. If a stronger
  link is ever wanted, generate it from the spec rather than maintaining it by
  hand.

## Process notes carried forward

- **Verify a carried-forward claim against the code before acting on it.** Three of
  the four items on the "remaining blocked" list were described in a way that did
  not survive checking, and one proposed task was already complete. A next-phase
  note is a hypothesis about the code, not a fact about it.
- **An ambiguous number in a published document will be misread, including by the
  author.** When a figure can be read two ways, the fix is a test that reports it,
  not a clearer sentence.
- **Counting beats reading.** The 192/191/1 figures came from parsing the spec and
  the route table directly. The hand-maintained version of the same claim was
  wrong in a way that survived review and a release.
