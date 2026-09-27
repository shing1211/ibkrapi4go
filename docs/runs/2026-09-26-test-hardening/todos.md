# Test Hardening - Todos

Schema: `| ID | Task | Role | Status | Depends On | Acceptance |`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T0 | Re-measure coverage with CI flags; dedupe profile blocks | orchestrator | done | - | Total reproduces `go tool cover -func` exactly: 43.3%, 6275 stmts, 62.8/pt |
| T1 | Annotate stale `-race` claim in ws-shutdown report | docs | done | T0 | Annotated in place, not rewritten |
| T2 | Remove false "replacement-blind" claim (3 locations) | docs | done | T0 | Struck; `check_design` catches replacements (main.go:265) |
| T3 | Correct stale 37.5% and goleak-framing claims | docs | done | T0 | 43.3% recorded; package-exit detection noted |
| T4 | Restate P2 around real gap (2 of 9 docs verified) | docs | done | T0 | No strictness claim remains |
| T5 | Mark ratchet question answered | docs | done | T0 | Struck, decision recorded |
| T6 | Confirm single canonical floor value | orchestrator | done | T0 | `ci.yml` only; run records are history |
| T7 | Reproduce `TestWS_Resilience` flake, read leaked stack | tester | done | T0 | `go test ./internal/ -count=2` fails on unmodified code |
| T8 | Fix cause: `StreamHub.closeAll` + `Server.Close` | backend | done | T7 | Reproducer passes; `-count=3` passes; `-race` clean |
| T9 | Remove dead `WSConn.waitForDone` | reviewer | done | T7 | Zero callers confirmed before removal |
| T10 | Per-test leak attribution, 4 files | tester | done | T8 | `pkg/ibkr` + `internal` green at `-count=2` and `-race` |
| T11 | Release v1.1.3 to GitHub + Gitee | release | done | T8,T10 | `f3e062d`, tag `1ab2b43`, both remotes verified |
| T12 | Slice 1: four wholly-uncovered managers | tester | done | T10 | 43.3% -> 48.9%; 49 funcs off 0%; 40 tests |
| T13 | Ratchet coverage floor 35% -> 48% | orchestrator | done | T12 | Gate passes at measured 48.9%, 0.9pt buffer |
| T14 | Slice 2: `rest_banking.go`, `rest.go`, `rest_accounts.go` | tester | todo | T13 | Every targeted func non-zero; full suite + `-race` green; **no ci.yml edit** |
| T14a | Slice 2a: `rest_accounts.go` 0% funcs | tester | done | T13 | 7/7 targets non-zero, 0 remaining at 0%; CI total 48.9% -> 50.0% |
| T21 | **DEFECT** `IsCompleted bool` + `omitempty` drops `false` | backend | done | T14a | One-line tag fix + regression test. Verified: no exported API change, `gofmt`/`vet`/suite/`-race`/`check_money` green. A pre-existing T14a test had encoded the bug; inverted to require explicit `false` (strictly stronger) |
| T22 | **DEFECT** connection reuse broken by `defer cancel()` | backend | done | T14a | Premise corrected: the "62 unclosed bodies" theory was **wrong** (`Body` is `[]byte`, already drained by `Parse*Response`). Real cause: `internal.Timeout` deferred `cancel()`, firing before body read and defeating `cancelOnCloseBody`. Fixed with `cancel()` on both error paths. **24 calls/24 conns -> 24 calls/0 conns** |
| T23 | Logout response body never closed | backend | review | T22 | `internal/session.go:112` discards the `*http.Response` without closing, so one connection is dropped per `Client.Close()`. Its `defer cancel()` there is correct and must stay. 3-line fix; not done — out of the approved T21/T22 scope |
| T14b | Slice 2b: `rest.go` 24 funcs at 0% | tester | todo | T14a,T21,T22 | Non-zero; no ci.yml edit |
| T14c | Slice 2c: `rest_banking.go` 21 funcs at 0% | tester | todo | T14b | Non-zero; no ci.yml edit |
| T15 | `cmd/ibkr` order validation + flag handling | tester | todo | T14c | `main()` stays uncovered; meaningful paths covered; **no ci.yml edit** |
| T16 | Re-measure and ratchet floor to Slice 2 value | orchestrator | todo | T14b,T14c,T15 | Floor never above measured |
| T17 | Docs sync across all project Markdown | docs | todo | T16 | No stale refs; summary table of files + actions |
| T18 | Release Slice 2 patch to both remotes | release | todo | T17 | **Requires explicit user approval before push** |
| T19 | Next-phase planning | planner | todo | T16 | 3-7 candidates + recommendation + open questions |
| T20 | Close-out: report, plan actuals, index row | orchestrator | todo | T17,T19 | `report.md` final; index row updated |

## Completed evidence

- T7 reproducer: `go test ./internal/ -count=2` failed on every attempt pre-fix; the
  leak was `mockgateway.(*Server).serveWS` at `stream.go:328`, spawned by
  `net/http.(*Server).Serve`, not `WSConn`.
- T12 note: budget was 548 statements (~8.7 pts if fully covered); actual +5.6
  pts. The gap is unexercised error branches in thin wrappers.
- T13 note: floor set to 48%, a 0.9-pt buffer under measured 48.9%, to absorb
  the Linux-CI / Windows-host difference.
- T14a note: orchestrator re-verified independently. All 7 targets non-zero
  (`Create` 50.0, `Status` 44.4, `Update` 61.1, `UpdateTasks` 63.2,
  `AssignTask` 53.3, `LoginMessages` 54.5, `LoginMessagesForAccount` 54.5); no
  route or fixture additions were needed; scope was 1 new test file plus an
  11-line backward-compatible refactor of `newRESTClient`. CI total 48.9% ->
  50.0%. The sub-agent also mutation-tested four of its own assertions, which
  is the right instinct.

## Orchestrator verification log

Sub-agent output is untrusted until independently checked. What was re-run by
hand, not taken on report:

- **T14a** — all 7 target functions re-measured non-zero; CI-flag coverage
  48.9% -> 50.0%; scope confirmed to 1 new file + 1 backward-compatible helper
  refactor. Note the sub-agent measured 44.5 -> 46.2% using different
  `-coverpkg` flags; the CI-flag number is the one that counts.
- **T21** — production diff re-read: one tag plus a rationale comment, no API
  change. The pre-existing test that had to change was verified to be an
  **inversion** (now requires the key present and explicitly `false`), not a
  weakening.
- **T22** — the sub-agent **refused the task** because the premise was false,
  which was correct: `GetAccountOwnersResponse.Body` is `[]byte`, so the
  prescribed `resp.Body.Close()` would not compile. The real defect was then
  reproduced by hand: **24 sequential REST calls opened 24 new connections**,
  and removing `defer cancel()` alone turned that into 0. Re-measured after the
  sub-agent's fix: **0 new connections**. The agent's goleak failure did not
  reproduce; `pkg/ibkr -count=3` is clean. Its decision to skip
  `CloseIdleConnections` was checked: `RoundTripFunc` has no such method, so
  `http.Client.CloseIdleConnections` genuinely cannot reach the base transport.

## Correction to an earlier orchestrator claim

This session recorded "66 `*WithResponse` calls, 4 `Body.Close()`, so every REST
success path leaks its response body". **That was wrong**, and the error was
inferring a leak from a grep count without checking that the type was even
closeable. The bodies are `[]byte` copies, already read and closed by
`Parse*Response`. Two earlier false conclusions in this run came from the same
habit — file-scoped greps and counts standing in for reading the code. Treat
grep-derived resource claims as hypotheses until a failing test confirms them.

## Open decision for the human

- Whether to cut a v1.1.4 patch now for T21 + T22 + the T14a coverage work, or
  keep accumulating and release once.
- Whether to fold in T23 (logout body, 3 lines) or defer it.
- Whether to finish coverage Slice 2 (T14b/T14c/T15) before releasing.

## Method notes for delegated tasks

- PowerShell splits unquoted commas in native-command args. Quote
  `-coverpkg=./pkg/ibkr,./internal,./cmd/...` or Go resolves fragments as
  package patterns.
- A `-coverprofile` run holds one block set per test binary. Dedupe by block
  before using a total for planning; naive summing reported 14.4% instead of
  43.3%.
- Routes live in **both** `routes_cpapi.go` and `routes_rest.go`. Two
  file-scoped checks in this run produced false "unrouted"/"unfixtured"
  conclusions before being corrected.
- `cli.REST()` requires OAuth2. Use the `newRESTClient` helper added in T12.
