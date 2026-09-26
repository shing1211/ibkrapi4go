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

## Slice 1: The Four Uncovered Managers

Baseline 43.3%, result **48.9%**, so **+5.6 points** for 40 new tests. The
estimate was 8.7 points if fully covered; thin REST wrappers never reach 100%
because their error branches stay unexercised, so 5.6 is the honest number. All
49 functions across the four files moved off 0%.

| File | Functions | Before | After |
|------|-----------|--------|-------|
| `restrictions.go` | 18 | 0% | all non-zero |
| `rest_utilities.go` | 9 | 0% | all non-zero |
| `notifications.go` | 12 | 0% | all non-zero |
| `trading_accounts.go` | 10 | 0% | all non-zero |

The floor moved from 35% to **48%**, a 0.9-point buffer under the measured
value. Exactly 48.9% would also be stable, since coverage is deterministic for a
fixed suite, but the buffer absorbs platform differences between the Linux CI
runner and the Windows host used here.

### Two false starts worth recording

The routes and fixtures for all four managers already existed. Two intermediate
conclusions were wrong because a check was file-scoped:

- Comparing the error-wrapping strings (`const op = "Restrictions.Account"`)
  against route registrations suggested all 35 operations were unrouted. They
  are keyed by constants like `OpGetAccountOwners` instead, and every one is
  routed.
- Checking fixtures only in `routes_rest.go` suggested most had none. They live
  in `routes_cpapi.go`. A test asserting `BrokerageAccounts` was unrouted failed
  immediately and proved the opposite.

`cli.REST()` also requires OAuth2 to be configured, so the surface tests needed
a client wired with `WithOAuth2ClientCredentials` against the gateway's own
`/oauth2/api/v1/token` route. The existing `rest_accounts_test.go` convention
builds a hand-written `httptest` handler; the new helper reuses the gateway
fixtures instead, so assertions run against realistic bodies.

## Per-Test Leak Attribution

Both goroutine-owning packages already leak-check from `TestMain`, so nothing was
unchecked; the gap was that a leak was reported without naming the test. Four
files now assert directly, and the wiring is one edit per helper rather than one
per test:

- `pkg/ibkr/ws_test.go` - `newWSServer` covers 18 tests
- `pkg/ibkr/endtoend_test.go` - `newGateway` covers every test that uses it,
  which includes all of `managers_e2e_test.go`
- `internal/ws_test.go` - 9 sites
- `internal/session_test.go` - the two tests that start the tickle goroutine

`TestSession_TickleGoroutine_NoLeak` asserted nothing. It relied on the ambient
`TestMain` check, so it would have passed even if its own goroutine leaked. It
now asserts.

`t.Cleanup` and `defer` are both LIFO, so the assertion has to be registered
*first* in order to run *last*. Getting that wrong is not subtle: wiring the
check into `newTestClient` rather than `newGateway` made it run before the
gateway shut down, and it duly reported `net/http/httptest`'s own accept loop as
a leak in six tests.

## The Flake Was Not a Flake

`TestWS_Resilience` had flaked once and then passed roughly fifteen times,
including under deliberate CPU oversubscription. The first hypothesis was a
goroutine-settle race in `WSConn.Close`, which documents a bounded wait and
signals completion from a goroutine that is still executing.

That hypothesis was wrong. A reliable reproducer turned up instead:

    go test ./internal/ -count=2

failed on every attempt. `-count=1` passed. The leaked goroutine was not
`WSConn` at all:

    mockgateway.(*Server).serveWS
      internal/mockgateway/stream.go:328
    created by net/http.(*Server).Serve

`serveWS` parks on `c.Read(context.Background())` (stream.go:328). That context
is never cancelled, and `httptest.Server.Close` does not track hijacked
connections, so the handler outlived the server that started it. The parent
`TestWS_Resilience` failed because a `serveWS` handler from an earlier subtest
was still alive when the parent's leak check ran, which is also why the original
failure named no subtest.

### Fix

`StreamHub.closeAll` closes every registered stream socket with `CloseNow`,
which is what actually unblocks a read parked on an uncancellable context.
`Server.Close` exposes it, and the tests now call it:

- `pkg/ibkr/ws_test.go` - one line in `newWSServer`, covering 10 call sites
- `internal/ws_test.go` and `internal/ws_resilience_test.go` - 16 call sites

`CloseNow` returns before the handler unwinds, so a bounded settle is still
needed on the test side. `waitForGoroutinesToSettle` polls until the goroutines
clear, bounded at 2s. A goroutine that is merely winding down exits on its own;
a real leak never does, so the helper cannot mask one. That property is tested
in both directions rather than assumed.

The reproducer went from failing to passing, and `-count=3` now passes too.

### Incidental

`WSConn.waitForDone` was dead code with zero callers; the live path inlines the
same logic. Removed.

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
- `go build ./...`, `go vet ./...`, `gofmt -s -l .` - clean
- `go test ./...` - pass
- `go test -race ./internal/... ./pkg/ibkr/...` - pass, no data races
- `go test ./internal/ -count=2` - the reproducer, pass (failed before the fix)
- `go test ./internal/ -count=3` - pass
- `go test ./pkg/ibkr/ -count=2` - pass
- `TestWaitForGoroutinesToSettle_*` - prove the settle helper reports a real
  leak, returns immediately when clean, and tolerates a slow shutdown

## Known Limitations

- The mutating model endpoints remain unverified against a real gateway.
- `cmd/ibkr` and `cmd/ibkr-mock-gateway` contribute 318 uncovered statements to
  the denominator. They stay in `-coverpkg` by decision; only their testable
  parts will be covered, and `main()` will remain uncovered.
