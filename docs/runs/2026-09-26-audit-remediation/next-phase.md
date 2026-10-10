# Next Phase: Post-Audit Directions

The audit remediation is complete. Every gap it found is closed or explicitly
documented as a limitation. This document records what is deliberately still
open, so the next run starts from facts rather than from the impression that
everything is finished.

## Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| Coverage margin is thin | Medium | ~~37.5% against a 35% floor.~~ Later correction: the 37.5% figure came from a stale profile two days older than the tests that produced it. A fresh measurement with the CI flags gives **43.3% against the 35% floor — an 8.3-point margin**, not a 2.5-point one. The margin was never as thin as recorded; the number was wrong. Still worth deciding whether to raise the floor deliberately. |
| goleak asserted in 4 of 42 test files | Medium | `docs/TESTING.md` now says only "tests that start background goroutines", which is true but leaves most of the suite unchecked. Either extend the assertion or state the gap in the doc. Later correction: the "4 of 42" framing understates what exists. Both goroutine-owning packages already run `goleak.Find()` from `TestMain` (`internal/transport_test.go`, `pkg/ibkr/endtoend_test.go`), so leaks **are** caught at package exit. The real gap is per-test *attribution*, not detection. |
| Model endpoints unverified against a real gateway | Low | The mock cannot route `SubmitModelPortfolioOrder` separately, and the integration suite stays read-only. Verifying dispatch needs an FA-enabled paper account and an opt-in build tag. |
| `TestWS_Resilience` timing sensitivity | Low | Flaked once under full-suite parallel load on Windows, then passed repeatedly. Later correction: "timing sensitivity" is the wrong diagnosis. The parent test failed with no subtest named, so it tripped its own deferred `goleak.VerifyNone(t)`, which has no settle window — unlike the 50 ms settle in `TestMain`. Leading hypothesis is a goroutine that outlives `WSConn.Close` under load, not a timing assertion. |

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

### P2 — Extend the design checker to the unverified documents
**Objective:** `check_design` validates `01-transport.md` (middleware chain) and
`03-managers.md` (manager method counts). Extend it to the seven design
documents it does not read: 02, 04, 05, 06, 07, 08, and 09.
**Why now:** The checker's existing two checks are sound — it verifies both
presence and order, and a replacement-shaped edit (delete one middleware, add
another) is caught, because the new layer is absent from the diagram. An earlier
version of this item claimed the opposite. The genuine gap is breadth, not
strictness.
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

**P1 — raise the coverage floor deliberately.** It is the one item where the
decision is genuinely open. Correction to the original rationale: this was
written when the margin was believed to be 2.5 points, which would have made
routine work fail. The real margin is 8.3 points, so nothing is urgent — the
argument for doing it is that an unraised floor stops being a ratchet.

## Open Questions

1. ~~Is the 35% floor meant as a ratchet that must be raised over time, or as a
   floor to stop the gate being meaningless?~~ **Answered: ratchet.** Raise it in
   reviewable slices, each setting the floor to the freshly measured value, and
   never above it. Tracked in the `2026-09-26-test-hardening` run.
2. Should the mutating model endpoints get their own opt-in build tag so they
   can be verified against a real FA account, or stay permanently mock-only?
