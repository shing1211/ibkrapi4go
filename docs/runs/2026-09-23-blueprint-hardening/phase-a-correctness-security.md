# Phase A — Correctness & Security (P0) — Detailed Plan

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Base commit:** `96d7d75`
- **Purpose:** Unblock CI and eliminate correctness/security defects before any
  other work. This is the critical path.

---

## A1 — Fix broken build

- **Role:** backend · **Size:** S · **Deps:** — · **Wave:** 0

**Objective:** Make `go build ./...` pass; `examples/mock/error-handling.go` and
`examples/mock/main.go` both declare `main` and `defaultGatewayURL` in the same
`package main`.

**Files:**
- Read: `examples/mock/main.go`, `examples/mock/error-handling.go`, `examples/README.md`
- Modify: relocate/repackage the error-handling example (preferred: move to
  `examples/mock-error-handling/main.go` as its own package, update
  `examples/README.md` link) OR add a build tag so only one `main` compiles.

**Approach:** Prefer a new sibling directory so both examples remain runnable
via `go run ./examples/<name>`. Update the README index link accordingly.

**Acceptance:** `go build ./...` exits 0; `go run ./examples/mock` and the new
example both build.

**Verification:** `go build ./...` · `go vet ./...`

**Constraints:** Do not change SDK code. Keep the mock example's behavior.

---

## A2 — Fix `scripts/check_money.py`

- **Role:** backend · **Size:** M · **Deps:** — · **Wave:** 0

**Objective:** The regex `^\s+([A-Z][A-Za-z0-9_]*)\s+\*?\[\]?(float32|float64)\b`
only matches slice-of-float exported fields; it misses scalar, pointer, map, and
array types. It also only scans `pkg/ibkr/*.go` (non-recursive, tests excluded).

**Files:**
- Modify: `scripts/check_money.py`

**Approach:**
1. Replace the regex logic with an AST-free but robust matcher (or use `go/ast`
   via a small Go helper if simpler) covering: `T`, `*T`, `[]T`, `[N]T`,
   `map[K]T`, and function params/returns where feasible.
2. Scan `pkg/ibkr/`, `internal/`, and `client/` (client may need an allowlist/
   ignore file for known generated non-money floats — document it).
3. Keep `make money-check` wiring intact.

**Acceptance:** A seeded `Money float64` (scalar), `*float64`, `map[string]float64`,
and `[2]float64` are all flagged; a clean tree passes.

**Verification:** `python scripts/check_money.py`; add a documented fixture test.

**Constraints:** No new Python dependencies (stdlib only).

---

## A3 — Fix money precision leaks

- **Role:** backend · **Size:** M · **Deps:** A2 · **Wave:** 0

**Objective:** ADR 0008 requires `string`/`json.Number` for money/quantity. Three
violations exist:

1. `pkg/ibkr/rest_banking.go` — helpers `strToF32`/`strPtrToF32Ptr` (lines ~22-40)
   parse decimal strings to `float32` and are applied to **outgoing**
   `Amount`/`Quantity`/`TransferPrice`/`TransferQuantity` (lines ~591, 638, 672,
   724, 772-775, 826-829, 875, 887, 950, 962, 1056, 1104).
2. `pkg/ibkr/rest_banking.go` — `CashBalance float32` decoded (~1023) then
   `fmt.Sprintf("%.2f")` (~1033) into `WithdrawableFundsResult.CashBalance string`.
3. `pkg/ibkr/rest.go` — tax voucher `DivAmount/Fee/Quantity/WithHeldAmount *float32`
   (~881-887) converted via `float32ToStr` (~912-916).

**Files:**
- Read: `pkg/ibkr/rest_banking.go`, `pkg/ibkr/rest.go`, `pkg/ibkr/trade.go`
  (reference: `orderTicketJSON` hand-builds decimal strings ~486-498),
  `docs/adr/0008-numeric-precision.md`
- Modify: `pkg/ibkr/rest_banking.go`, `pkg/ibkr/rest.go`
- Possibly modify: `scripts/patch_spec.py` (add `x-go-type` overrides if the
  generated `client/` float types are the root cause) — if so, regenerate.

**Approach:** Hand-build request JSON with decimal strings for transfer
operations (mirroring `trade.go`). For responses, decode into `json.Number`/
`string` (do not round-trip through float). Prefer fixing at the generated-type
level via `patch_spec.py` where the field originates there.

**Acceptance:** No `float32`/`float64` cast on any money/quantity path; round-trip
tests preserve exact decimal values; `make money-check` passes.

**Verification:** `make money-check` · new round-trip tests · `go test ./pkg/ibkr/...`

**Constraints:** If `patch_spec.py` changes, run `make codegen` + `make codegen-verify`.
Never hand-edit `client/*.gen.go`.

---

## A4 — Fix timeout body-cancellation

- **Role:** backend · **Size:** S · **Deps:** — · **Wave:** 0

**Objective:** `internal/transport.go:138-152` uses `defer cancel()`, which fires
when `RoundTrip` returns (after headers), cancelling the request context before
the caller reads the body. Large/slow success bodies can then fail with
`context.Canceled`.

**Files:**
- Read: `internal/transport.go`
- Modify: `internal/transport.go`

**Approach:** Return a body wrapper whose `Close` calls `cancel()` (standard
`http` pattern), so cancellation happens after the body is consumed/closed.

**Acceptance:** A test that reads a large, slowly-streamed body through the
timeout middleware succeeds and does not hit `context.Canceled`.

**Verification:** new `internal/transport` test · `go test -race ./internal/...`

---

## A5 — Bound success-response sizes

- **Role:** backend · **Size:** M · **Deps:** — · **Wave:** 0

**Objective:** Success bodies are read with unbounded `io.ReadAll`
(`pkg/ibkr/response.go:42`; 193 occurrences in `client/client.gen.go`). Only error
(4 KiB) and OAuth (1 MiB) bodies are bounded.

**Files:**
- Read: `pkg/ibkr/response.go`, `internal/transport.go`, `oapi-codegen.yaml`
- Modify: `pkg/ibkr/response.go` (+ generated-code strategy in `oapi-codegen.yaml`
  / `scripts/patch_spec.py` if a global limit is needed)

**Approach:** Introduce a configurable max response size (default sane limit,
e.g. 32 MiB) enforced via `http.MaxBytesReader` or `io.LimitReader` in the
central decode path. Ensure the limit is overridable via an `Option`.

**Acceptance:** Oversized response yields a typed error, not an OOM; normal
responses unaffected. Test covers the boundary.

**Verification:** new test in `pkg/ibkr` · `go test -race ./pkg/ibkr/...`

**Constraints:** Do not break streaming (`internal/ws.go`) — this is HTTP only.

---

## A6 — Harden redaction

- **Role:** security · **Size:** S · **Deps:** — · **Wave:** 0

**Objective:** `internal/transport.go:179-181` redacts only `Authorization|Cookie|Set-Cookie`
header *lines*. Missing: bare bearer tokens, `sess=` cookie values without a
`Cookie:` prefix, `x-csrf-token`, and form fields `client_secret`/`refresh_token`.

**Files:**
- Read: `internal/observability.go`, `internal/transport.go`, `internal/oauth.go`
- Modify: `internal/transport.go`, `internal/observability.go`

**Approach:** Add structural redaction patterns for the missing secrets; apply at
all log sites (HTTP, OAuth, WS). Extend `internal/observability_test.go`.

**Acceptance:** Tests prove each secret class is masked; no secret appears in
formatted logs.

**Verification:** `go test ./internal/...`

---

## A7 — Sanitize `Error.Message`

- **Role:** security · **Size:** S · **Deps:** — · **Wave:** 0

**Objective:** `docs/ERRORS.md:21` promises error messages are sanitized, but only
logs redact. `Error.Message` can still carry unsanitized broker payloads.

**Files:**
- Read: `internal/transport.go:186-255`, `internal/errors.go`, `internal/observability.go:104-116`
- Modify: `internal/transport.go`, `internal/errors.go`

**Approach:** Apply the redaction function at error construction for
user-visible `Message`; keep a non-redacted detail field for internal logs only.

**Acceptance:** A crafted error containing a secret yields a redacted
`Error.Message`; test added.

**Verification:** `go test ./internal/...`

---

## Phase A Gate

All of: `go build ./...` · `go vet ./...` · `go test -race ./...` ·
`make money-check` · `make check` pass. No `/*.gen.go` hand edits.
