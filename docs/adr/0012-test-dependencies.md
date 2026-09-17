# 0012 — Test-only dependencies

- Status: Accepted
- Date: 2026-09-16

## Context

The testing strategy requires detecting goroutine leaks in any test that starts
background goroutines (tickle loop, WebSocket reader, reconnect loop). The stdlib
`testing` package does not provide this. `go.uber.org/goleak` is the de facto
standard for goroutine leak detection in Go.

[TESTING.md](../TESTING.md) already mandates goleak by name, and
[ARCHITECTURE.md](../ARCHITECTURE.md) lists `testify` for assertions. This creates
an internal inconsistency: hard rule #4 (no new dependencies without an ADR) is
violated by a dependency that is already specified in other project documents.

## Decision

Add the following test-only dependencies:

- `go.uber.org/goleak` — goroutine leak detection; required by `TESTING.md`.

Do **not** add `testify` or other assertion libraries in this ADR. Use stdlib
`testing` for assertions. Adding testify requires a separate ADR.

These are `test` build-tagged imports only and are not included in any production
binary.

## Consequences

- `go test -race` and `goleak` together catch concurrent resource leaks in CI.
- Test-only deps appear in `go.mod` under a `//go:build tool` comment or a
  separate `tools_test.go` marker; `go mod tidy` places them in `require` as
  `// indirect` when imported only by test files.
- If `go.uber.org/goleak` ever needs an update, it follows normal `go get`
  workflow; no separate ADR needed for patch updates.
