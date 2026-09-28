# Run: lint gate, CLI tests, and design-doc coverage

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `860cafd` (`v1.1.8`)
- **Scope**: everything the previous run's `next-phase.md` listed as remaining,
  plus the real defects that fixing the lint gate exposed.

## Goal

Make the lint gate actually enforce, fix what it finds, give `cmd/ibkr` its first
tests, and close the last unverified design document. The standing decision from
the previous run ("document the lint state, change nothing") is superseded by the
instruction to finish all remaining work.

## The headline: the lint gate was not a gate

`.golangci.yml` declared `version: 2` but was written in the v1 schema. golangci-lint
2.9.0 rejected it outright on three counts:

```
jsonschema: "issues" does not validate: additional properties
            'exclude-dirs', 'exclude-use-fatal', 'exclude-rules' not allowed
jsonschema: "version" does not validate: got number, want string
jsonschema: "" does not validate: additional properties 'linters-settings' not allowed
```

The job had never analysed a line of code. The "733 findings" recorded in earlier
runs was never a real count. After migration the true count was 192.

Migrating it was not a formality. It surfaced five real defects that no other gate
was watching for.

## Real defects found and fixed

### `SubmitDocument` accepted a `mimeType` and threw it away

The signature takes `mimeType string`, defaults it to `application/pdf`, and then
calls `w.CreateFormFile`, which hardcodes `application/octet-stream`. The argument
never reached the wire. The part is now built by hand so the caller's type is
honoured, with the filename escaped per RFC 7578.

### The default response size limit was never applied

`internal/transport.go` declared `defaultMaxResponseBytes = 32 << 20`, and applied
the cap only `if cfg.MaxResponseBytes > 0`. The constant was therefore unreferenced
and every default-configured client had **no limit at all** on the response body it
buffers. The cap is now always applied.

### `MaxBytes` truncated silently, and claimed otherwise

`maxBytesReader` wrapped the body in an `io.LimitedReader` and returned a short
read. Its own doc comment said "Truncation is detected by the caller via a short
read and surfaced as a typed error" - no caller did that, anywhere. A capped
response became corrupt JSON with no indication why. It now returns the new
`internal.ErrResponseTooLarge`.

### Enabling the cap broke every WebSocket, and the tests caught it

A `101 Switching Protocols` body is the hijacked connection itself, and
gorilla/websocket requires it to be an `io.ReadWriteCloser`. Wrapping it in a
reader made every dial fail with `response body is not a io.ReadWriteCloser`. This
never surfaced before because the cap defaulted to off. 1xx responses now pass
through unwrapped, with a regression test.

### A panic in a subscription goroutine was swallowed with a lie in the comment

`pkg/ibkr/ws.go` recovered a panic in the long-lived subscription goroutine into an
empty branch whose comment read "sink is already gone; log and exit". It did not
log. A panic there would have made a subscription go silently quiet. It now logs
through the client logger and closes the subscription.

## Two more findings worth naming

- **`errorlint` in a test was making that test vacuous.**
  `internal/ws_resilience_test.go` compared errors with `==` against
  `ErrWSDisconnected`. The WebSocket layer wraps its errors, so the comparison
  could never match and the test was not detecting the disconnect it exists to
  detect. Now `errors.Is`.
- **`unused` was reporting a constant that documented an unimplemented intent**
  (`defaultMaxResponseBytes`) and a struct field that was set then overwritten
  (`mimeType`). Both turned out to be real defects rather than dead code, which is
  the argument for treating `unused` as a defect-finder rather than a cleanup.

## Lint triage

The 192 findings were not fixed by suppression. Roughly:

| Linter | Was | Disposition |
|--------|-----|-------------|
| errcheck | 83 | 33 unchecked `fmt.Fprintf` made explicitly discarded, 2 ignored `json.Unmarshal` annotated; the remaining deferred `Close` calls are this codebase's convention and are excluded with a written reason |
| revive | 32 | doc comments written for all 32 exported symbols |
| bodyclose | 24 | 6 in `pkg/ibkr` are false positives: every response goes through `decodeJSON`, which does `defer resp.Body.Close()`. Excluded with that reason; 1 in `internal` is the standard gorilla `Dial` pattern, annotated inline |
| contextcheck | 14 | would require adding a `ctx` parameter to the public `Close()`, a breaking change. Excluded for the WebSocket layer with that reason; still enforced for REST and OAuth |
| gosec | 9 | 1 real-ish (TLS `MinVersion` now set explicitly in the mock gateway); 4 G101 are metric names, an OAuth URN and a public endpoint URL; 4 annotated inline with reasons |
| staticcheck | 8 | 4 auto-fixed, 4 by hand. `ST1012` wants `TransitionError` renamed to `ErrTransition`; it is published API, so it carries a documented `//nolint` instead |
| unconvert | 9 | 9 redundant conversions removed |
| unused | 10 | 4 dead wire structs, 2 dead helpers, 1 dead test helper, plus the two above |
| ineffassign | 1 | resolved by the `mimeType` fix |
| errorlint | 2 | `errors.Is` |

Every exclusion in `.golangci.yml` carries a comment explaining why, so the next
reader can tell a deliberate decision from an oversight.

## Design-doc coverage: 8 of 9 to 9 of 9

`docs/design/07-money-and-numbers.md` was deliberately unverified, on the reasoning
that `check_money.py` already enforces its main claim. That reasoning is right about
the *negative* claim and wrong about the positive one. The document also **names**
four fields as `json.Number`, and `check_money.py` cannot see that: it proves no
float exists under `pkg/ibkr`, not that a particular field in *generated* code has a
particular type. Since `client/` is regenerated from the spec, a spec edit could
quietly put a float back.

`checkMoneyNumberFields` closes that: it parses the fields the document names and
requires each to be `json.Number` in `client/client.gen.go`. Five red cases plus a
control prove it bites, including the case where the naming sentence is removed
entirely - an empty field set must not read as "nothing to check".

## cmd/ibkr: first tests in the package

The package had no test files at all. `run()` and `parseGlobalFlags()` read
`os.Args` directly, so nothing was reachable. Both now take their arguments and
output streams as parameters.

This surfaced its own bug immediately: `runCompletion()` still read `os.Args`, so
the first test run failed with `unsupported shell "-test.timeout=10m0s"` - the test
binary's own flags. The completion command is now parameterised too, and its
generated scripts are asserted to mention every advertised command, which is a real
staleness check on the dispatch table.

## Not a gap: 193 documented operations vs 191 fixtures

This looked like missing coverage and is not. `docs/SPEC.md` has 193 rows but only
192 unique operation IDs - `getTradingSchedule` is documented under two methods.
All 192 have mock-gateway op constants. 191 have fixture bodies;
`submitModelPortfolioOrder` is deliberately unrouted because its path normalises to
the same shape as `submitNewOrder` and the router could never select it, which
`TestModelOrderRouteCollision` already pins.

## Also fixed: brittle red tests

Six design-checker red cases asserted on error messages containing hard-coded line
numbers (`pkg/ibkr/trade.go:100`). Adding a three-line doc comment to `trade.go`
shifted them and the cases failed with "failed for the wrong reason" - the checker
had in fact done its job. The comparison now strips line numbers from both sides,
so a case asserts on the file, symbol and problem rather than on an incidental
offset.

## Verification

| Check | Result |
|-------|--------|
| `golangci-lint run` | **0 issues**, exit 0 |
| `go build ./pkg/... ./internal/... ./cmd/...` | clean |
| `go vet ./...` | clean |
| `gofmt -l` | clean |
| `go test ./...` | pass |
| `go test -race` | clean |
| coverage (CI-equivalent) | **60.0%** (was 59.2%) |
| `check_money.py` | OK, 55 files |
| `check_design` | **9 of 9** documents verified |
| `check_links.py` / `check_i18n.py` / `check_spec_version.py` | pass |
| codegen drift | 0 lines |

Every behavioural fix carries a mutation check: reverted, confirmed red with a
diagnostic naming the defect, restored.

## Known remaining

- **`TestWS_DialAndSubscribe` is a wall-clock flake.** It asserts "at least one
  update after 500ms" and misses under parallel load. It passes 4/4 in isolation.
  Not caused by this run, and not fixed here: loosening the window would mask a
  real delivery regression. It is a test-design problem, not a code one.
- **D15 has not been confirmed against a live gateway** (carried from v1.1.8).
- **`contextcheck` is disabled for the WebSocket layer** rather than satisfied.
  Making it pass means adding a `ctx` parameter to the public `Close()`.
