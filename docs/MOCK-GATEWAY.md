# Mock gateway

`internal/mockgateway` is an in-repo mock of the IBKR gateway used by tests,
examples, and the standalone `cmd/ibkr-mock-gateway` binary. It adds no
dependencies: it is built on `net/http` and the `github.com/coder/websocket`
dependency the streaming client already uses. It serves both API surfaces on
one port, covers every operation in [SPEC.md](./SPEC.md) (193 operations: 123
CPAPI + 70 IB REST), records requests, injects scriptable faults, and mocks the
WebSocket stream.

It is a development and test aid, not a conformance suite. See
[Limitations](#limitations).

## Quick start

Run the standalone server:

```bash
go run ./cmd/ibkr-mock-gateway   # or: make mock-gateway
```

It listens on `:5001` by default and prints the exposed URLs plus the
environment variables the SDK reads:

```
ibkr-mock-gateway listening on http://localhost:5001 (seed=1, scenario=none, tls=false)
  CPAPI            http://localhost:5001/v1/api
  CPAPI WebSocket  ws://localhost:5001/v1/api/ws
  IB REST          http://localhost:5001/gw/api/v1 and http://localhost:5001/gw/api/v2
  OAuth2 token     http://localhost:5001/oauth2/api/v1/token
```

In another terminal, point an example or your own program at it:

```bash
export IBKR_GATEWAY_URL=http://localhost:5001
export IBKR_REST_GATEWAY_URL=http://localhost:5001
go run ./examples/mock
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:5001` | Listen address (`host:port`). |
| `-scenario` | `none` | Fault scenario: `none`, `expired` (401 on protected routes), `ratelimited` (429 on every route), or `flaky` (every 3rd request 503, every 7th dropped connection). |
| `-latency` | `0` | Fixed delay added to every response (for example `25ms`). Must not be negative. |
| `-seed` | `1` | Deterministic seed for generated values. |
| `-tls` | `false` | Serve TLS with an in-memory self-signed certificate for loopback. |
| `-verbose` | `false` | Log one line per request. |
| `-auth-required` | `false` | Reject protected CPAPI routes until `ssodh/init` establishes a session. |

With `-tls`, set `IBKR_INSECURE_SKIP_VERIFY=true` so the SDK accepts the
self-signed loopback certificate.

## Exposed URLs

All surfaces share the one listen port.

| Surface | Path | SDK configuration |
|---------|------|-------------------|
| CPAPI | `/v1/api/*` | `IBKR_GATEWAY_URL` / `ibkr.WithGatewayURL` |
| CPAPI WebSocket | `/v1/api/ws` | derived from `IBKR_GATEWAY_URL` |
| IB REST | `/gw/api/v1/*`, `/gw/api/v2/*` | `IBKR_REST_GATEWAY_URL` / `ibkr.WithRESTGateway` |
| OAuth2 token | `/oauth2/api/v1/token` | OAuth2 client options |

The OAuth2 token endpoint accepts the `client_credentials`, `refresh_token`,
and `private_key_jwt` flows (for `private_key_jwt`, the JWT assertion is checked
for shape only). IB REST routes require a bearer token issued by that endpoint;
CPAPI session routes are described in [SESSIONS.md](./SESSIONS.md).

## Using it from Go tests

The package lives under `internal/`, so it is importable from within this
module (`pkg/ibkr` tests, examples, and `cmd`) but is not part of the public
SDK API.

```go
import (
    "net/http/httptest"

    "github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

scn := mockgateway.NewScenario()
scn.Set(mockgateway.OpSubmitNewOrder, &mockgateway.Fault{
    Status: 500,
    Body:   `{"error":"internal error"}`,
})

srv := mockgateway.New(
    mockgateway.WithScenario(scn),
    mockgateway.WithSeed(42),
)
ts := httptest.NewServer(srv.Handler())
defer ts.Close()

// Point the SDK at ts.URL (WithGatewayURL / WithRESTGateway), then inspect.
req, _ := srv.Recorder().LastRequest()
_ = req
```

### Server options

| Option | Effect |
|--------|--------|
| `WithFixtures(*Fixtures)` | Use a custom fixture registry instead of `DefaultFixtures()`. |
| `WithRecorder(*Recorder)` | Use a custom recorder instead of a new one. |
| `WithScenario(*Scenario)` | Use a custom fault scenario. |
| `WithLatency(time.Duration)` | Add a fixed delay to every response; when a fault also sets a delay, the larger of the two applies. |
| `WithSeed(int64)` | Set the deterministic seed. |
| `WithAuthRequired(bool)` | Enforce the CPAPI session before protected routes. |
| `WithFaultPolicy(FaultPolicy)` | Install a per-request fault policy, consulted first. |
| `WithStreamScript(*StreamScript)` | Install the scripted WebSocket behavior. |

`Server` also exposes the pieces it built: `Handler()`, `Recorder()`,
`Fixtures()`, `Scenario()`, `Stream()`, `PushTick(Tick)`, `Seed()`, and
`Operations()` (the registered operation IDs).

### Recorder

`Recorder` captures a deep copy of every inbound request: `Requests()`,
`LastRequest()`, and `Reset()`. Each recorded `Request` carries the method,
path, query, headers, body, and receive time. Route parameters (`Params`) are
populated on the live request while it is routed, so fault policies and dynamic
fixtures can read them; recorded snapshots are taken before routing, so they do
not include `Params`.

### Scenario (faults)

A `Scenario` composes three fault sources, resolved in order: a policy callback
(`SetPolicy`), a per-operation fault (`Set`/`Clear`), then a global fault
(`SetGlobal`/`ClearGlobal`). Each `Fault` may set a `Status`/`Body`, add a
`Delay`, `DropConnection`, or block as a `Timeout`. Scenario also toggles
`SetSessionExpired`, `SetOrderReplyConfirm`/`SetOrderReplyBody`, and a global
`SetLatency`.

### Stream script (WebSocket)

`WithStreamScript` takes a `StreamScript`:

- `DropFirstConnection` closes the first connection after its first subscribe
  (exercises reconnect).
- `OnSubscribe` returns the `Tick`s to emit for a subscribe frame.
- `ErrorOnSubscribe` returns an error message instead of ticks.
- `Limits` (`MaxConIDsPerRequest`, `MaxSubscriptions`) enforces ceilings.
- `Hello` emits `sts`/`ntf` frames for protocol fidelity.

`Server.Stream()` exposes the hub, and `Server.PushTick` delegates to it:
`Push`/`PushTick` broadcast a tick to subscribers, `Broadcast` sends an
arbitrary frame, and `Subscribers()` reports the connected count. `StatusFrame`,
`NotificationFrame`, `UserFrame`, and `OrderFrame` build the standard frame
families.

The hub accepts both the JSON protocol used by `internal/ws.go`
(`{"method":"subscribe","params":{...}}`) and the raw CPAPI text forms
(`smd+265598+{...}`, `umd+265598`).

## Coverage guard

`internal/mockgateway/coverage_test.go` parses [SPEC.md](./SPEC.md), normalizes
dynamic path segments to `{}`, and fails if any CPAPI, IB REST, or combined
operation has no registered route, or if any route has no fixture. It asserts
the parsed counts match the canonical 123 CPAPI + 70 IB REST = 193 operations
(AGENTS.md rule 6), so the mock cannot silently drift from the spec.

`internal/mockgateway/shape_test.go` (`TestFixtureShapeConformance`) validates
that every fixture contains well-formed JSON and uses a strict decoder
(`json.Decoder` with `DisallowUnknownFields()`) to catch type mismatches. It
would have caught the nil-`interface{}` (D1) and wrong-decode-shape (D2) bug
classes. Dynamic, paginated, and mutation-ack fixtures are skip-listed by name.

## Example

[`examples/mock`](../examples/mock/main.go) starts no server itself; it connects
the SDK to an already-running gateway, initializes a session, and lists the mock
accounts:

```bash
go run ./cmd/ibkr-mock-gateway        # terminal 1
go run ./examples/mock                # terminal 2
```

## Limitations

- **Synthetic fixtures.** Bodies are valid JSON shaped to the generated models,
  not recorded from the real gateway. They are not a conformance suite and may
  differ from live payloads.
- **Shape-level auth.** Session and OAuth handling validates flow and encoding
  (including JWT well-formedness) but never verifies signatures.
- **No cryptographic conformance.** A passing run against the mock does not by
  itself prove interoperability with the real API; the paper-account
  integration tier still applies ([TESTING.md](./TESTING.md)).
- **Client-side issues are out of scope.** Driving every wrapper against the
  mock surfaced pre-existing generated-client and REST-wrapper defects; those
  are fixed in `scripts/patch_spec.py`/regeneration and the wrapper layer, not
  here.
