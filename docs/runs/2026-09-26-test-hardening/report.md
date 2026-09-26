# Test Hardening - Report

Status: in progress. Steps 0 and 1 are complete; steps 2 through 4 are open.

## What This Run Set Out To Do

The `audit-remediation` run left five open items. Four were selected: raise the
coverage floor deliberately, extend goroutine-leak checking, fix the
`TestWS_Resilience` flake, and correct the run's own false claims. The fifth,
verifying the mutating model endpoints against a real gateway, needs an
FA-enabled paper account and stays open.

## The Measurement Was Wrong

The inherited number was 37.5% against a 35% floor, described as a thin 2.5-point
margin. Re-running the CI command gives **43.3%**, an 8.3-point margin. The stale
`coverage.out` was two days older than the tests that produced it, and it made the
margin look roughly three times worse than reality.

The practical effect: "the margin is thin, a new feature will ship a red gate" was
never true, so the urgency argument for raising the floor was unfounded. The floor
is still worth raising, but as a deliberate ratchet rather than a rescue.

## Four False Claims Corrected

1. **Stale `-race` note** in the `ws-shutdown` report. It described an
   environment property, not a project limitation; a later host with
   `CGO_ENABLED=1` and a C compiler runs `-race` cleanly. Annotated in place, in
   the style of the earlier `gofmt` correction.
2. **"The coverage gate cannot detect a replacement."** Wrong twice: misattributed
   to the coverage gate, and false regardless. `check_design` verifies presence
   *and* order, so deleting one middleware and adding another fails the presence
   check on the new layer. The row is struck rather than restated.
3. **"goleak asserted in only 4 of 42 test files."** Understated what exists. Both
   goroutine-owning packages call `goleak.Find()` from `TestMain`, so leaks are
   caught at package exit. The genuine gap is per-test attribution.
4. **"37.5% against a 35% floor, so the margin is thin."** Superseded by
   measurement, as above.

P2 in the inherited `next-phase.md` was rebuilt around the real gap: `check_design`
reads 2 of the 9 design documents, so the weakness is breadth, not strictness.

## Two Measurement Traps

Both were hit in this run's first pass and are recorded in `plan.md`:

- PowerShell splits unquoted commas in native-command arguments, so
  `-coverpkg=a,b` arrives split and Go resolves the fragments as package
  patterns. Quote comma-bearing flags.
- A `-coverprofile` run holds one block set per test binary. Naively summing
  blocks reported 14.4% instead of 43.3%. Deduplicating by block reproduces
  `go tool cover -func` exactly. Any coverage number used for planning must
  reproduce the tool's own total.

## Verification

- `scripts/check_links.py` - all local markdown links resolve
- `scripts/check_i18n.py` - 6 languages consistent
- Coverage total cross-checked against `go tool cover -func`

## Known Limitations

- The mutating model endpoints remain unverified against a real gateway.
- `cmd/ibkr` and `cmd/ibkr-mock-gateway` contribute 318 uncovered statements to
  the denominator. They stay in `-coverpkg` by decision; only their testable
  parts will be covered, and `main()` will remain uncovered.
- The `TestWS_Resilience` flake cause is still a hypothesis, not a confirmed
  finding. Reproduction is step 2.
