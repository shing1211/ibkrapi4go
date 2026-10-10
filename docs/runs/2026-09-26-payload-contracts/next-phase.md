# Payload Contracts and Design Checks - Next Phase

## What this run completed

It started as the recommended phase from `2026-09-26-test-hardening` and grew a
sixth item when a check that had been inert for the life of the mock gateway was
made real and immediately found 88 fixtures it should have been rejecting. The
`rest_banking.go` wire-contract fixes were deliberately **not** attempted: the
prior run gated them on questions only a human or a real gateway can answer, so
N1 pinned the current payloads byte-for-byte instead, giving every one of those
decisions a before and an after. Two inert fixtures were corrected, three dead
helpers and an orphaned test helper deleted, a wasted token fetch dropped, and
the fixture shape check went from provably inert (`DisallowUnknownFields` set
against `var js any`) to strict, with 191 fixtures at 180 passed / 4 skipped /
**0 failed** — and stricter than the naive version that preceded it, key-checking
59 operations against a single response type rather than 50. Coverage 58.6% ->
58.9% against a 58% floor.

## Open items

### Blocked on a human decision — unchanged, still not started

| ID | Defect | Location | Question |
|----|--------|----------|----------|
| D10 | `CancelInstructionsBulk` drops `Reason`; `reason` ships as `""` for every element while the single path sets it | `pkg/ibkr/rest_banking.go:390-394` | Does bulk-cancel accept a reason at all? |
| D11 | V2 quantity is a JSON **number** on the single path, a **string** on bulk | `:750` vs `:802`, `:820` | Which does the gateway accept? ADR 0008 sides with the bulk path — the single path routes a `float32` through `strToDecimal`, whose own comment concedes "Precision loss is accepted for now" |
| D12 | `AssetTransferRequest.Quantity` ignored on both V2 paths | `:748-752`, `:816-821` | Bug, or a V1-only field? |
| D14 | Tax-voucher money arrives as `float32`, so ADR 0008 holds in shape but not in substance | `client.gen.go:16344-16353` via `rest.go:769-774` | Is the rounding a real risk at IBKR's dividend magnitudes? Needs a spec change plus `make codegen` |
| D15 | `ListTaxDocumentsAvailable` always sends `year=` | `rest.go:281-283` | Same shape as D14 |

N1 means each of these can now be fixed with a provable before/after.

### Found this run, not fixed

| Defect | Location | Note |
|--------|----------|------|
| Fixture shape check cannot reach 4 operations | `decode_path_test.go`, `fixtures.go` | `fetchDividends_1`, `createExternalAssetTransfers_2`, `bulkExternalAssetTransfers_2`, `getCurrentState_1`: oapi-codegen renames the method, the fixture layer files it under the old name. A codegen divergence, not a fixture bug |
| `forecastMarketResponse.payout` is `number/double` in the spec, generates as `*string` | `specs/ibkr_spec.json` vs `client.gen.go:18569` | Same family as the deferred `quantity` item. The fixture omits the field rather than guess a JSON type |
| `orderCancelSuccess.order_id` is `integer/int64`, sibling `orderSubmitSuccess.order_id` is `string` | `client.gen.go:19992` vs `:20247` | Upstream spec inconsistency. Fixtures now match their generated types; the spec disagreement remains |
| `pkg/ibkr/forecast.go` declares a manager with no methods | `pkg/ibkr/forecast.go`, 10 lines | Five forecast operations are unreachable from the SDK, so their fixtures cannot be exercised end to end |
| `shape_test.go` call resolution is by method name, not by type | `decode_path_test.go` | `go/ast` without `go/types` over-approximates where same-named methods exist. Safe here — a decode site resolving to 0 or >1 ops never exempts anything, and a guard test fails the build on the >1 case — but a receiver-aware resolver would need `go/types` |
| 12 operations have no establishable decode path and stay key-checked | `decode_path_test.go:701` | Reported, not exempt. Uncertainty should not produce a pass |

### Deferred by choice

| Item | Why |
|------|-----|
| N8/N9 — `check_design` for `02-client.md` and `04`-`09` | The shape-check work expanded to fill the run. Still the cheapest structural win available |
| N10 — `cmd/ibkr` order validation and flag handling | Coverage margin is adequate at 0.9 points. `runOrdersSubmit` reads `os.Args` directly (`cmd/ibkr/orders.go:82`), so the parse loop wants extracting first |
| N5 follow-up — `payout` and `order_id` fixture divergence | Belongs with D11/D14, which are blocked |

## Candidate next phases

### P1 — Resolve the banking wire contracts (D10, D11, D12)
**Objective:** Make single and bulk V2 asset-transfer paths send the same payload
shape; carry `Reason` through bulk cancel; decide what
`AssetTransferRequest.Quantity` means on a V2 path.
**Why now:** N1 has made all three provable, and each is money-adjacent.
**Effort:** S, given the characterisation tests exist.
**Dependencies:** Answers to open questions 1 and 2 below. **Not startable without
them** — picking wrong means a rejected bulk cancel or a rejected transfer.
**Risks:** Changing what goes on the wire speculatively.

### P2 — Resolve the `float32` money precision (D14, D15, `payout`)
**Objective:** Get money onto string types end to end so ADR 0008 holds in
substance, and stop sending a bare `year=`.
**Why now:** The only item touching both an ADR's intent and the codegen
pipeline. Easy to leave open because the public API *looks* compliant.
**Effort:** M.
**Dependencies:** A spec change in `scripts/patch_spec.py`, then `make codegen`.
`make codegen-verify` must pass before merge.
**Risks:** Regeneration is a large diff and should be reviewed alone.

### P3 — Extend `check_design` to the seven unverified design documents
**Objective:** A check per design document, one document per commit, so drift is
a build failure rather than a stale paragraph.
**Why now:** This run's biggest win was a checker that existed and did nothing.
Two more of the same class may be waiting. A checker over
`07-money-and-numbers.md` would have surfaced D14 from a document that already
states the rule.
**Effort:** S.
**Dependencies:** None.
**Risks:** Will surface pre-existing drift that must be fixed in the same commit.

### P4 — Cover `cmd/ibkr`'s validation and parsing helpers
**Objective:** Get the pure helpers off 0% without pretending `main()` is
testable. Extract the `runOrdersSubmit` parse loop first so tests need not mutate
`os.Args`.
**Why now:** Last identified block of meaningful uncovered statements; the floor
cannot rise much further while 318 `cmd/*` statements sit in the denominator.
**Effort:** S.
**Dependencies:** The `os.Args` extraction.
**Risks:** A test that only juggles globals is brittle.

## Recommended next phase

**P1 + P2, in that order, and P1 only once its two questions are answered.**

P1 is the highest-value work available and is now cheap, but it must not be
started blind. P2 is the largest outstanding correctness item and deserves a
review window to itself. P3 and P4 are worth doing and neither is urgent.

## Open questions for the human

1. **For D10 and D11, which is authoritative — the published spec or the live
   gateway?** `CancelInstructionsBulk` may not accept a reason, and the two V2
   quantity JSON types cannot both be right. If the answer needs an FA account,
   these should stay open rather than be decided from the spec alone.
2. **Is `float32` money precision acceptable?** ADR 0008 is satisfied in shape
   but not in substance: `float32ToStr(16777217)` is `"16777216"`. Is the
   rounding a real risk at IBKR's dividend magnitudes, or is documenting it
   enough?
3. **Is `AssetTransferRequest.Quantity` on a V2 path a bug or a deprecation?**
   Only someone who knows the callers can say whether real code passes both it
   and `Positions`.
4. **Should the coverage floor keep ratcheting, and at what cadence?** It has
   moved 35% -> 48% -> 58% in four releases. Diminishing returns are arriving.
   Options: every release, every N releases, only when a slice lands, or trim
   `-coverpkg` to `./pkg/ibkr,./internal` and track `cmd/*` separately.
5. **Should the mutating model endpoints get their own opt-in build tag, or stay
   permanently mock-only?** The mock cannot route `SubmitModelPortfolioOrder`
   separately because the payloads are byte-identical, so mock coverage of that
   path is structurally impossible.
6. **Should `check_design` enforce the design documents, or are they prose the
   ADRs and the code own?** Extending it from 2 of 9 to 9 of 9 makes design drift
   a build failure. That is a policy choice about which artifact is canonical,
   worth making deliberately.

## Process note for whoever picks this up

Sub-agents corrected the orchestrator's briefs **six times** this run, and every
correction was right — including two where the brief asserted something that did
not exist in the code, one where a caller count was off by 4, and one where a
hazard class was under-specified by roughly 100 sites. Assume the brief's
incidental claims need the same independent check as its main deliverable.

One agent also received a self-contradictory brief (do not change fixtures, do
not suppress failures, leave the suite green — with 88 pre-existing defects in
the way) and chose to hand over a red test rather than a green one that could
not fail. That refusal is what surfaced the 76 false positives instead of
baselining them away. When a brief and the code disagree, the red test is the
deliverable.
