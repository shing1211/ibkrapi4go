# Next phase

State as of the end of this run. Nothing below is actionable without credentials
or an explicit decision.

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** The spec marks `year` optional
  and the wrapper omits it when unset. That is settled in the code and the spec;
  what is missing is a live call proving the gateway accepts the request without
  it. A real account is required, and this is a read-only endpoint, so the risk is
  low - it simply cannot be done offline.

- **Mutating model-endpoint dispatch.** `submitModelPortfolioOrder` is
  deliberately left unrouted. Its path collides with `submitNewOrder`, so routing
  it would silently send portfolio orders to the wrong endpoint - a live-money
  hazard, not a tidy-up. The fixture set is 193 rows / 192 unique opIds
  (`getTradingSchedule` appears twice); 191 are routed. An FA-enabled paper account
  is needed to confirm the collision rather than infer it from the spec.

## Blocked on an API decision

- **WebSocket `contextcheck`.** `Subscription.Close()` takes no context by
  design: it is a public method on an exported type, commonly called from a
  `defer`, and threading a context through is a breaking change. The linter stays
  excluded for `pkg/ibkr/ws.go` and `internal/ws.go`. If that exclusion is ever
  revisited it is an ADR and a major version, not a lint tweak.

## Available next, no credentials needed

- **Coverage headroom.** The floor is 60% against 63.5% actual. The obvious next
  target is `internal/`, which is exercised mainly through the mock gateway rather
  than directly. Ratchet the floor when the next batch lands, not before.

- **Mock gateway coverage as a contract.** `internal/mockgateway/` is the substrate
  for `pkg/ibkr`'s tests. Direct tests of its order-cancellation and
  subscription-confirmation paths would harden the harness the rest of the suite
  depends on, which is a higher-leverage target than more `pkg/ibkr` unit tests.

- **The unrouted `submitModelPortfolioOrder` deserves a static guard.** Even
  before an account exists, a test could assert that the two colliding paths stay
  unrouted together, so a future edit cannot route one without the other. That is
  checkable offline and would convert an account-blocked hazard into a locally
  enforced invariant.

## Process notes carried forward

- Both mutations in this run were caught by the mutation harness, not by reading
  the diff. One of them - the `range` rewrite that silently dropped flags - looked
  like a clean improvement on review.
- The two lint profiles can disagree about whether a suppression is needed. Prefer
  a structural fix when one exists; a `//nolint` that satisfies one profile and
  breaks the other costs more than the finding did.
- `go test -coverprofile` needs its flags quoted in this PowerShell environment;
  unquoted `-coverpkg=./pkg/ibkr,./internal,./cmd/...` produced a spurious
  `FAIL .out [setup failed]` and an empty total.
