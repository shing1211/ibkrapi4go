# Design Checkers and Residual Gaps - Next Phase

## What this run completed

`check_design` went from verifying 2 of the 9 design documents to **8 of 9**, and
from running only when a maintainer typed `make docs-check` to running on every
push. Eighteen checks were added across `02`, `04`, `05`, `06`, `08` and `09`,
each proven load-bearing by a harness that grew to **232 subtests** with seven
top-level control tests, including two negative controls that fail if a strict
branch is replaced by a naive one. The checks found five false statements in the
design documents — most consequentially, `09-orders-and-confirmation.md`
documented `Reply.Message string` where the code declares `Messages []string`, so
a caller following the document got a compile error, and `02-client.md`
documented two `WithOAuth2JWT*` signatures that do not exist. All five were
corrected in the code's favour. No production code changed.

Along the way the run established that the `lint & security` CI job has been red
throughout, with **742 uncapped findings** (733 in library code) and a
`.golangci.yml` that fails schema validation, so the author's linter settings have
never applied. That was documented and left unchanged by decision.

## Open items

### Design-document claims recorded as disagreements, unenforced by choice

Each is left unenforced by the specific check that found it, so a human can decide
which side is authoritative. Both sides are quoted in the checkers' header
comments.

| Document | Disagreement | Note |
|----------|--------------|------|
| `02-client.md` | The Options block was wrong and has been **fixed**; the *completeness* direction is now unenforced because the list is curated and omits seven exported options | Add the seven, or add a sentence scoping the list, and the reverse direction can be enforced |
| `04-generated-wrapping.md` | The Adapter pattern block is a stale sketch: it names `NetLiq`, `Currency`, `toAccountSummary` and `accountSummaryGenerated`, none of which exist | Marked illustrative by its own `{ ... }`, so not enforced |
| `05-streaming.md` | The components diagram writes `Close()` where the code returns `error` | Only the method's existence is compared |
| `08-concurrency.md` | "Every public method takes `context.Context` first" is false — `Close()`, `GatewayURL()` and 21 `With*` options do not | Not a defect; an option that could not honour a context would be worse. Unenforced |
| `07-money-and-numbers.md` | Unverified entirely, because `check_money.py` already covers its main claim | Its other claim — that the `amounts` helpers do not exist — is checked by nothing |

### Lint and CI hygiene

| Item | State |
|------|-------|
| `lint & security` job | Red. 742 uncapped findings, 733 under `pkg/`, `internal/`, `cmd/`. `scripts/` is now 0 |
| `.golangci.yml` | Fails `golangci-lint config verify`. A v1 file with `version: 2`; `linters-settings` and `issues.exclude-*` are rejected and the whole `issues` block is silently discarded, so the intended revive and test-file exclusions never applied |
| `license-check` | Not a CI step, which is why two shell scripts lacked SPDX headers for so long. Now fixed locally; the gate is still absent |
| `TestWS_DuplicateUpdatedSequence` | Flaked once in three full-suite runs across this series; passed 4/4 in isolation each time. Pre-existing timing sensitivity |
| `go test` link failures on this host | Intermittent `.test.exe` file-lock, environmental |

### Carried forward, still blocked on a human decision

Unchanged from the previous two runs. Each changes what goes on the wire for a
money-adjacent operation.

| ID | Blocked on |
|----|------------|
| D10 | Whether bulk-cancel accepts a `Reason` at all |
| D11 | Which V2 quantity JSON type the gateway accepts. ADR 0008 sides with the bulk path, since the single path routes a `float32` through `strToDecimal` |
| D12 | Whether `AssetTransferRequest.Quantity` on a V2 path is a bug or a deprecation |
| D14, D15 | Whether `float32` money precision is acceptable. Needs a spec change plus `make codegen` |
| — | Whether the mutating model endpoints get their own opt-in build tag, or stay permanently mock-only |

### Deferred by choice

| Item | Why |
|------|-----|
| `check_design` for `07-money-and-numbers.md` | `check_money.py` already enforces its main claim; a second checker would duplicate a live gate |
| `cmd/ibkr` order validation and flag handling | Coverage margin is adequate at 1.0 point. `runOrdersSubmit` reads `os.Args[2:]` directly (`cmd/ibkr/orders.go:82`), so the parse loop wants extracting first |
| Coverage ratchet | The floor is 58% against a measured 59.0%, and this run added no production statements, so there was nothing to ratchet to |

## Candidate next phases

### P1 — Make the lint gate mean something
**Objective:** Decide the lint story. Either migrate `.golangci.yml` to v2 so the
author's settings actually apply, or work the 733 library findings down by linter.
**Why now:** A security gate that is red on `main` and whose configuration is
silently discarded provides no assurance, and its redness trains contributors to
ignore it.
**Effort:** S for the migration decision, L for the findings.
**Dependencies:** None technically. The migration is a judgement call because it
activates exclusions that drop roughly 95 findings from the report.
**Risks:** Migrating silently weakens the gate. Fixing findings is a large project
that needs its own slicing.

### P2 — Resolve the banking wire contracts
**Objective:** D10, D11, D12 — make single and bulk V2 asset-transfer payloads
consistent, carry `Reason` through bulk cancel, settle what
`AssetTransferRequest.Quantity` means on a V2 path.
**Why now:** The characterisation tests exist, so each fix is provable. All three
have been open since v1.1.5.
**Effort:** S, given the tests.
**Dependencies:** **The human answers.** Not startable blind.
**Risks:** Picking wrong means a rejected bulk cancel or a rejected transfer.

### P3 — Close the remaining design-doc gaps
**Objective:** Add the seven missing `With*` options to `02-client.md` or scope the
list; give `05-streaming.md`'s diagram a result type; refresh `04`'s stale Adapter
sketch; add a check for `07`'s "helpers do not exist" claim.
**Why now:** Each is recorded as a specific unenforced disagreement, so closing
them converts documented gaps into enforced ones.
**Effort:** S.
**Dependencies:** None.
**Risks:** Adding a sentence to scope a list is weaker than adding the facts.

### P4 — Add `license-check` to CI
**Objective:** Make the SPDX rule enforced, since it demonstrably was not.
**Effort:** S.
**Why now:** Two files went without headers long enough to be invisible. The same
will happen again.
**Risks:** None; `addlicense -check` already passes.

## Recommended next phase

**P4, then P1's migration decision, then P3.**

P4 is a one-line CI step against a rule that is already satisfied, so it should
just be done. P1's *decision* is cheap and unblocks a badly broken gate, even if
the findings work is deferred. P3 closes gaps this run documented but could not
enforce, and is a natural continuation of the same theme.

P2 stays blocked until the human answers, and its characterisation tests mean it
can be picked up instantly once they do.

## Open questions for the human

1. **For D10 and D11, which is authoritative — the published spec or the live
   gateway?** If the answer needs an FA account, these should stay open rather
   than be decided from the spec alone.
2. **Is `float32` money precision acceptable?** ADR 0008 is satisfied in shape but
   not in substance: `float32ToStr(16777217)` is `"16777216"`.
3. **Is `AssetTransferRequest.Quantity` on a V2 path a bug or a deprecation?**
4. **Should `.golangci.yml` be migrated to v2?** It would activate exclusions the
   author intended, which drops roughly 95 findings from the report. That is a
   security-gate decision, not a hygiene one.
5. **Should the coverage floor keep ratcheting, and at what cadence?** It has
   moved 35% -> 48% -> 58% across five releases.
6. **Should the mutating model endpoints get their own opt-in build tag, or stay
   permanently mock-only?** The mock cannot route `SubmitModelPortfolioOrder`
   separately because the payloads are byte-identical.

## Process note for whoever picks this up

Sub-agents corrected the orchestrator's briefs **eleven times** across these three
runs, and every correction was right: claims about code that did not exist, a
caller count off by four, a field declaring one key where the brief said two, a
hazard class under-specified by roughly 100 sites, and one orchestrator figure
that was a tooling artifact rather than a fact. Assume the brief's incidental
claims need the same independent check as its main deliverable.

Two agents reported that their briefs were self-contradictory and chose to hand
over red tests rather than green ones that could not fail. Both times that refusal
is what surfaced the real problem instead of baselining it away. **When a brief
and the code disagree, the red test is the deliverable.**
