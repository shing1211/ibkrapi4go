# Testing

Testing strategy for ibkrapi4go.

## Test tiers

| Tier | Location | Build tag | Needs gateway | Runs in CI |
|------|----------|-----------|---------------|------------|
| Unit | `*_test.go` next to source | — | No | ✅ |
| Session | `internal/session_test.go` | — | No (fakeAPI) | ✅ |
| Manager e2e | `pkg/ibkr/*_test.go` | — | No (httptest) | ✅ |
| Codegen | `scripts/validate_codegen.sh` | — | No | ✅ (scheduled too) |
| Integration | `test/` | `integration` | Yes (paper) | On demand |

## Unit tests

- Use `httptest.NewServer` to emulate the gateway. No network, no account.
- Assert on **request** (method, path, headers, body) and **response** handling.
- Never hit the real API in unit tests.

Fixtures:

- Store representative JSON responses under `testdata/`.
- Derive fixtures from the spec where possible so they track schema changes.

## WebSocket tests

- Run a local `httptest` server that upgrades to WebSocket and echoes frames.
- Cover: subscribe → receive → unsubscribe, reconnect after forced drop, buffer
  overflow policy, and `goleak` after `Close`.

## Concurrency & leaks

- Every test that starts goroutines asserts no leak via `go.uber.org/goleak`.
- Session and manager e2e tests: use a 50ms post-`m.Run()` sleep before
  `goleak.Find()` to allow the scheduler to reap exited tickle goroutines before
  the leak check runs.
- Transport tests: use `goleak.VerifyTestMain` directly.
- `pkg/ibkr` e2e tests exercise the real transport + generated client against an
  `httptest` gateway, asserting request headers, string-money precision, and
  sentinel error mapping.
- Run with `-race` in CI (`make test-race`).

## Integration tests

```go
//go:build integration
```

- Require a running, browser-authenticated Client Portal Gateway and a **paper
  account**.
- Read `IBKR_GATEWAY`, `IBKR_USERNAME`, `IBKR_PASSWORD` from the environment.
- Never run write operations (orders/transfers) against a live account.
- Skipped automatically when `IBKR_GATEWAY` is unset.

## Money and ADR checks

- `scripts/check_money.py` (run by `make check`) fails if any exported struct
  field under `pkg/ibkr` is `float32`/`float64` (ADR 0008). Money and quantities
  are `string`/`json.Number`.
- Order mutation safety is tested by asserting the outbound request count:
  `TestOrders_MutationsAreSingleAttempt` verifies exactly one request on a 5xx
  for submit, modify, and cancel (ADR 0009).
- Order submissions are asserted to serialize money/quantity as JSON strings.

## Codegen validation

```bash
make codegen-verify
```

Regenerates into a temp dir and diffs against the committed `client/`. Also run
on a schedule, because the upstream spec can change without a commit here.

## Coverage

- Target: meaningful coverage on `internal/` (session, transport, ratelimit,
  retry) and managers; generated code is excluded.
- `make coverage` produces `coverage.out` and an HTML report.

## Fakes

- Prefer a hand-written fake gateway (`httptest`) over interface mocks.
- Where mocks are needed, use small interfaces so fakes are easy.

## What we do not do

- No tests that require network access in CI.
- No golden-file tests over generated code (use `codegen-verify` instead).
- No live-account tests.
- WebSocket streaming not yet implemented (pending).

## Required cases (minimum)

| Area | Cases |
|------|-------|
| Session | happy path, tickle failure → EXPIRED, re-init, idempotent Close, `goleak` |
| Transport | header injection, request id, 401 handling, error parsing, redaction |
| Rate limit | steady-state, burst, `Retry-After`, unsafe methods not retried |
| Errors | `errors.Is/As`, status mapping |
| Pagination | multi-page positions |
| Orders | reply/confirmation flow, **no auto-retry on mutation** |
| Streaming | subscribe/unsubscribe, reconnect, overflow, `goleak` |
