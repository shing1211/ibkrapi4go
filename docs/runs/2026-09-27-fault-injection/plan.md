# Plan: cover the fault injection and the recorder

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `217bd45` (`v1.1.16`)

## Objective

Fourth request to "plan and implement remaining blocking items". The blocked list
has not moved and is genuinely blocked:

- D15 live verification - needs a real account
- confirming the `submitModelPortfolioOrder` collision - needs an FA paper account

This run checks whether either has an offline half left, then builds the work the
last two runs named.

## Is any of D15's work actually offline?

D15 is "`year` must be omitted from `listTaxDocumentsAvailable` because the spec
marks it optional". The live call is unreachable, but the client half is not: does
a test assert the parameter is genuinely *absent* from the wire, rather than present
and empty?

Yes - `rest_reports_e2e_test.go:296` already asserts `year` is not present in the
recorded query at all, and a second case asserts it is sent when set. The offline
half is already covered. Nothing to add; D15 stays purely live-blocked.

## The real target

The previous run's note named `stream.go` (66%) and `scenario.go` (30.2%), with the
argument that untested fault injection quietly weakens the SDK's retry tests. That
argument is worth checking rather than repeating, because it decides the priority.

Checking what `pkg/ibkr` actually installs:

```go
scn := mockgateway.NewScenario()
scn.SetPolicy(func(op string, _ *mockgateway.Request) *mockgateway.Fault { ... })
```

The SDK's retry, error-classification and order-reply tests all inject through
`SetPolicy` and `applyScenario` - not through `Scenario.Set` or `SetGlobal`. So the
path those tests depend on is `SetPolicy` (3 uncovered statements) and
`applyScenario` in `server.go` (11 uncovered blocks).

If that path silently stopped injecting, the client would receive a clean fixture
instead of a 500, the test would see success where it expected failure, and it
would pass - or fail - for the wrong reason, with nothing reporting a broken mock.
That is a vacuous-test risk in the substrate, and it is the highest-value thing
available offline.

The recorder ranks next for the same reason. `Recorder.clone` exists solely so
recorded snapshots are not mutated once routing enriches the live request. If the
copy were shallow, every "assert what was sent" test in the suite would be
asserting against a structure that changes under it.

## What this run implements

1. `scenario_test.go` - `FaultFor` resolution order, every setter/getter including
   the zero-value `Scenario`, and the four injection modes end to end: status, no
   body, latency with per-fault precedence, dropped connection, and timeout.
2. `recorder_test.go` - the deep-copy property per field, the nil-receiver and
   nil-request guards, the slice copy, reset, concurrent access, and the
   documented pre-routing contract that a snapshot has no `Params`.
3. `stream_parse_test.go` - `parseStreamFields`, `parseStreamConids` and
   `parseStreamFrame` across both wire shapes the mock accepts.

## Expected result

`internal/mockgateway` measured at 86.6% for the package, with `scenario.go` and
`recorder.go` effectively complete. The headline total moves 65.4% -> 66.3% only
because this package is a small share of the covered code - the value here is not
the percentage.

## Constraints held

- No credentials used.
- No production behaviour changed. Three test files, plus the changelog and run
  artifacts.
- No public API change.
