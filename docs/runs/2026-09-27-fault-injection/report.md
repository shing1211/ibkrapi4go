# Report: the substrate under the retry tests

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `217bd45` (`v1.1.16`)
- **Outcome**: complete, uncommitted pending approval

## The vacuous-test risk

`internal/mockgateway` is where fault injection lives, and the SDK's retry,
error-classification and order-reply tests all depend on it. That dependency is
load-bearing in a way that fails quietly.

`pkg/ibkr/endtoend_test.go:54` injects like this:

```go
scn.SetPolicy(func(op string, _ *mockgateway.Request) *mockgateway.Fault { ... })
gw.srv = mockgateway.New(mockgateway.WithScenario(scn))
```

So the path under test is `SetPolicy` and `applyScenario` in `server.go`. Before
this run: `SetPolicy` had 3 uncovered statements and `applyScenario` had 11
uncovered blocks. `Scenario.Set`, `SetGlobal` and `FaultFor` - the declarative API
- were untested outright, at 30.2% for the file.

If that path stopped injecting, the client would receive a clean fixture where it
expected a 500. A retry test would then either pass for the wrong reason or fail
confusingly, and nothing anywhere would report that the mock was broken. The
percentage of a test harness says little about this; the behaviour has to be
asserted.

## What was added

**`scenario_test.go`** - the resolution order (policy, then per-op, then global,
then nothing), every setter and getter including that an unset order-reply body
reads as `""` so the server falls back to its default, that a policy declining
*defers* rather than suppressing, and that a zero-value `Scenario` works, which
matters because `Set` creates the map lazily.

The four injection modes are asserted end to end, because the unit test of
`FaultFor` alone would not prove anything reaches the client:

| Mode | Assertion |
|---|---|
| status | injected status and body replace the fixture; `Clear` restores it |
| no body | a status-only fault still returns parseable JSON, not an empty response |
| latency | measured, not trusted; a per-fault delay wins over the global one |
| drop connection | via a real TCP connection, since a recorder cannot see a hijack |
| timeout | must honour the client context, or every SDK timeout test would hang |

**`recorder_test.go`** - `clone` exists so a recorded snapshot is not mutated once
routing enriches the live request. Each field is checked independently, so a
partial regression is distinguishable: aliasing the body, the query, or the headers
each produce their own failure. Plus the nil-receiver and nil-request guards,
`Requests` handing out a copy of the slice rather than its backing array, `Reset`,
concurrent `Record` alongside `Requests`/`LastRequest`, and the documented
pre-routing contract that a snapshot has no `Params`.

**`stream_parse_test.go`** - `parseStreamFields` across all three accepted forms
and the whitespace, quoting and trailing-comma edges; `parseStreamConids`
including entries that must be skipped rather than fatal; `parseStreamFrame` across
both the JSON control frame and the legacy `smd+`/`umd+` text protocol, with the
negative cases - an unknown method must be ignored, because treating one as a
subscribe would push frames at a client that never asked for them.

## One test premise that was wrong

`TestServer_SessionExpiredFaultTakesPrecedence` asserted that a session-expired
scenario takes precedence over the unauthenticated response. It does not: in
`ServeHTTP` the auth check comes first, so an anonymous request got
`{"error":"not authenticated"}`.

The fix was to correct the test rather than the code, and to drop
`WithAuthRequired` so the switch is exercised on its own - it is checked inside the
protected-route block but is independent of `authRequired`. Worth recording because
the original name asserted a precedence that does not exist, and it would have
"passed" only after being weakened.

## Result

| file | before | after |
|---|---|---|
| `scenario.go` | 30.2% | **100.0%** |
| `recorder.go` | 40.6% | **96.9%** |
| `server.go` | 62.2% | 79.7% |
| `stream.go` | 66.0% | 73.1% |
| package | 75.5% | **86.6%** |

Headline coverage 65.4% -> **66.3%** against a 62% floor. The small headline move
is expected: the mock gateway is a modest share of the covered code. The value here
is the pinned behaviour, not the percentage - this run is the third in a row where
the percentage moved less than the finding.

## Mutation verification

15 mutations, all caught:

- **Fault resolution** - per-op beating the policy; global beating per-op; `Set`
  storing nothing; an unset order-reply body returning a non-empty default.
- **applyScenario** - a per-fault delay losing to global latency; a status-only
  fault returning an empty body; `DropConnection` ignored; `Timeout` ignored; the
  session-expired switch ignored.
- **Recorder** - the recorded body aliasing the live request; the recorded query
  aliasing it; `Requests` handing out its backing slice.
- **Stream parsers** - the `{"fields":[...]}` form no longer parsed; an unknown
  method treated as a subscribe; quoted field entries not unquoted.

Two of the fifteen were missed on the first pass because *my anchor strings* were
wrong, not because the tests were weak. Both were re-run against corrected anchors
and caught.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`,
`check_design`, `check_links.py`, `check_i18n.py`, `check_spec_version.py`.
Coverage 66.3% against 62%, computed with CI's own `-coverpkg`.

`codegen-verify` not re-run: no spec or generated-code change.

## Production code

None. Three test files, the changelog, and run artifacts. The mock gateway behaves
exactly as it did - these tests describe what it already did, which is the point.
