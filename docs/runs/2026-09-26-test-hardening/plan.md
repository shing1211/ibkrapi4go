# Test Hardening - Plan

## Objective

Close the open items recorded by the `2026-09-26-audit-remediation` run: correct
the false claims it inherited, stabilise the WebSocket resilience test, extend
goroutine-leak attribution, and raise the coverage floor deliberately in slices.

## Starting State

Measured, not assumed. Every number below comes from a fresh run of the CI
coverage command, not from a carried-forward figure.

| Fact | Value | How established |
|------|-------|-----------------|
| Coverage | 43.3% | `go test -coverprofile ... -coverpkg=./pkg/ibkr,./internal,./cmd/... ./...` |
| Floor in `ci.yml` | 35% | single canonical source: `.github/workflows/ci.yml:76` |
| Real margin | 8.3 points | 43.3 - 35 |
| Total statements | 6275 | deduplicated profile blocks |
| Uncovered statements | 3557 | same |
| Statements per 1.0 point | 62.8 | 6275 / 100 |
| goleak in `TestMain` | `internal`, `pkg/ibkr` | `goleak.Find()` at package exit |
| Per-test goleak | 4 of 42 files | live filesystem count |

## Superseded Figures

The `audit-remediation` run recorded 37.5% and described the margin as thin. That
figure came from a `coverage.out` two days older than the tests that produced it.
It is wrong, and the thin-margin concern it created does not exist.

The same run recorded "goleak asserted in only 4 of 42 test files" and implied
most of the suite was unchecked. Both goroutine-owning packages already leak-check
at package exit, so detection exists; the real gap is per-test attribution.

## Method Notes

Two measurement traps, both hit during this run's first pass:

1. **PowerShell splits unquoted commas in native-command arguments.**
   `-coverpkg=a,b` arrives as two arguments and Go resolves the fragments as
   package patterns. Quote any comma-bearing flag.
2. **A `-coverprofile` run contains one block set per test binary.** Summing
   blocks naively double-counts and reported 14.4% instead of 43.3%. Deduplicate
   by block key and treat a block as covered if any binary covered it. The result
   must reproduce `go tool cover -func` exactly before it is used to plan work.

## Steps

| Step | Scope | Gate |
|------|-------|------|
| 0 | Re-measure coverage; rank uncovered statements | Deduplicated total matches `go tool cover` |
| 1 | Correct the false doc claims | `check_links.py`, `check_i18n.py` |
| 2 | Fix `TestWS_Resilience` at the cause | N consecutive full-suite runs green |
| 3 | Extend per-test goleak for attribution | Full suite green, `-count=3` on both packages |
| 4 | Raise the floor in slices | Floor never above measured value |

Steps 2 and 3 precede 4 deliberately: new coverage tests should inherit
per-test leak checking rather than hide behind package-exit detection.

## Slice 1 Target

The four wholly-uncovered managers: `restrictions.go` (234), `rest_utilities.go`
(119), `notifications.go` (112), `trading_accounts.go` (83). 548 statements,
about 8.7 points. All are thin REST wrappers following the same shape as the
already-covered managers, so the existing mock-gateway fixtures apply directly.

`cmd/ibkr` and `cmd/ibkr-mock-gateway` stay in `-coverpkg`. Their `main()` and
Cobra wiring stay uncovered; only the parts worth testing (order validation, flag
handling) get tests. The denominator is not trimmed to flatter the number.

## Non-Goals

- No new dependencies. `goleak` is already required.
- No edits to `client/*.gen.go`.
- No ADR 0009 relevance: no order-mutation retry behaviour changes here.
