# Payload Contracts and Design Checks - Plan

## Objective

Execute the recommended phase from `2026-09-26-test-hardening/next-phase.md`:
P1 + P6 + P7 + P3. Characterise the deferred `rest_banking.go` wire payloads,
delete the three dead helpers, correct the inert mock fixture, drop the wasted
token fetch, and widen `check_design` from 2 of 9 design documents to more.

## What is in scope, and what is not

**In scope** — all of it is either test-only or a change whose correct outcome is
already determined:

| Task | Work | Risk |
|---|---|---|
| N1 | Byte-level characterisation tests for four banking payloads | None — pure tests, passing on current code |
| N5 | Delete three dead package-private helpers | Negligible — unexported, no production caller |
| N6 | Correct the shared `OpGetRequestsStatus` fixture | Low — shared fixture, consumers checked first |
| N7 | Drop the wasted token fetch in `TradeConfirmations.ListAvailable` | Low — behaviour preserved, verified by header assertion |
| N8 | Add a `check_design` check for `02-client.md` | Low — new check, one document |
| N10 | Cover `cmd/ibkr` order validation and flag handling | Low — test-only |
| N11 | Re-measure and ratchet the floor | None |
| N12 | Close-out | None |

**Deliberately NOT in scope**, because the plan itself gates these on a human
decision or a real gateway:

- **D10** — `CancelInstructionsBulk` dropping `Reason`. Open question 4 asks
  whether bulk-cancel even accepts a reason. Changing the payload without that
  answer risks a rejected bulk cancel on a money-moving operation.
- **D11** — the V2 quantity JSON-type divergence (number vs string). Open
  question 4 again: both cannot be right, and the spec alone may not settle it.
- **D12** — `AssetTransferRequest.Quantity` ignored on the V2 paths. Open
  question 5: only someone who knows the callers can say whether it is a bug or
  a deprecation.
- **D14/D15** — `float32` money precision and the forced `year=`. These need a
  spec change plus `make codegen`, and a regeneration diff should not be
  reviewed alongside twelve other files.

N1 is the one piece of P1 that *is* safe: pinning the current payloads with
byte-level tests makes whichever of D10/D11/D12 the human later approves provable
rather than speculative, and it is useful even if all three stay open.

## Approach

Sequential, one sub-agent per task, orchestrator verification between. The
tasks share a working tree and several touch `pkg/ibkr/rest_banking.go`, so
parallel agents would conflict and would make a failure hard to attribute.

Per the process lesson from the previous run: verify every incidental finding a
sub-agent reports, individually. Two of three incidental claims in one report
last run were false.

## Risks

- **N6 touches a shared fixture.** Other suites may depend on the current body.
  Consumers must be enumerated before the change, and the `oneOf` arm without a
  timestamp must stay covered by an explicit override.
- **N8 may surface pre-existing drift** in `02-client.md`, which then has to be
  fixed in the same change. One document per commit keeps it attributable.
- **N11**: the floor must never be set above the measured value, and the
  measurement must come from a deduplicated profile that reproduces
  `go tool cover -func`.

## Non-Goals

- No edits to `client/*.gen.go`. No regeneration.
- No new dependencies.
- No speculative wire-format changes.
- No removal or renaming of any exported field.

## Actuals vs. Plan

| Planned | Outcome |
|---------|---------|
| N1 characterise the four banking payloads | Done. 383 insertions, 0 deletions. Also **corrected the plan twice**: the plan said exact key-set assertions already existed for `CancelInstructionsBulk` (they did not) and that the file was untracked (it was committed) |
| N5 delete the three dead helpers | Done. An orphaned test helper, `sortedKeys`, was removed too — leaving it would have planted new dead code in a dead-code-removal task |
| N6 correct the inert requests-status fixture | Done, and it surfaced a second inert fixture the plan had not found: `createSsoSessions` |
| N7 drop the wasted token fetch | Done. One part of the task proved **not deliverable as specified** — a count-based token test cannot fail pre-fix, because the redundant call hits the same cache and issues no separate fetch. No such test was written |
| N8/N9 `check_design` for the remaining design docs | **Deferred.** The shape-check work expanded to fill the run |
| N10 cover `cmd/ibkr` | **Deferred by choice.** Coverage margin is adequate at 0.9 points |
| N11 re-measure and ratchet | Done. 58.9% against the existing 58% floor; the floor was not moved, because 58.9% leaves only a 0.9-point buffer and no new statements were added to justify more |
| N12 close-out | Done |

### Where the plan was wrong

The plan's own premise about scope was wrong in an instructive way. It listed
two bypass classes as one — 17 `json.Unmarshal(resp.Body, ...)` sites — when
there are in fact roughly 100 further sites reached through the non-`WithResponse`
generated methods. That under-specification is what produced 88 false positives
in the first version of the strengthened shape check.

A sixth task appeared that the plan did not anticipate: the fixture shape check
itself (`internal/mockgateway/shape_test.go`) had `DisallowUnknownFields` set
against `var js any`, so it could never fire. It existed to catch exactly the
class of defect that two of the plan's own tasks turned out to be instances of.
