# Next Phase: Post-Audit Directions

The audit remediation is complete. Every gap it found is closed or explicitly
documented as a limitation. This document records what is deliberately still
open, so the next run starts from facts rather than from the impression that
everything is finished.

## Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| Coverage margin is thin | Medium | 37.5% against a 35% floor. Any new uncovered code breaks the gate, which is either a useful ratchet or an annoying one depending on intent. Decide whether to raise the floor with a test-writing phase or lower it. |
| goleak asserted in 4 of 42 test files | Medium | `docs/TESTING.md` now says only "tests that start background goroutines", which is true but leaves most of the suite unchecked. Either extend the assertion or state the gap in the doc. |
| Coverage gate is replacement-blind | Low | It checks order and presence. Deleting one middleware and adding another keeps the count, so the check still passes. A stronger check would compare the exact set. |
| Model endpoints unverified against a real gateway | Low | The mock cannot route `SubmitModelPortfolioOrder` separately, and the integration suite stays read-only. Verifying dispatch needs an FA-enabled paper account and an opt-in build tag. |
| `TestWS_Resilience` timing sensitivity | Low | Flaked once under full-suite parallel load on Windows, then passed repeatedly. Likely a real timing sensitivity worth tightening if it recurs. |

## Candidate Next Phases

### P1 — Raise the coverage floor deliberately
**Objective:** Write tests for the largest uncovered public surfaces and raise
the gate from 35% in steps.
**Why now:** The floor was set at the current value purely to make an existing
gate honest. It is not a quality target.
**Effort:** M
**Dependencies:** None.
**Risks:** Effort is open-ended; pick surfaces by uncovered-statement count
rather than by percentage alone.

### P2 — Tighten the design checker further
**Objective:** Make `check_design` compare the exact middleware set, not just
order and presence, and extend it to the other design documents.
**Why now:** It now catches real drift, but a replacement-shaped edit still
passes.
**Effort:** S
**Dependencies:** None.
**Risks:** Extending to more documents may surface pre-existing drift that has to
be fixed at the same time.

### P3 — Unified typed streaming events
**Objective:** One entry point for account, portfolio, order, notification, and
user events, preserving the existing per-subscription channels.
**Why now:** Carried forward from the `ws-shutdown` run; the typed events all
exist now, so the remaining work is ergonomic rather than foundational.
**Effort:** M
**Dependencies:** None.
**Risks:** Must not change existing channel behaviour under ADR 0015.

## Recommended Next Phase

**P1 — raise the coverage floor deliberately.** It is the only item that
currently makes routine work fail, and it is the one where doing nothing is
actively costly: a 2.5-point margin means the next feature ships a red gate.

## Open Questions

1. Is the 35% floor meant as a ratchet that must be raised over time, or as a
   floor to stop the gate being meaningless? The two imply very different
   amounts of work.
2. Should the mutating model endpoints get their own opt-in build tag so they
   can be verified against a real FA account, or stay permanently mock-only?
