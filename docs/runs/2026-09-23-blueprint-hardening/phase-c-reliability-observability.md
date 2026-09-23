# Phase C — Reliability & Observability (P1/P2) — Detailed Plan

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Depends on:** Phase A (A1 build fix; C1/C8 need a green build)
- **Purpose:** Cover the untested streaming core, make timing tests deterministic,
  close metric/tracing/health gaps.

---

## C1 — `internal/ws.go` unit tests

- **Role:** tester · **Size:** M · **Deps:** A1 · **Wave:** 2

**Objective:** `internal/ws.go` is **0.0% covered** across every function
(`DialWS`, `Subscribe`, `readLoop`, `writeLoop`, `pingLoop`, `reconnect`,
`dispatch`, `parseSystemFrame`, `wsScalarString`, `backoffDelay`, etc.). All WS
coverage currently lives in `pkg/ibkr` and isn't credited to `internal` (no
`-coverpkg`).

**Files:**
- Read: `internal/ws.go`, `internal/mockgateway/stream.go`, `pkg/ibkr/ws.go`
- Create: `internal/ws_test.go`

**Approach:** Unit-test the pure/near-pure helpers directly against the mock
stream hub: `parseSystemFrame`, `wsScalarString`, `wsReservedField`, `dispatch`,
`wsURLFromGateway`, `backoffDelay`, `jsonNumberToInt64`. Use the seed-derived
deterministic hub (`WithSeed`) for ordering.

**Acceptance:** `internal/ws.go` materially covered; tests pass under `-race`.

**Verification:** `go test -race ./internal/... -cover`

---

## C2 — Fix flaky `TestWS_SystemUpdates`

- **Role:** tester · **Size:** S · **Deps:** C1 · **Wave:** 2

**Objective:** `pkg/ibkr/ws_test.go:218` intermittently fails (`gotStatus=false`)
because the mock sends the `sts connected` frame at connection accept
(`internal/mockgateway/stream.go:296-298`) before the client registers its
subscription, so `deliverSystem` can miss it.

**Files:**
- Modify: `internal/mockgateway/stream.go` (emit `sts` on subscribe/Hello, after
  registration) and/or `pkg/ibkr/ws_test.go` (tolerate ordering)

**Approach:** Move the initial `sts` emission to after the subscription is
registered so it cannot be dropped.

**Acceptance:** Test passes 20/20 runs.

**Verification:** `go test -race ./pkg/ibkr/... -run TestWS_SystemUpdates -count=20`

---

## C3 — Resilience tests

- **Role:** tester · **Size:** M · **Deps:** C1 · **Wave:** 2

**Objective:** Missing tests for reconnect storms, heartbeat/ping timeouts,
cancellation-during-write, duplicate events, and out-of-order updates.

**Files:**
- Read: `internal/ws.go`, `internal/mockgateway/stream.go`, `internal/fault_injection_test.go`
- Create/modify: `internal/ws_resilience_test.go`, mock stream script hooks

**Approach:** Add scripted stream scenarios (drop/reconnect bursts, delayed frames,
duplicate `_updated`, missing/out-of-order sequence) with short
`PingInterval`/`PongTimeout` injected.

**Acceptance:** Tests pass under `-race`; no goroutine leaks (`goleak`).

**Verification:** `go test -race ./internal/... ./pkg/ibkr/...`

---

## C4 — Injectable clock/dialer

- **Role:** backend · **Size:** M · **Deps:** C1 · **Wave:** 2

**Objective:** No clock abstraction exists — direct `time.Now`/`NewTicker` in
`internal/session.go:315`, `internal/ws.go:338`, `internal/breaker.go:139`,
`internal/oauth.go:121`; `websocket.Dial` called directly (`internal/ws.go:224`).
Timing tests use real sleeps (flaky/slow). `internal/fake/` is dead code.

**Files:**
- Create: `internal/clock.go` (Clock interface + real impl)
- Modify: `internal/session.go`, `internal/ws.go`, `internal/breaker.go`,
  `internal/oauth.go` (accept injected Clock); `internal/ws.go` or `WSOptions`
  (accept injected dial func)
- Use or delete: `internal/fake/`

**Approach:** Introduce a minimal `Clock` interface (`Now`, `NewTicker`,
`After`) with a real default; inject via existing config structs. Add a fake in
tests. Wire a dialer function through `WSOptions`.

**Acceptance:** Timing-sensitive tests use the fake clock; no new deps.

**Verification:** `go test -race ./internal/...`

---

## C5 — Missing metrics

- **Role:** backend · **Size:** M · **Deps:** — · **Wave:** 2

**Objective:** `internal/metrics.go` has HTTP latency, rate-limit waits, breaker
state, OAuth refreshes, and WS reconnects — but **no** retry counts, dedicated
429/pacing, heartbeat failures, dropped events, queue depth, or order lifecycle
durations.

**Files:**
- Modify: `internal/metrics.go`, `internal/retry.go` (add `Metrics`),
  `internal/session.go` (tickle metrics; add `Metrics` to `SessionConfig`),
  `internal/ws.go` (queue depth gauge, dropped counter, subscription gauge),
  `pkg/ibkr/trade.go` (order duration histograms), `docs/OBSERVABILITY.md`

**Approach:** Add the missing instruments and emit at the identified sites;
re-export via `pkg/ibkr`. Update docs to match the actual metric set.

**Acceptance:** New metrics emitted and asserted in tests; docs match code.

**Verification:** `go test ./internal/... ./pkg/ibkr/...`

---

## C6 — Composite health/readiness probe

- **Role:** backend · **Size:** M · **Deps:** C5 · **Wave:** 2

**Objective:** Only `SessionManager.State()/Status()` exist. No probe that
distinguishes transport availability, authenticated session validity,
market-data authorization, and trading readiness.

**Files:**
- Read: `pkg/ibkr/auth.go`, `internal/session.go`
- Modify/Create: `pkg/ibkr/health.go` (e.g. `Health(ctx) HealthReport`)

**Approach:** Compose existing signals (session state, validation) into a
structured report with per-dimension status. No destructive calls.

**Acceptance:** `Health(ctx)` returns structured, documented statuses; tested
against the mock gateway (healthy + expired-session cases).

**Verification:** `go test ./pkg/ibkr/...`

---

## C7 — Real OTel tracing bridge

- **Role:** backend · **Size:** L · **Deps:** C5 · **Wave:** 2

**Objective:** `contrib/otel` bridges **metrics only**; no tracer/spans exist
anywhere. Required spans: outbound REST, WS lifecycle, event decode, subscription
changes, order reconciliation.

**Files:**
- Read: `internal/observability.go` (Telemetry interface), `contrib/otel/otel.go`
- Modify: `internal/observability.go` (extend Telemetry hooks for lifecycle),
  `internal/transport.go`, `internal/ws.go`, `pkg/ibkr/trade.go`
- Modify: `contrib/otel/otel.go` (implement a `Telemetry`→span bridge)

**Approach:** Extend the existing callback interface with span-style hooks; keep
the core free of an OTel dependency (bridge lives in `contrib/otel`). Propagate
`X-request-id` into spans.

**Acceptance:** Spans emitted for REST + WS lifecycle when the bridge is wired;
`contrib/otel` tests pass.

**Verification:** `go test ./contrib/otel/...` (separate module) +
`go test ./internal/...`

**Constraints:** No new deps in the root module (ADR 0004).

---

## C8 — Fuzz in CI + WS/REST-error targets

- **Role:** tester · **Size:** S/M · **Deps:** A1 · **Wave:** 2

**Objective:** 51 fuzz targets exist but **never run in CI**; documented corpus
dirs don't exist; malformed/fragmented WS frames and REST error decoding are not
fuzzed.

**Files:**
- Modify: `Makefile` (add `fuzz` target), `.github/workflows/ci.yml` (short fuzz
  smoke job)
- Create: `FuzzParseStreamFrame` (`internal/mockgateway`), `FuzzWSDispatch`
  (`internal`), `FuzzErrorEnvelope`/`FuzzDecodeJSON` (`internal`/`pkg/ibkr`)
- Create/commit: `testdata/fuzz/` corpora or correct `docs/OBSERVABILITY.md`

**Approach:** Add a bounded `-fuzztime` smoke job (e.g. 30s per target, or
`-run=Fuzz` corpus replay) and the new targets.

**Acceptance:** CI runs fuzz smoke; new targets find no crash on seeds.

**Verification:** `make fuzz` / `go test -run=Fuzz ./...`

---

## Phase C Gate

`go test -race ./...` green; `internal/ws.go` covered; coverage floor raised;
new metrics/tracing/health documented and tested; fuzz runs in CI.
