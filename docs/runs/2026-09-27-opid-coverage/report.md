# Report: opId-aware coverage, and a corrected claim

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `ead8dab` (`v1.1.14`)
- **Outcome**: complete, uncommitted pending approval

## The error this run starts from

The 1.1.14 changelog said:

> 191 of 192 unique fixture operations are routed; the duplicate
> `getTradingSchedule` row is a fixture artifact, not a gap.

The arithmetic is defensible. The framing is not, and it caused a real mistake:
auditing the blocked list afterwards, I read that sentence as "an operation is
unimplemented" and nearly re-reported it as an open engineering gap.

The sentence is ambiguous because of a genuine hole in the suite. Coverage is
checked by matching a SPEC row against the route table on **normalized method and
path**. `submitModelPortfolioOrder` and `submitNewOrder` are both
`POST /v1/api/iserver/account/{segment}/orders`, so once placeholders normalize
they are indistinguishable, and the model operation satisfies the coverage check
through the *other* operation's route. Nothing tested that it is unroutable.

A published document cannot be fixed by remembering harder. The gap gets closed.

## What the code actually says

| Claim | Status |
|---|---|
| `SubmitModelPortfolioOrder` implemented in the SDK | Yes - `pkg/ibkr/models.go:761` |
| Covered by a test | Yes - `TestModels_SubmitModelPortfolioOrder` |
| All 193 SPEC operations implemented | Yes - `docs/ROADMAP.md` |
| Has a route in the mock gateway | No - deliberately, path collides |
| Documented by a dedicated test | Already - `TestModelOrderRouteCollision` |
| A static guard for the collision | Already existed at `coverage_test.go:185` |

The previous run's next-phase proposed adding that static guard. It was already
there, which is worth recording: the notes were describing work that had been
done.

## Measured, not assumed

Parsing `docs/SPEC.md` and the route table's opId constants directly:

- 193 SPEC rows
- **192 distinct opIds** - `getTradingSchedule` is used for two rows
- 192 route op constants declared
- **0 opIds with no declaration**
- 192 method+path keys, of which **exactly one** is shared by two operations:
  `submitModelPortfolioOrder` vs `submitNewOrder`

So the accurate statement is: every distinct operation is declared; 191 are
routable; one is not, and is explained.

## The gate

`TestEveryOpIDIsRoutedOrExplained` requires each distinct SPEC operation to be
either registered, or listed in `pathCollisionOps` naming its shadower. The entry
is verified on every load:

- the operation is actually routed -> a stale excuse, fail
- the shadower has no route either -> an entry hiding a missing route, fail
- the two do not share a method and normalized path -> a false reason, fail

`sharedOpIDs` handles the inverse hazard: one opId covering several routes, which
would have them share a single fixture. New declarations must be added, and stale
ones removed - checked in both directions.

The test logs the real figures, so the numbers are read from a run rather than
inferred from a comment:

```
opId coverage: 192 distinct operations, 191 routed, 1 explained by a path
collision: submitModelPortfolioOrder
```

## Mutation verification

Six mutations, all caught, each naming the defect:

| Mutation | Message |
|---|---|
| exception deleted | `1 operation(s) have neither a route nor a pathCollisionOps entry` |
| exception for a routed op | `... but GET /orders and POST /orders do not [collide]` |
| exception whose shadow is unrouted | `... but "submitModelPortfolioOrder" has no route either` |
| exception naming non-colliding ops | `... but POST /{modelCode}/orders and GET /orders do not` |
| stale `sharedOpIDs` allowance | `sharedOpIDs says "getOpenOrders" covers 3 routes, but SPEC gives it 1` |
| `sharedOpIDs` for a non-existent op | `... which is not an operation in docs/SPEC.md` |

## Documentation

The 1.1.14 entry is corrected **inline**, not silently. A reader who lands on that
entry - which is likely, since it is where the claim is - sees the correction next
to it. A 1.1.15 entry states the correction and the new gate.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `golangci-lint run --tests=false`,
`check_money.py`, `check_design`, `check_links.py`, `check_i18n.py`,
`check_spec_version.py`.

`codegen-verify` not re-run: the spec, `scripts/patch_spec.py`, and
`client/*.gen.go` were not touched.

## Still blocked, now with evidence rather than notes

- **D15** - `listTaxDocumentsAvailable` with `year` omitted, against a real gateway.
- **`submitModelPortfolioOrder`** - the SDK side is implemented and tested; what
  needs an FA-enabled paper account is confirming that a real gateway also treats
  the two operations as one path.
- **WebSocket `contextcheck`** - needs `Subscription.Close(ctx)`, a breaking API
  change and a maintainer decision.
