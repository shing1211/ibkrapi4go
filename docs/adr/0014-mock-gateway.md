# 0014 — In-repo mock IBKR gateway

- Status: Accepted
- Date: 2026-09-17

## Context

Exercising the SDK end to end needs an IBKR gateway that answers every
operation it implements. The real options are a browser-authenticated, locally
running Client Portal Gateway, or hosted IB REST credentials. Neither belongs in
CI, and neither is deterministic.

Before this decision, the tests carried ad-hoc `httptest` gateways
(`pkg/ibkr/endtoend_test.go`, `pkg/ibkr/ws_test.go`) that covered only the
handful of routes a given test touched. That duplicated routing, fixtures, and
WebSocket plumbing, and left the rest of the surface unverified.

The mock must:

- serve **both** API surfaces — the CPAPI (`ssoBearer`) and IB REST
  (`oauth2Bearer`) models of [ADR 0001](./0001-two-api-surfaces.md) and
  [ADR 0002](./0002-auth-models.md) — including the OAuth2 token endpoint of
  [ADR 0011](./0011-oauth2-surface.md);
- cover **every** operation in [SPEC.md](../SPEC.md), not just the ones a test
  happens to use;
- be scriptable (faults, latency, deterministic values) and observable
  (request recording, streaming frames);
- be reusable from tests, examples, and a standalone binary;
- add **no** dependency ([ADR 0004](./0004-minimal-dependencies.md)) and expose
  **no** new public SDK surface.

## Decision

- Add the package `internal/mockgateway`, built on `net/http` plus the existing
  `github.com/coder/websocket` dependency already used for the streaming client.
  It imports no `testing` package, so `cmd` and examples can use it unchanged.
- Serve both surfaces from a single `http.Handler` on one port: CPAPI
  `/v1/api/*`, the CPAPI WebSocket at `/v1/api/ws`, IB REST at
  `/gw/api/v1/*` and `/gw/api/v2/*`, and the OAuth2 token endpoint at
  `/oauth2/api/v1/token`.
- Register a route and a fixture for **every** operation in
  [SPEC.md](../SPEC.md) — currently 185 operations (115 CPAPI + 70 IB REST),
  which remains the canonical count (AGENTS.md rule 6). A package test parses
  `docs/SPEC.md`, normalizes dynamic path segments to `{}`, and fails if any
  operation lacks a route or a fixture.
- Make faults scriptable through a `Scenario`: a per-operation fault map, a
  global fault, and a policy callback, each able to set a status/body, add
  latency, drop the connection, or block as a timeout. A `Recorder` captures
  deep copies of inbound requests; session and OAuth state are in-memory.
- Mock the WebSocket stream with a `StreamHub`: subscribe/unsubscribe,
  scripted ticks (`StreamScript`), explicit `PushTick`/`Broadcast`, enforced
  subscription limits, and `sts`/`ntf`/`usr`/`sor` frame helpers.
- Keep fixtures **synthetic**. They are valid JSON with money and quantities as
  strings ([ADR 0008](./0008-numeric-precision.md)), but they are a test aid,
  not a conformance suite: they neither redistribute the IBKR spec nor assert
  schema fidelity against it.
- Validate auth at the **shape/flow** level only: a session cookie or
  `Authorization` token for CPAPI, an opaque bearer/refresh pair for IB REST,
  and a JWT assertion checked for well-formedness (three segments, `alg`, and an
  `iss`/`sub` claim). Signatures are never verified.
- Locate the implementation under `internal/` so it is not part of the public
  API, and ship a standalone `cmd/ibkr-mock-gateway` binary that serves the
  surfaces with scenario, latency, seed, TLS, logging, and auth-enforcement
  flags.

## Consequences

- Manager and WebSocket tests share one gateway and exercise the whole surface;
  `examples/mock` and the `make mock-gateway` target use the same code path.
- Session enforcement is **opt-in** (`WithAuthRequired`): by default protected
  routes answer fixtures so a test can focus on the operation under test.
- The coverage guard binds the mock to `docs/SPEC.md`. Adding or removing spec
  operations requires route/fixture updates, and the guard fails loudly if they
  drift.
- The mock is not the real gateway. Synthetic bodies can differ from live
  payloads, cryptographic auth is not exercised, and a paper-account integration
  tier is still required for release confidence.
- Known client-side issues surfaced by driving every wrapper against the mock
  (generated-client nil-parameter panics and REST wrapper decode mismatches) are
  **out of scope** here; they belong to `scripts/patch_spec.py`/regeneration and
  the wrapper layer, not to the mock.
- No new dependency and no public surface expansion.

## Alternatives considered

1. **Public `mockgateway/` package** — reusable by SDK consumers, but it
   expands the public API and its maintenance burden, contrary to the minimalism
   of [ADR 0004](./0004-minimal-dependencies.md).
2. **Test-only helpers in `pkg/ibkr/*_test.go`** — smallest change, but they
   cannot back a `cmd` binary or examples, and they cannot be shared across
   packages.
3. **A third-party mock server** — less code to own, but it adds a dependency
   and cannot be tailored to the SDK's fixtures and generated-client shapes.
