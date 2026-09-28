# Plan: audit the remaining blocked list, and close the gap that caused a wrong claim

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `ead8dab` (`v1.1.14`)

## Objective

The instruction was to "plan and implement remaining blocking items". The first
job was therefore to find out what was actually left, rather than to work from the
notes the previous runs left behind.

## What the audit found

The carried-forward blocked list was three items, and checking each against the
code rather than the notes changed the picture:

- **D15 live verification** - genuinely blocked. Needs a real account.
- **`submitModelPortfolioOrder` dispatch** - the notes described this as an
  operation that "stays unrouted" with 191 of 192 operations covered. Checking the
  code first: `pkg/ibkr/models.go:761` implements it,
  `TestModels_SubmitModelPortfolioOrder` covers it, and `docs/ROADMAP.md` states
  all 193 operations are implemented. The unrouted thing is the *mock gateway
  route*, not the SDK operation. A dedicated test,
  `TestModelOrderRouteCollision` (`coverage_test.go:185`), already documents it.
- **WebSocket `contextcheck`** - genuinely blocked, and not on this: satisfying it
  means a `ctx` parameter on the public `Subscription.Close()`, which is a breaking
  change belonging in a minor version. That is a decision for the maintainer, not
  a task.

The previous run's next-phase also proposed adding a test asserting the two
colliding paths stay unrouted together. That test already exists.

So: nothing on the list was actionable offline, and one of the proposals was
already done.

## The real finding

The notes were wrong because of a genuine hole in the suite. Coverage is checked
by matching a SPEC row against the route table on **normalized method and path**.
An operation that shares a path with another one satisfies that check through the
other operation's route, so it is unroutable and nothing says so.

That is exactly how a wrong sentence got written and published in 1.1.14, and how
it then misled me again ten minutes later while auditing the same list.

## What this run does

1. Correct the published 1.1.14 changelog entry, inline, with the real figures.
2. Add `TestEveryOpIDIsRoutedOrExplained` - an opId-aware check where every
   distinct SPEC operation must be routed or carry a verified, reasoned exception.
3. Add `sharedOpIDs` for the reverse case: one opId serving several routes, which
   would otherwise share a fixture silently.
4. Mutation-verify all six failure branches.

## Not attempted, and why

No SPEC-to-SDK method mapping. A complete gate would need a 192-entry table
duplicating `docs/SPEC.md`, which would rot independently of the spec it mirrors.
Each manager method is already pinned by its own test, so a removed method fails
its test; what was genuinely unenforced is the *router* claim, and that is what
this run gates.

## Constraints held

- No credentials used, and none required.
- `client/*.gen.go`, the spec, and `scripts/patch_spec.py` untouched.
- No public API change.
