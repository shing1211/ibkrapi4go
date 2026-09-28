# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.9] - 2026-09-27

The lint gate now runs, and it found five real defects that nothing else was
watching for.

### Fixed

- **`SubmitDocument` discarded its `mimeType` argument.** The signature accepted a
  MIME type, defaulted it to `application/pdf`, and then built the multipart part
  with `CreateFormFile`, which hardcodes `application/octet-stream`. The caller's
  argument never reached the wire. The part is now constructed explicitly, with
  the filename escaped per RFC 7578.
- **The default response size limit was never applied.** `internal/transport.go`
  declared `defaultMaxResponseBytes` and applied the cap only when
  `MaxResponseBytes` was explicitly configured, leaving the constant unreferenced
  and every default-configured client with no limit at all on the response body it
  buffers. The cap is now always applied.
- **Exceeding that limit failed silently.** `maxBytesReader` returned a short read
  and its own documentation claimed the truncation was "detected by the caller and
  surfaced as a typed error" - no caller did that anywhere, so an oversized
  response decoded as corrupt JSON with no indication why. It now returns the new
  `internal.ErrResponseTooLarge`, and an exact fit still succeeds.
- **A panic in a subscription goroutine was swallowed.** The recover block in
  `pkg/ibkr/ws.go` was empty, under a comment reading "sink is already gone; log
  and exit"; it did neither. A panic there would have made a live subscription go
  quietly quiet. It now logs through the client logger and closes the
  subscription.
- **A WebSocket resilience test was not testing anything.** It compared errors with
  `==` against `ErrWSDisconnected`, but the WebSocket layer wraps its errors, so
  the comparison could never match and the test could not detect the disconnect it
  exists to detect. Now uses `errors.Is`.

### Changed

- **`.golangci.yml` was rejected by the linter and had never analysed any code.**
  It declared `version: "2"` but used the v1 schema, so `linters-settings` and the
  `issues.exclude-*` keys were all invalid. The job passed by never running. The
  config is migrated to the v2 schema, revive's `exported` rule is configured
  correctly (it had also been failing to set up and silently reporting nothing),
  and every remaining exclusion carries a comment explaining its reason. The gate
  now reports **0 issues** over a true finding count of 192.
- 32 exported symbols gained doc comments; 9 redundant conversions, 33 unchecked
  writes and 2 ignored `json.Unmarshal` calls were made explicit; 4 unused wire
  structs, 2 unused helpers and 1 unused test helper were deleted.
- `examples/models` uses `ModelsPager` instead of the deprecated `AllModels`.
- The mock gateway sets `tls.Config.MinVersion` explicitly instead of relying on
  the zero value.

### Added

- **Tests for `cmd/ibkr`, which had none.** `run`, `parseGlobalFlags` and
  `runCompletion` now take their arguments and output streams as parameters
  instead of reading `os.Args`, so the dispatch table, help, version, flag
  parsing and the generated shell completions are all reachable from a test.
- **A design-doc checker for `docs/design/07-money-and-numbers.md`**, bringing
  verified design documents from 8 of 9 to **9 of 9**. `check_money.py` proves no
  exported field is a float; this check additionally requires that the four
  `json.Number` money fields the document names are still `json.Number` in the
  generated client, so a spec edit cannot quietly put a float back.
- Regression tests for the response size cap: default applied without opt-in,
  overflow reported rather than truncated, exact fit accepted, and a 101 upgrade
  response left unwrapped so WebSocket dialing keeps working.
- Five red cases and a control for the new money checker.

### Fixed in the tooling itself

- The design-checker red tests asserted on error messages containing hard-coded
  line numbers, so adding a doc comment anywhere above a declaration broke them
  with a misleading "failed for the wrong reason". The comparison now strips line
  numbers from both sides.

## [1.1.8] - 2026-09-27

Five wire-contract and numeric-precision defects, plus two CI gates that existed
but were never enforced. Two of the defects had been carried as "needs live
gateway evidence" and turned out to be decidable from the committed spec.

### Fixed

- **Bulk instruction cancel silently dropped the caller's reason.**
  `CancelInstructionsBulk` built each instruction with only `InstructionId`, even
  though the generated `CancelInstruction` carries `Reason` and the single-cancel
  path sets it. A reason supplied to a money-moving operation was discarded before
  the request left the process.
- **Bulk external asset transfer V2 sent `quantity` as a JSON string.** The spec
  models `TradingInstrumentV2.Quantity` as a number, and the single-item path
  already emitted one, but the bulk path hand-rolled a local struct with
  `Quantity string` — so the bulk endpoint received a different type for the same
  field than the single endpoint did. The bulk path now converts with the same
  `strToDecimal` helper, and its triplicated inline position type is collapsed
  into one named type.
- **Tax voucher money was silently rounded.** `divAmount`, `withHeldAmount`,
  `fee` and `quantity` generated as `float32`, whose 24-bit mantissa rounds
  anything above 2^24 (16777216). An input of `12345678.91` was reported to the
  caller as `"12345679"`. These four fields are now `json.Number`, which accepts
  the JSON number the gateway actually sends and preserves its digits. The
  existing spec patch that retypes money to `string` cannot be used here: that
  would fail to decode, because Go cannot unmarshal a number into a string.
- **The available-tax-years call sent `year=`.** The spec marked the tax year as
  required on the operation that reports which years exist, so the generated
  field was non-pointer and the wrapper sent a present-and-empty parameter. The
  parameter is now optional for that one operation and is omitted. The shared
  component is untouched for operations that genuinely require a year.
- `AssetTransferRequest.Quantity` is now documented as the V1 single-instrument
  field. No behaviour change — the V2 paths were already correct to ignore it.

### Added

- `scripts/check_money.py` (the ADR 0008 guard) now runs in CI. It existed as a
  Make target but was never enforced, so a float money field could land in
  `pkg/ibkr` without failing the build.
- `make license-check` now runs in CI, so a missing SPDX header fails the build.

### Changed

- `patch_spec.py` gains two defects: one retyping always-numeric money fields to
  `json.Number`, one relaxing the tax year on the available-years operation.
- `docs/design/07-money-and-numbers.md` now records both reasons the SDK uses
  `json.Number` internally, including the float32-rounding case this release
  fixes.

## [1.1.7] - 2026-09-27

Docs and tooling only. No production code, no generated code, no dependency
change: the only file touched under `pkg/` is the `Version` constant.

### Changed

- **`check_design` is now a release gate.** It previously ran only when a
  maintainer typed `make docs-check`; neither CI nor the pre-PR `make check`
  invoked it, so drift in the design documents could not fail anything. The
  `docs` job gained a `setup-go` step and a `Check design docs against the code`
  step, and `make check` is now `fmt vet money-check design-check test`. For the
  first time, a design document that disagrees with the code stops the build.

- **The checker now verifies 8 of 9 design documents, up from 2.** New coverage
  for `02-client.md` (5 checks), `04-generated-wrapping.md` (2),
  `05-streaming.md` (3), `06-errors-retries.md` (6), `08-concurrency.md` (3) and
  `09-orders-and-confirmation.md` (6) - **25 checks in total**, each one observed
  failing when its claim is broken, so none of them is inert.
  `07-money-and-numbers.md` is deliberately excluded: `check_money.py` already
  enforces its main claim over the whole tree, and a second gate over the same
  fact would be one more thing to keep in step and no stronger.

- **The checker is backed by a mutation harness of 232 subtests** across five
  suites, including **two negative controls** that swap a strict branch for a
  naive one and require the specific cases to go red, so a future edit cannot
  quietly weaken a check. A third control breaks two documents at once and
  requires both messages from a single run, proving the harness is not inert.

### Fixed

- **Five false statements in design documents**, all found by the new checks and
  all resolved in favour of the code:

  - `09-orders-and-confirmation.md` documented `Reply.Message string` where the
    code declares `Messages []string` (`pkg/ibkr/trade.go`). A caller following
    the document got a **compile error**. Its `OrderRequest` block also listed
    `TimeInForce` before `StopPrice` and omitted `ParentID` and `IsSingleGroup`
    entirely.
  - `02-client.md` documented `WithOAuth2JWTKey(key []byte)` and
    `WithOAuth2JWTKeyPath(path string)`. Neither compiles: the real signature
    takes `*rsa.PrivateKey`, and the path form is `WithOAuth2JWTKeyFile`. The
    same block's completeness note claimed to omit two fields that the block
    lists directly above it.
  - `06-errors-retries.md` omitted `RetryPolicy.Metrics`, which the struct
    declares and which drives the retry backoff and retry metrics.

### Fixed (tooling hygiene)

- **Every `golangci-lint` finding in `scripts/` is resolved.** `golangci-lint
  run` over `./scripts/...` now reports **0 issues** uncapped; the run recorded 8
  before. Every `//nolint` carries a site-specific reason.

- **`scripts/changelog-gen.sh` and `scripts/sbom-gen.sh` now carry the SPDX
  header** that `AGENTS.md` requires of every source file.
  `addlicense -check scripts pkg internal cmd` exits 0.

### Known, deliberately unchanged

- The `lint & security` job **remains red**: `golangci-lint run` uncapped over
  `./pkg/... ./internal/... ./cmd/... ./scripts/...` reports **733 issues**, all
  of them in library code, and `.golangci.yml` still **fails**
  `golangci-lint config verify` - a v1 file declaring `version: 2`, so
  golangci-lint silently discards its `linters-settings` and `issues` blocks and
  the author's exclusions have never applied. This release is out of that
  scope; the state is documented in
  [`docs/runs/2026-09-27-design-checkers/report.md`](./docs/runs/2026-09-27-design-checkers/report.md)
  and is **not** fixed here. The three blocked wire-contract items from v1.1.6
  (bulk-cancel `Reason`, the V2 `quantity` number/string divergence, the ignored
  `AssetTransferRequest.Quantity`) also remain open.

## [1.1.6] - 2026-09-27

The mock gateway shipped a fixture-shape check that could not fail, and two
fixtures it should have caught. Both classes of defect are closed here, along
with the check itself.

### Fixed

- **The mock gateway was lying about the API contract in two places.**

  `createSsoSessions` sent `accessToken` and `tokenType`, but the generated
  `CreateSessionResponse` tags them `access_token` and `token_type`
  (`client/client.gen.go:11717`, `:11721`). Decoding is non-strict, so
  `CreateSessionResponse.AccessToken` was permanently `""` - the fixture looked
  like it returned a token, and the SDK never saw one.

  `getRequestsStatus` sent `executedAt`, a key belonging to a different
  operation. The real type carries `dateSubmitted`, so
  `RESTRequestInfo.ExecutedAt` was permanently nil on the default path. Its test
  asserted that nil as though it were correct; the assertion is now inverted to
  require the populated value, using a `-05:00` timestamp that crosses midnight
  so the UTC normalisation is genuinely exercised rather than copied.

- **A wasted round trip on every `TradeConfirmations.ListAvailable` call.** The
  method acquired an OAuth2 token and passed it as a request parameter, but the
  transport's `Auth` middleware overwrites the `Authorization` header at
  RoundTrip time, so the acquisition was discarded. Removed; the header still
  arrives, via the middleware, and the test now pins its exact value so a header
  carrying an empty token cannot satisfy it.

### Changed

- **The fixture shape check now works, and is stricter than a naive fix.**
  `internal/mockgateway/shape_test.go` set `DisallowUnknownFields` on a decoder
  whose target was `var js any`. `any` has no fields, so the option could never
  fire: the check only ever validated JSON well-formedness and could not detect a
  wrong-key fixture - the exact class of defect the two items above are
  instances of. It now resolves each operation's real response type,
  key-checks the body against it, and derives from `pkg/ibkr` call sites (via
  `go/ast`) which operations production actually decodes through the generated
  type at all. **113 of the 184 operations** `pkg/ibkr` reaches are not: they
  unmarshal `resp.Body` into their own structs, so for those the generated type
  is irrelevant and comparing against it was invalid. Final state: **180
  passed, 4 skipped, 0 failed** of 191 fixtures, with enforcement tighter than
  the naive version - key-checked against a single response type rose from 50 to
  59 operations, per-key `oneOf` checking from 1 to 2. Both historical bugs stay
  caught by `TestValidateShapeRejectsMismatchedKeys`. Coverage gaps are reported
  with the `file:line` that justifies each exemption, never silent.

- **Twelve further fixtures corrected to their generated shape**: five forecast
  operations (the generated types are objects, the fixtures were arrays), three
  OAuth token endpoints (`token` is declared by no field), two market-data and
  portfolio acknowledgements (`status` is not a declared key), `ackServerPrompt`
  (the type is a bare JSON string), and `cancelOpenOrder` (`order_id` is `int64`
  there, versus `string` in its sibling operations).

- **Dead code removed**: `strToDecimalPtr`, `makeTradingInstrumentRef` and
  `f32PtrToInt64Ptr` in `rest_banking.go` had no production caller - the only
  references were tests that existed solely to reach them. An orphaned test
  helper, `sortedKeys`, was removed with them. 383 insertions, 0 deletions
  elsewhere: no exported API changed.

### Testing

- **Four `rest_banking.go` request payloads are now pinned byte for byte** -
  exact key set and exact JSON type per key - so the deferred wire-contract
  decisions can be made with a provable before and after rather than a guess.
  Three remain **open and blocked** on a human decision or a real gateway, and
  are **not** fixed here: whether bulk-cancel accepts a `Reason`
  (`rest_banking.go` copies only `InstructionId`, so every element ships
  `reason: ""`), the V2 `quantity` number/string divergence (a JSON number on
  the single path, a JSON string on the bulk path, from the identical caller
  input `"10"`), and the ignored `AssetTransferRequest.Quantity` on both V2
  paths. The pinning tests say so in their own comments.

## [1.1.5] - 2026-09-27

### Fixed

- **`instructionSetId` was silently truncated on three banking
  acknowledgements.** `rest_banking.go` rendered the ID with
  `strconv.FormatFloat(float64(id), 'f', -1, 32)` on an `int` field. A 32-bit
  float keeps only 24 mantissa bits, so any ID above 2^24 (16,777,216) was
  rounded — the spec's own documented example, `1988905739`, came back as
  `1988905700`, off by 39. Callers polling or reconciling by that ID addressed
  the wrong instruction. Now uses `strconv.Itoa`, matching the three sibling
  sites in the same file that already did.

- **REST error detail was being discarded.** `wrapOp` unconditionally wrapped
  its argument in a new `*Error`, so the 79-plus `>= 400` guards that pass a
  typed `*Error` produced an outer error with `Code: ""` and `HTTPStatus: 0`,
  with the real values buried one level down in `.Err`. Callers following the
  documented `errors.As(err, &e); e.HTTPStatus` idiom in `docs/ERRORS.md` read
  zero. `wrapOp` now adopts an existing `*Error`, filling only a missing `Op`.

- **`RESTRequestInfo.ExecutedAt` was never populated** and the method had an
  `if` whose two branches were identical. The 200 body is a `oneOf` union; it is
  now decoded properly and `ExecutedAt` is filled from the `dateSubmitted`
  field, the only timestamp the spec provides for that operation. The field name
  is a known misnomer (it carries submission time) and is kept for API
  compatibility.

- **Dead code removed**: five unused response types and their `toPublic`
  converters in `rest.go` had no caller anywhere in the module.

### Changed

- **`TradeConfirmationRequest.Gzip` is documented as not sent**: the upstream
  body schema has no `gzip` property and the operation takes no gzip query
  parameter, so there is nothing to forward it into. The field is retained for
  API compatibility. `RESTStatements` does support gzip.

- **`ActiveCountries` documentation corrected**: it returns display names such
  as "United States", not ISO codes. Each upstream record also carries
  `countryCode`, which this method does not return.

### Added

- **Coverage is now 58.7%**, with the CI floor raised from 48% to **58%** — the
  third step of a ratchet that started at 35% in v1.1.3. The reports,
  tax-vouchers and REST-banking surfaces gain end-to-end tests in this release,
  following the accounts, utilities and notifications surfaces in v1.1.4; no
  production code changed to make them pass.

## [1.1.4] - 2026-09-27

### Fixed

- **REST connection reuse was completely broken: every request dialled a fresh
  connection.** `internal.Timeout` installed `defer cancel()` inside its
  `RoundTrip` closure, so the request context was cancelled the moment
  `RoundTrip` returned, before the caller read the response body. `net/http`
  reacts to an already-cancelled request context by closing the connection
  instead of returning it to the keep-alive pool, which also defeated
  `cancelOnCloseBody` — the wrapper whose documented purpose is exactly to defer
  that cancel until the body is consumed. Measured with 24 sequential REST
  calls: **24 new TCP connections before the fix, 0 after**. The cancel func is
  now handed to the body wrapper on success and called explicitly on the error
  paths, so no request timer leaks.

- **The logout response body was never drained or closed.**
  `internal.httpAPI.logout` discarded the `*http.Response` returned by
  `client.Do`, dropping one connection per `Client.Close()`. Draining to EOF
  before closing is required here: `net/http` still discards the connection on a
  bare `Close`, so a close without a drain would not have fixed it. The
  `defer cancel()` already in that function remains load-bearing and was left in
  place.

- **`UpdateTasks` silently dropped `isCompleted: false`.** The internal wire
  struct tagged a plain `bool` with `omitempty`, and `encoding/json` omits
  `false` for `omitempty`, so a request marking a task not-completed or declined
  serialised to `{"taskId":"t2"}` with the field absent. The call is a `PATCH`,
  where absence means "leave unchanged", which made that update inexpressible.
  The tag no longer carries `omitempty`. The exported `TaskUpdate.IsCompleted`
  type is unchanged, so this is not an API change.

### Added

- **REST coverage work continued.** The accounts surface gained end-to-end
  tests. CI coverage is now **50.0%** against a floor of **48%**, up from 35% in
  the previous release.

## [1.1.3] - 2026-09-26

### Fixed

- **The mock gateway leaked a goroutine per parked WebSocket handler.**
  `serveWS` blocked on `c.Read(context.Background())`, a context that is never
  cancelled, and `httptest.Server.Close` does not track hijacked connections, so
  a handler outlived the server that started it. This surfaced as an
  intermittent `TestWS_Resilience` failure that named no subtest: the parent
  failed because a handler from an earlier subtest was still alive when its leak
  check ran. `go test ./internal/ -count=2` reproduced it reliably.
  `StreamHub.closeAll` now closes every registered stream socket with
  `CloseNow`, which is what actually unblocks a read parked on an
  uncancellable context, and `Server.Close` exposes it.

- **`WSConn.waitForDone` was dead code** with zero callers; the live path inlines
  the same logic. Removed.

### Added

- **`waitForGoroutinesToSettle`** gives the WebSocket tests a bounded settle
  before their goroutine-leak assertion, and is tested in both directions to
  prove it reports a real leak rather than tolerating one.

### Changed

- **Four inaccurate claims in the run records are corrected.** The `ws-shutdown`
  report's `-race` note described an environment property rather than a project
  limitation. The `audit-remediation` run recorded coverage as 37.5% against a
  35% floor and called the margin thin; that figure came from a profile two days
  older than the tests that produced it, and a fresh measurement gives 43.3%, an
  8.3-point margin. Its claim that the coverage gate "cannot detect a
  replacement" was misattributed and false, since `check_design` verifies both
  presence and order. Its "goleak in 4 of 42 test files" note understated what
  exists, since both goroutine-owning packages already leak-check from `TestMain`.

## [1.1.2] - 2026-09-26

### Added

- **`TestModelOrderRouteCollision`** documents a known limitation of the mock
  gateway's path-based router: `submitNewOrder`
  (`POST /v1/api/iserver/account/{accountId}/orders`) and
  `submitModelPortfolioOrder` (`POST /v1/api/iserver/account/{modelCode}/orders`)
  are identical once placeholders are normalized, and the SDK sends a
  byte-identical payload for both, so the first-declared route always wins. A body
  predicate was implemented and then removed after the payloads were found to be
  identical. `OpSubmitModelPortfolioOrder` is now declared but deliberately left
  unrouted, since a registered route there could never be selected.

- **`TestModels_SubmitModelPortfolioOrder` now verifies the full response
  decode.** The test installs the broker's real snake_case response shape on the
  shared route for the duration of the call, exercising request build, the
  transport chain, the generated client, and decoding into the public type. It
  asserts `order_id`, `order_status`, the `id` reply field, and the `message`
  array. Previously it could only assert that one response entry came back.

### Changed

- `test/integration_test.go` records why the mutating model endpoints are
  excluded: the suite declares itself read-only, and those operations submit
  orders or move money. It also gains a read-only `AllocationModels` case that
  skips when the account lacks financial-advisor entitlement.

### Documentation

- Marks E3 complete in the `blueprint-hardening` todos and report, which had both
  said `review` after the work shipped in v1.1.1.
- Rewrites the `ws-shutdown` next-phase document: collapses the two duplicated
  "Recommended next phase" headings, folds the completed P2, P3, E3, and spec
  drift items into a single "Completed since this run was written" section, and
  marks P4 as not started.
- Adds `docs/runs/2026-09-26-audit-remediation/` with the full artifact set and
  an index row, so the v1.0.7 through v1.1.2 work has a home.

### Known limitations

- The mock gateway cannot route `SubmitModelPortfolioOrder` separately, and the
  integration suite stays read-only, so whether a real gateway dispatches that
  operation correctly is unverified. The response decode itself is covered.

## [1.1.1] - 2026-09-26

### Added

- **Multi-drop reconnect scripting in the mock gateway.** `StreamScript` gains
  `DropConnections int`, which closes the first N accepted connections after
  their first subscribe frame. `DropFirstConnection` still works and is
  equivalent to `DropConnections: 1`. `StreamHub.AcceptedConnections` exposes the
  running connection count so a test can assert how many reconnects happened.

- **Coverage for previously untested paths:**
  - late-dial discard, so a connection dialled as shutdown begins is force-closed
    and never published;
  - dispatch-level sequence-gap detection, asserting a lower `_updated` value
    surfaces exactly one `*WSGapError` while still delivering the tick;
  - `ForceRefresh` joining an in-flight fetch instead of starting a second one,
    and honouring context cancellation while waiting on it;
  - `RESTSurface.Invalidate` being safe and idempotent on a closed client;
  - the OAuth2 example's `isAuthError`, `handleAuthError`, and
    `httpStatusCheck` helpers.

### Fixed

- **Streamed `Update.Status` was unreachable.** Field `6509` was listed in
  `wsReservedField`, so `dispatch` dropped it before delivery and the
  `Subscription.Deliver` branch that populates `Update.Status` could never run.
  The status code is now delivered, so delayed, frozen, and not-subscribed states
  are visible on streamed updates as documented.

- **`check_design` never compared anything.** Its middleware-order extraction
  matched `*ast.Ident` against `cfg.Field` conditions, but those parse as
  `*ast.SelectorExpr`, and `cfg.Retry.enabled()` as a `*ast.CallExpr`, so the
  extracted order was always empty and the code-versus-document comparison was
  vacuous. Extraction now walks the `ms = append(ms, ...)` calls directly, fails
  loudly if it finds nothing, and diffs the assembled order against the
  documented chain. Reordering or adding middleware in the code without updating
  `docs/design/01-transport.md` now fails the check.

- **`make codegen` and `make codegen-verify` failed on Windows.** Both scripts
  embedded the `mktemp -d` path into the oapi-codegen config, but that path is an
  MSYS path which the native Windows binary cannot resolve. They now convert it
  with `cygpath -m`, which yields a forward-slash Windows path that needs no YAML
  escaping.

- **`make docs-spec` crashed on Windows.** `gen_spec_index.py` read the spec
  without an explicit encoding, and Windows defaults to cp1252, which cannot
  decode the spec's non-ASCII characters. Both the read and the write now pin
  UTF-8, matching the earlier `patch_spec.py` fix.

### Changed

- `docs/design/01-transport.md` now lists the `maxBytes` layer, which the code
  has always assembled but the diagram omitted.
- Corrected inaccurate claims in `docs/TESTING.md` (fixture location, goleak
  scope, `TestMain` behaviour, race invocation, codegen scheduling) and
  `docs/STREAMING.md` (channel closure on error, per-channel overflow policy,
  what the overflow test actually asserts).
- Corrected `CHANGELOG.md` history: 53 fuzz functions rather than 47, the shape
  test is a structural check rather than 180 per-operation comparisons, the
  migration guide is 219 lines, and fuzz corpora are not stored under
  `testdata/`.
- Dropped the pre-alpha wording from the Japanese, Korean, and Spanish READMEs;
  the project is stable and unofficial, matching the English README.
- Corrected the README lifecycle and transport-chain diagrams to match the code,
  including that `Client.Close` closes the WebSocket before session release.

## [1.1.0] - 2026-09-26

### Added

- **Eight model-portfolio and allocation operations are now exposed.** The v2.40
  spec carries 193 operations but only 185 had public wrappers. Adds
  `ModelManager.IsFullMaster`, `ModelCashAnalyzer`, `RebalanceToExistingTargets`,
  `RebalanceToNewTargets`, `RebalanceToSpecificTargets`, `TwsInvestDivest`,
  `SubmitModelPortfolioOrder`, and `AllocationManager.AllocationModels`, with
  matching mock gateway routes and fixtures. `docs/SPEC.md` is regenerated to
  193 operations and 451 schemas, and the canonical coverage counts move from
  115/70/185 to 123/70/193.

  Money and quantity values in the new public types are strings, per ADR 0008.
  `RebalanceToNewTargets`, `RebalanceToSpecificTargets`, `TwsInvestDivest`, and
  `SubmitModelPortfolioOrder` take hand-marshalled request bodies because their
  generated request types embed anonymous structs that cannot be named from
  another package.

  `SubmitModelPortfolioOrder` has no dedicated mock route: its path template
  normalizes to the same method and path as the Phase-1 `submitNewOrder` route,
  and the mock router is first-match, so that route always wins. The operation
  still satisfies the SPEC coverage check, which compares normalized
  method and path. Its response decode is therefore not exercised by the mock
  and needs a real gateway to verify.

### Fixed

- **`make docs-spec` crashed on Windows.** `gen_spec_index.py` read the spec
  without an explicit encoding, and Windows defaults to cp1252, which cannot
  decode the spec's non-ASCII characters. Both the read and the write now pin
  UTF-8, matching the earlier `patch_spec.py` fix.

## [1.0.8] - 2026-09-25

### Fixed

- **Reconnect notification ordering.** `reconnect` called `notifyReconnect`
  before `resubscribeAll`, so a consumer reacting to `ErrWSReconnected` could
  read its `Updates` channel before the subscribe frames had been issued and
  lose the first updates after a reconnect. Resubscribing now precedes the
  notification, matching the order already documented in
  [docs/STREAMING.md](./docs/STREAMING.md). Adds a regression test that fails
  under the previous ordering.

### Changed

- Added `.gitattributes` enforcing `* text=auto eol=lf`. Without it, a Windows
  checkout can leave CRLF in the working tree, which makes a local
  `gofmt -s -l .` and `make check` report files that are correctly stored as LF
  in git. This is the same class of problem that previously shipped CRLF in
  `client/client.gen.go` and broke the codegen drift check.
- Aligned `.github/workflows/pre-commit.yml` with `ci.yml`: the gofmt
  exclusion now matches Windows path separators, and `actions/setup-go` moves
  from the pinned `@v5` to `@v7` used by every other workflow.
- Documented the golangci-lint, gofmt-pattern, and Windows line-ending CI
  fixes that shipped inside the `v1.0.7` range without a changelog entry.

## [1.0.7] - 2026-09-25

### Fixed

- **golangci-lint was installed from a non-existent module path.** The lint job
  ran `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`;
  the v2 module moved that command to `.../golangci-lint/v2/cmd/golangci-lint`,
  so the install could not succeed. CI steps also gained an explicit
  `shell: bash` default, and the release workflow was restructured.
- **`gofmt` exclusion pattern was POSIX-only.** The format check excluded the
  generated client with `grep -v '^client/'`, which does not match the
  backslash-separated paths `gofmt` emits on Windows. It now uses
  `grep -vE '(^|[\\/])client[\\/]'`, matching `ci.yml`.
- **Windows checkout line endings.** CI now sets `core.autocrlf false` and
  re-checks out the tree on Windows so formatting checks see consistent
  endings. This treats the symptom; the underlying cause is addressed by the
  `.gitattributes` added in `[Unreleased]`.
- **Coverage gate never enforced anything.** `.github/workflows/ci.yml` parsed
  the total with `awk -F'[.%]' '{print $3}'`, which extracts an empty field from
  `total: ... 36.8%`. The subsequent `[ "$TOTAL" -lt 60 ]` then errored rather
  than comparing, so the step passed unconditionally. The parser now reads the
  last field and strips `%`, and the threshold uses a float comparison instead of
  `[ -lt ]` (which rejects decimals). Threshold set to 35%, the current measured
  coverage, as a ratchet baseline.
- **`codegen drift` CI job failing on every run.** The committed
  `client/client.gen.go` blob used CRLF line endings and the repository has no
  `.gitattributes`. `validate_codegen.sh` diffs the committed file against a
  fresh generation, and on Linux the generator emits LF, so the check failed even
  though the generated code was otherwise identical. Regenerated with LF; the
  drift check now passes.
- **`make codegen` / `make codegen-verify` crashed on Windows.**
  `patch_spec.py` wrote the patched spec to `sys.stdout`, which defaults to
  cp1252 on Windows and cannot encode the spec's non-ASCII characters. stdout is
  now reconfigured to UTF-8.

### Changed

- **`patch_spec.py` defect 8:** money and quantity fields declared inline under
  `paths` are now retyped from `number` to `string` (ADR 0008). Defect 7 only
  walked `components.schemas`, so eight fields across three operations still
  generated as `float32`. The new allowlist is deliberately separate from defect
  7 so that live banking request schemas (`FopInstruction`, `DwacInstruction`,
  `ComplexAssetTransferInstruction`, `singleOrderSubmissionRequest`) keep their
  existing wire format. See `docs/CODEGEN.md`.
- Regenerated `client/client.gen.go` from the patched v2.40.0 spec. All six
  money/quantity fields on `SubmitModelPortfolioOrderJSONBody` and both
  `AmtToInvest` fields are now `*string`.

### Documentation

- `docs/CODEGEN.md` — documented all eight spec defects, added a line-endings
  section explaining the drift failure, and corrected the measured line count
  (72,532 → 75,688).

## [1.0.6] - 2026-09-24

### Added

- Added explicit `RESTSurface.ForceRefresh` and `RESTSurface.Invalidate`
  lifecycle controls while preserving automatic token refresh.
- Hardened OAuth single-flight result delivery and prevented invalidated
  in-flight results from repopulating the access-token cache.
- Updated the OAuth2 live example to avoid printing token material.

### Fixed

- Made the optional Gitee CI mirror authenticate with `GITEE_TOKEN` and skip
  cleanly when the secret is not configured.

## [1.0.5] - 2026-09-24

### Fixed

- Fixed the release workflow's Gitee mirror step to push a detached tag
  checkout as `HEAD:main`.

## [1.0.4] - 2026-09-24

### Fixed

- Fixed the WebSocket shutdown hang caused by the D4 sequence-gap change: an
  uninitialized `lastUpdated` map could panic while holding `WSConn.mu`.
- Made explicit WebSocket close cancel active I/O and close the socket without
  waiting indefinitely for a graceful handshake.
- Prevented reconnect from publishing a connection after shutdown began.
- Reserved market-data field `6509` consistently.

### Changed

- Added regression coverage for sequence tracking, silent peers, active
  subscriptions, and repeated WebSocket close/reconnect runs.
- Corrected the prior run report's diagnosis of the Windows failure.
- Made `money-check` struct-aware so generated code, unexported adapters, and
  function bodies no longer create false positives; the gate now passes.
- Reconciled current release/spec references and removed stale error aliases
  from the user-facing documentation.

## [1.0.3] - 2026-09-24

### Added

- **Phase B CI hardening:** golangci-lint + gosec, single release workflow with
  GoReleaser, coverage threshold 60%, OS×Go test matrix, DCO enforcement,
  dependency-review, TruffleHog secret scanner, CycloneDX SBOM, GoDoc exported
  enforcement.
- **Phase C reliability/observability:** `internal/ws.go` unit tests, ws
  resilience tests + fuzz targets, injectable `Clock` + `DialWSFunc`, missing
  metrics, composite `Health()` probe, OTel tracing bridge, fuzz targets in CI.
- **Order state machine + duplicate-submission protection:** `OrderState` with
  legal transitions and a per-`TradeManager` `ClientOrderID` registry that
  rejects duplicate submissions (`pkg/ibkr/orderstate.go`).
- **Bracket / OCA / multi-leg orders:** `OrderRequest.ParentID` and
  `IsSingleGroup` with builder helpers, plus `TimeInForce` `FOK`/`GTD` values and
  `OrderRequest.Validate` (`pkg/ibkr/ids.go`, `pkg/ibkr/trade.go`,
  `pkg/ibkr/builders.go`).
- **Typed WebSocket events:** `OrderEvent`, `NotificationEvent`, and
  `UserMessageEvent` are parsed and delivered on `Subscription.SystemUpdates()`
  (`pkg/ibkr/events.go`).
- **Account/portfolio streaming:** `AccountManager.SubscribeAccount` and
  `PortfolioManager.SubscribePortfolio` deliver typed `AccountUpdateEvent`
  (`acq`) and `PortfolioEvent` (`pos`) values on dedicated channels
  (`pkg/ibkr/ws.go`). See [docs/STREAMING.md](./docs/STREAMING.md).
- **Delayed-data / permission surfacing:** `MarketDataStatus` (field `6509`) is
  exposed on `Snapshot.Status` and `Update.Status`
  (`pkg/ibkr/marketdata.go`, `pkg/ibkr/ws.go`).
- **`ClientOrderID` round-trip:** populated on `Order` and `OrderStatus` from
  broker responses (`pkg/ibkr/trade.go`).
- **WebSocket gap detection:** `WSGapError` is emitted when the `_updated`
  sequence jumps backwards (`internal/ws.go`).
- **Docs:** `docs/GATEWAY-SETUP.md` and `docs/PERMISSIONS.md`; README
  architecture flow diagram; cancellation and reconciliation examples
  (`examples/mock/cancel-order`, `examples/mock/reconcile-open-orders`).

### Fixed

- **Misleading live examples:** removed the unused `IBKR_USERNAME` /
  `IBKR_PASSWORD` requirement (the gateway is browser-authenticated), made
  `options-chain` use SDK managers, and corrected the OAuth2 "force refresh"
  example.

### Removed

- **`RESTSSOSessions.CreateSessionRaw`:** removed because it leaked the generated
  `*client.CreateSsoSessionsResponse` through a public signature
  (`docs/design/04-generated-wrapping.md`). Use `CreateSession`.

## [1.0.1] - 2026-09-21

### Added

- **WS system frame routing:** `Subscription.SystemUpdates()` channel exposes
  `sts`, `ntf`, `sor`, `usr` frames separately from market data. Existing
  `Updates()` behavior is unchanged. See `pkg/ibkr/ws.go`.

### Fixed

- **`decodeJSON` consistency:** All 7 `json.Unmarshal` calls in
  `pkg/ibkr/rest_accounts.go` now use `decodeJSONBytes` (`UseNumber` mode),
  preserving decimal precision per ADR 0008.
- **`patch_spec.py` defect 5:** ConID and banking ID fields (`conid`,
  `clientInstructionId`, `instructionId`, `instructionSetId`, `ibReferenceId`)
  now generate as `int64` instead of `float32` — eliminates silent precision
  loss for IDs exceeding 2^24.
- **`patch_spec.py` defect 6:** `twsInvestDivestResponse` schema renamed to
  `TwsInvestDivestResponseData` to avoid collision with the auto-generated
  HTTP response wrapper type (v2.40.0 spec).
- **`patch_spec.py` defect 7:** 16 money amount fields (SMA, Balance,
  BuyingPower, NetLiquidationValue, etc.) now generate as `string` instead
  of `float64` per ADR 0008.
- **`portfolio.go`/`rest_banking.go`:** Updated calls to match regenerated
  client signatures (Params structs, int64 ID types).

### Changed

- **Spec updated to v2.40.0:** `specs/ibkr_spec.json` refreshed; all 7 defects
  applied before codegen.
- **`client/client.gen.go` regenerated:** ConID as `int64`, banking IDs as
  `int64`, money fields as `string`.

## [1.0.0] - 2026-09-21

This release marks the first stable API surface. All public symbols in `pkg/ibkr`
are now covered by a stability contract (ADR 0015). The API is production-ready
for Go 1.26+.

### Breaking changes from v0.x

This release is not fully API-compatible with v0.x. A migration guide and
automated codemod are provided.

- **57 methods** renamed: the `Get` prefix is removed per Go naming conventions.
  Run `scripts/codemod.sh` to automate the mechanical renames. See
  `docs/MIGRATION.md` for the full table and manual steps.
- **`float32` → `int64`** for banking IDs (`ClientInstructionID`,
  `InstructionID`, `IbReferenceID`).
- **14 initialism casing fixes** (e.g. `EchoHTTPSResponse` → `EchoHTTPSResponse`,
  `RealizedPnL` → `RealizedPnL`). Automated via `scripts/codemod.sh`.
- **`ErrStreamDisconnected` / `ErrStreamReconnected`** removed; use
  `ErrWSDisconnected` / `ErrWSReconnected`.

### Stability Hardening

- **Goroutine safety (S1):** All 6 async goroutine paths now recover from panics.
  `TestGoleak` integration covers every goroutine-creating function. ` goleak`
  is a test-only dependency; no runtime cost.
- **Backpressure (S2):** Context deadline propagation through all WS/tickle/token
  paths. New `WithRequestTimeout` option sets a hard timeout per operation.
- **Production resilience (S3):** `WithEndpointTimeout` and
  `WithCircuitBreakerBudget` options. Error budget tracking in `internal/breaker.go`
  with `BreakerMetrics` export.
- **Fault injection (S4):** 36 fault-injection tests across `internal/` and
  `pkg/ibkr/` covering transport errors, timeout, circuit breaker, and order
  submission paths.
- **Performance baseline (S5):** HTTP/2 connection pooling in `internal/transport.go`.
  Allocation benchmarks in `pkg/ibkr/alloc_test.go`. Baseline stored in
  `benchmark.baseline`; CI compares every run.

### Ecosystem

- **CLI tool (E1):** `cmd/ibkr/` binary with `accounts`, `positions`, `orders`,
  `stream`, `portfolio`, `config`, and `completion` subcommands.
- **Migration guide (E2):** `docs/MIGRATION.md` and `scripts/codemod.sh`
  (74 rename rules) covering all v0.x → v1.0 breaking changes.
- **API reference site (E3):** `docs/api.html`, `docs/architecture.html`,
  `docs/decisions.html` with CSS/JS assets.
- **Release automation (E4):** `.github/workflows/release-automation.yml` with
  semver enforcement, `.github/workflows/supply-chain.yml` with govulncheck and
  SBOM generation. Gitee auto-push on release.
- **Supply-chain security (E5):** `go mod verify`, `govulncheck`, secret scanning,
  SLSA-level provenance generation.

### Architectural

- **Interface segregation (A1):** Five interfaces in `internal/interfaces.go`:
  `TokenProvider`, `RoundTripper`, `SessionMachine`, `WSClient`, `RateLimiter`.
  `internal/fake/` package with five fake implementations for testing.
- **Typed builders (A2):** `OrderBuilder`, `ContractBuilder`,
  `TransferInstructionBuilder` in `pkg/ibkr/builders.go`.
- **Middleware plugin (A3):** `WithTransportMiddleware` option in
  `pkg/ibkr/middleware.go`. Example in `examples/middleware/`.
- **Unified pagers (A4):** Generic `*Pager[T]` in `pkg/ibkr/pager.go`. Four
  concrete pagers: `ModelsPager`, `FYIsPager`, `TransactionsPager`,
  `SubaccountsPager`. Four old slice-returning methods deprecated.
- **Multi-client transport (A5):** `TransportPool` in `pkg/ibkr/transport_pool.go`
  with shared `*http.Client`, session, and rate limiter across clients.

### OpenTelemetry Support

- **OTel metrics bridge (P2):** `contrib/otel/` module ships a first-class
  `OTelMetrics` implementation of `ibkr.Metrics` bridging to OpenTelemetry
  counters, histograms, and gauges. Instrument creation is lazy and cached.
  Import `github.com/shing1211/ibkrapi4go/contrib/otel` only when needed;
  the core SDK carries no OTel dependency (ADR 0004).

### Examples

- **3 new live examples** in `examples/` require paper trading credentials
  (`IBKR_GATEWAY`, `IBKR_USERNAME`, `IBKR_PASSWORD`). All operations are
  read-only.
  - `live-portfolio/`: Session.Initialize, Account.List, Portfolio
    Positions/Ledger/Summary
  - `options-chain/`: symbol search + options strike lookup for a given month
  - `screener/`: ScannerParameters discovery + live market scanner
- **Mock examples** (`mock/`, `portfolio/`, `orders/`, `marketdata-streaming/`,
  `models/`, `middleware/`) target the in-repo mock gateway and need no
  credentials.

### Deprecated

- `ScannerManager.ScannerResults` — use `Scanner().ScannerResults` via the
  `ScannerManager` returned by `Client.Scanner()`
- `PortfolioManager.Subaccounts` — use `SubaccountsPager` instead
- `FYIManager.FYIs` — use `FYIsPager` instead
- `PerformanceManager.Transactions` — use `TransactionsPager` instead
- `ModelManager.Models` — use `ModelsPager` instead
- `DefaultServerURL` — use `DefaultGatewayURL` instead

## [0.3.0] - 2026-09-18

### Breaking

This release contains breaking API changes. All symbols are frozen for the v0.x
series; breaking changes will not occur after v1.0.0.

#### `Get` prefix removed from 57 methods

Go convention is to omit the `Get` prefix when the receiver already provides
context. All affected methods are on manager types.

| Manager | Old name | New name |
|---------|----------|----------|
| `ScannerManager` | `GetScannerParameters` | `ScannerParameters` |
| `ScannerManager` | `GetScannerResults` | `ScannerResults` |
| `TradingAccountManager` | `GetAccountOwners` | `AccountOwners` |
| `TradingAccountManager` | `GetDynamicAccounts` | `DynamicAccounts` |
| `TradingAccountManager` | `GetFundSummary` | `FundSummary` |
| `TradingAccountManager` | `GetBalanceSummary` | `BalanceSummary` |
| `TradingAccountManager` | `GetMarginSummary` | `MarginSummary` |
| `TradingAccountManager` | `GetAccountMarketSummary` | `AccountMarketSummary` |
| `TradingAccountManager` | `GetBrokerageAccounts` | `BrokerageAccounts` |
| `ModelManager` | `GetModelPresets` | `ModelPresets` |
| `ModelManager` | `GetAccountsInModel` | `AccountsInModel` |
| `ModelManager` | `GetInvestedAccountsInModel` | `InvestedAccountsInModel` |
| `ModelManager` | `GetAllModels` | `AllModels` |
| `ModelManager` | `GetAllModelPositions` | `AllModelPositions` |
| `ModelManager` | `GetModelSummarySingle` | `ModelSummarySingle` |
| `SessionManager` | `GetSessionValidation` | `SessionValidation` |
| `SessionManager` | `GetSessionToken` | `SessionToken` |
| `WatchlistManager` | `GetSpecificWatchlist` | `SpecificWatchlist` |
| `WatchlistManager` | `GetAllWatchlists` | `AllWatchlists` |
| `AllocationManager` | `GetAllocatableSubaccounts` | `AllocatableSubaccounts` |
| `AllocationManager` | `GetAllocationGroups` | `AllocationGroups` |
| `AllocationManager` | `GetSingleAllocationGroup` | `SingleAllocationGroup` |
| `AllocationManager` | `GetAllocationPresets` | `AllocationPresets` |
| `FYIManager` | `GetFYIDelivery` | `FYIDelivery` |
| `FYIManager` | `GetFYIDisclaimers` | `FYIDisclaimers` |
| `FYIManager` | `GetAllFYIs` | `AllFYIs` |
| `FYIManager` | `GetFYISettings` | `FYISettings` |
| `FYIManager` | `GetUnreadFYIs` | `UnreadFYIs` |
| `TradeManager` | `GetTradingSchedule` | `TradingSchedule` |
| `TradeManager` | `GetAlgosByInstrument` | `AlgosByInstrument` |
| `TradeManager` | `GetInfoAndRules` | `InfoAndRules` |
| `TradeManager` | `GetCurrencyPairs` | `CurrencyPairs` |
| `TradeManager` | `GetExchangeRates` | `ExchangeRates` |
| `TradeManager` | `GetBondFilters` | `BondFilters` |
| `TradeManager` | `GetContractInfo` | `SecDefInfos` |
| `TradeManager` | `GetContractSymbolsFromBody` | `ContractSymbolsFromBody` |
| `TradeManager` | `GetConidsByExchange` | `ConidsByExchange` |
| `TradeManager` | `GetFutureBySymbol` | `FutureBySymbol` |
| `TradeManager` | `GetInstrumentDefinition` | `InstrumentDefinition` |
| `TradeManager` | `GetTradingScheduleBySymbol` | `TradingScheduleBySymbol` |
| `TradeManager` | `GetStockBySymbol` | `StockBySymbol` |
| `PerformanceManager` | `GetPerformanceAllPeriods` | `PerformanceAllPeriods` |
| `PerformanceManager` | `GetSinglePerformancePeriod` | `SinglePerformancePeriod` |
| `PerformanceManager` | `GetTransactions` | `Transactions` |
| `AlertManager` | `GetAlertDetails` | `AlertDetail` |
| `AlertManager` | `GetMtaDetails` | `MtaDetail` |
| `AlertManager` | `GetAllAlerts` | `AllAlerts` |
| `ForecastManager` | `GetForecastCategories` | `ForecastCategories` |
| `ForecastManager` | `GetForecastContract` | `ForecastContract` |
| `ForecastManager` | `GetForecastMarkets` | `ForecastMarkets` |
| `ForecastManager` | `GetForecastRules` | `ForecastRules` |
| `ForecastManager` | `GetForecastSchedule` | `ForecastSchedule` |
| `PortfolioManager` | `GetAllAccountsForConid` | `AllAccountsForConid` |
| `PortfolioManager` | `GetManySubaccounts` | `ManySubaccounts` |
| `PortfolioManager` | `GetComboPositions` | `ComboPositions` |
| `PortfolioManager` | `GetUncachedPositions` | `UncachedPositions` |
| `RESTRequests` | `GetStatus` | `Status` |

#### Type consistency: `AccountID` and `ConID`

- `AccountID` (type `string`) is now used consistently for all account-ID fields
  across 21 struct fields in response types. Callers passing raw `string`
  account IDs may need to wrap with `AccountID(...)`.
- `ConID` (type `int`) is now used consistently for `ConID` fields in banking
  transfer request types (`AssetTransferRequest`, `PositionV2Request`,
  `InternalAssetTransferRequest`).

#### `float32` → `int64` for banking IDs

`BankInstructionCreateRequest.ClientInstructionID` and `TransferResult`
fields (`ClientInstructionID`, `InstructionID`, `IbReferenceID`) are now
`int64` instead of `float32`. These are integer IDs, not floating-point values.

#### Go initialism casing fixed

14 symbols renamed to follow Go convention for initialisms:
`EchoHTTPSResponse`, `ListEchoHTTPS`, `SignedJWTEchoRequest`,
`SignedJWTEchoResponse`, `CreateEchoSignedJWT`, `SSOBrowserSessionRequest`,
`SSOSessionRequest`, `CSVApplyResponse`, `CSVVerifyRequest`,
`CSVVerifyResponse`, `RealizedPnL`, `UnrealizedPnL`,
`RequestAccessToken`, `RequestLiveSessionToken`, `RequestTempToken`.

### Deprecated

- `ErrStreamDisconnected` and `ErrStreamReconnected` are removed. Use
  `ErrWSDisconnected` and `ErrWSReconnected` instead.

### Internal

- `RESTInstructions` type removed (zero methods, dead code).

## [0.2.0] - 2026-09-18

### Added

- **Three new ADRs:** ADR 0015 (public API surface and stability contract),
  ADR 0016 (error handling philosophy), ADR 0017 (logging interface).
- **Godoc sprint:** 65 previously undocumented exported symbols across 7 files
  now have godoc comments, including multi-line docs with usage examples for
  complex types (`RESTClientInstruction`, `RESTTransaction`, `Form`, `Bank`,
  `CashBalanceDetail`, `ListRequestsFilter`, `RESTRequestSummary`,
  `RESTAccountSummary`, `RESTAccountStatus`, `RegistrationTaskItem`,
  `TaxVoucherDividend`, `TaxVoucherState`).
- **4 new examples:** `portfolio`, `marketdata-streaming`, `orders`, `models`
  with updated `examples/README.md`.
- **Integration test scaffold:** `test/integration_test.go` with
  `//go:build integration` gate and env-gated skip; `make test-integration`
  target.
- **Shape conformance guard:** `TestFixtureShapeConformance` in
  `internal/mockgateway/shape_test.go`, which walks every default fixture and
  validates the first JSON value parses. It is a structural check, not a
  per-operation shape comparison.
- **Spec drift detection:** `.github/workflows/spec-drift.yml` (daily cron +
  workflow_dispatch) with `scripts/check_spec_version.py`.
- **`docs/STABILITY.md`:** user-facing stability contract derived from ADR 0015.

### Changed

- **ADR 0008 compliance (banking types):** `AssetTransferRequest`,
  `CashTransferRequest`, `InternalAssetTransferRequest`,
  `InternalCashTransferRequest`, and `PositionV2Request` field types changed
  from `float32` to `string` for `ClientInstructionID`, `Quantity`, `Amount`,
  and `TransferQuantity`. Call sites use `strToF32`/`strPtrToF32Ptr` helpers
  at the generated-client boundary.

### Deprecated

- `ErrStreamDisconnected` — use `ErrWSDisconnected` instead (removed in v0.3.0).
- `ErrStreamReconnected` — use `ErrWSReconnected` instead (removed in v0.3.0).

### Fixed

- Remaining ADR 0008 violations in `rest_banking.go` call sites (pos.Quantity,
  req.TransferPrice, InternalCashTransferInstruction.ClientInstructionID).

### Internal

- 4 wrapper types unexported: `LoginMessagesWrapper` →
  `loginMessagesWrapper`, `AccountStatusBulkWrapper` →
  `accountStatusBulkWrapper`, `Au10TixWrapper` → `au10tixWrapper`,
  `RegistrationTasksWrapper` → `registrationTasksWrapper`.
- 8 stale `FIX:` comments removed from REST wrappers.

## [0.1.1] - 2026-09-17

### Fixed

- **Codegen root cause for nil-`interface{}` panics.** `scripts/patch_spec.py`
  now applies a fourth spec patch (defect 4): inline query parameters that omit
  `type` (`type: null`) are retyped to `type: string`. `oapi-codegen` therefore
  emits a concrete `string` (or named string enum) instead of a bare
  `interface{}` for the affected 22 parameters, eliminating the nil-panic class
  at the source. The generated `client/client.gen.go` no longer contains the
  post-generation nil guards, and `scripts/patch_gen.py` is now a documented
  no-op kept for backward compatibility.
- Updated SDK callers for the retyped parameters: `TradeManager`
  `GetTradingScheduleBySymbol` casts `assetClass` to the generated enum type,
  and `FYIManager.ModifyFYIEmails` serializes the `enabled` flag with
  `strconv.FormatBool`.

## [0.1.0] - 2026-09-17

### Added

- **Complete public SDK (`pkg/ibkr`)** covering both API surfaces — **185/185
  operations**: 115 on the Client Portal API (`ssoBearer`) and 70 on the IB REST
  API (`oauth2Bearer`). Every manager is registered on `Client` with an accessor.
  - **CPAPI managers:** `SessionManager` (`Session`), `AccountManager`
    (`Account`), `PortfolioManager` (`Portfolio`), `TradeManager` (`Trade`,
    including contracts), `MarketDataManager` (`MarketData`: snapshot, history,
    and WebSocket `Subscribe`), `TradingAccountManager` (`TradingAccount`),
    `AlertManager` (`Alert`), `ForecastManager` (`Forecast`, event contracts),
    `ScannerManager` (`Scanner`), `AllocationManager` (`Allocation`),
    `ModelManager` (`Model`), `FYIManager` (`FYI`, notifications),
    `OAuthManager` (`OAuth`, OAuth1), `WatchlistManager` (`Watchlist`), and
    `PerformanceManager` (`Performance`, PortfolioAnalyst).
  - **IB REST surface (`Client.REST()`):** `RESTSurface` with `RESTAccounts`,
    `RESTBanking` (banking and transfers), `RESTRequests`, `RESTUtilities`,
    `RESTRestrictions`, `RESTBalances`, `RESTTaxVouchers`, `RESTStatements`,
    `RESTTaxDocuments`, `RESTTradeConfirmations`, `RESTSSOSessions`, and
    `RESTEcho`.
- **Internal subsystems (`internal/`):** transport middleware chain, session
  state machine with tickle/heartbeat, WebSocket connection management and
  reconnect, OAuth2 token source (`fetchWithSecret`, `fetchWithJWTAssertion`),
  per-endpoint and global rate limiting, circuit breaker, retry with
  `Retry-After` handling, and observability (structured logging, redaction,
  telemetry hooks).
- **JWT / OAuth2 infrastructure:** `internal/jwt.go` (RS256 `private_key_jwt`
  assertions with PKCS8/PKCS1 PEM parsing) and `internal/oauth.go` (access and
  refresh tokens, single-flight refresh, refresh-token rotation).
- **Unified logging framework:** structured `log/slog` output with
  `ibkr.<subsystem>` message prefixes, per-component nil-safety (a logger is
  never nil after construction), and redaction of tokens, cookies, and
  `Authorization` headers. See [docs/LOGGING.md](./docs/LOGGING.md).
- **Dependency-free metrics layer:** an OTel-shaped `Metrics` interface with
  `InMemoryMetrics`/`NopMetrics`, `WithMetrics`, and documented `ibkr.*` metric
  names for HTTP, orders, rate limiting, circuit breaking, WebSocket, and OAuth.
  See [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) and
  [ADR 0013](./docs/adr/0013-metrics.md).
- **In-repo mock IBKR gateway:** `internal/mockgateway` serves both API surfaces
  and the WebSocket stream (185/185 operations) with scriptable faults, request
  recording, deterministic fixtures, and shape-level auth, with no new
  dependency; `cmd/ibkr-mock-gateway`, `examples/mock`, and `make mock-gateway`
  drive it. See [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) and
  [ADR 0014](./docs/adr/0014-mock-gateway.md). Manager and WebSocket tests now
  run against it. The mock is a development/testing aid, not a conformance
  suite.
- **Benchmarks (`pkg/ibkr/benchmark_test.go`, `internal/benchmark_test.go`):**
  `testing.B` benchmarks for JSON encode/decode across 20 public response types,
  HTTP round-trip latency (mock gateway), WebSocket subscribe/unsubscribe, and
  session init. Baseline stored in `benchmark.baseline`. See
  [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md).
- **Fuzz tests (`pkg/ibkr/fuzz_test.go`, `internal/fuzz_test.go`):** 53 `testing.F`
  fuzz functions covering all major public JSON decode types, plus response decode
  fuzzing across all op response shapes using mock gateway fixtures. No
  panics found. Fuzz corpora are generated on demand by `make fuzz` into the Go
  build cache, not committed under `testdata/`.
- **Benchmark CI gate (`.github/workflows/ci.yml`):** `benchmarks` job compares
  current results against `benchmark.baseline`; fails on >10% regression in
  ns/op. `scripts/bench_compare.go` is pure stdlib.
- **Coverage badge (`.github/workflows/ci.yml`):** `go test -coverprofile=coverage.out`
  uploaded to codecov.io via `codecov/codecov-action@v4`; badge rendered in all
  6 README translations.
- **Pre-commit CI gate (`.github/workflows/pre-commit.yml`):** `gofmt -s -l .`,
  `go vet ./...`, and `make check` run on every push and PR; parallel to the main
  CI workflow (~30s).
- **FUNDING.yml (`.github/FUNDING.yml`):** GitHub Sponsors link for `shing1211`.
- **Docs website (`docs/index.html`):** branded GitHub Pages landing page served from
  the `docs/` folder at `https://shing1211.github.io/ibkrapi4go/`; pure HTML/CSS,
  zero new dependencies, zero build step. Enable at `Settings → Pages → Source:
  main branch, /docs folder`.
- Initialized the Go module (`go.mod`) and committed the generated OpenAPI
  client (`client/client.gen.go`, package `client`), with a deterministic SPDX
  header and a reproducible `make codegen-verify` check.
- Project documentation set: `SPEC.md`, `ARCHITECTURE.md`, `AUTH.md`,
  `SESSIONS.md`, `ERRORS.md`, `RATE-LIMITING.md`, `STREAMING.md`, `LOGGING.md`,
  `OBSERVABILITY.md`, `CODEGEN.md`, `TESTING.md`, `MOCK-GATEWAY.md`,
  `RELEASING.md`, `CONFIG.md`, `GLOSSARY.md`, `ROADMAP.md`.
- Architecture Decision Records (`docs/adr/0001`–`0014`).
- Per-module design contracts (`docs/design/01`–`09`).
- `scripts/patch_spec.py`: generalized OpenAPI spec patcher (path-parameter
  reconciliation, operation-ID de-duplication, Go type-name collision fixes).
- `scripts/validate_codegen.sh`: reproducible codegen validation. OpenAPI
  codegen against IBKR Web API v2.39.0 succeeds after three classes of spec
  patches; the generated client is ~72k LOC and compiles cleanly. See
  [docs/CODEGEN.md](./docs/CODEGEN.md).
- `scripts/patch_gen.py`: deterministic post-generation fixups. It wraps
  unguarded `runtime.StyleParamWithOptions` calls for bare-`interface{}` query
  parameters in a nil guard, and is invoked by `scripts/codegen.sh` and
  `scripts/validate_codegen.sh` so a fresh generation matches the committed
  output byte-for-byte.
- Full Apache License 2.0 text, `NOTICE`, `THIRD_PARTY_NOTICES.md`,
  `DISCLAIMER.md`.
- Code of Conduct, Security Policy, Support and Governance documents.
- README translations: 简体中文 (`zh-Hans`), 繁體中文 (`zh-Hant`), 日本語 (`ja`),
  한국어 (`ko`), Español (`es`), plus `TRANSLATING.md` and a CI consistency
  check (`scripts/check_i18n.py`). `README.zh-CN.md` is now a redirect stub.

### Changed

- README and `docs/` set updated to describe 185/185 coverage across both API
  surfaces, replacing pre-implementation "planned" language.

### Fixed

- Removed the stale claim that the public SDK (`pkg/ibkr`) and `internal/`
  packages are "not yet implemented"; both are complete.
- Corrected the documented dependency set: `testify` is not used;
  `go.uber.org/goleak` is the test-only dependency.
- Fixed nil-interface{} panics in the generated client (`client/client.gen.go`):
  16 nil guards added to request builders for `GetContractInfo` (6 fields),
  `GetAllFyis` (3 fields), `GetAssetAllocation` (1 field),
  `GetPaginatedPositions` (3 fields), `GetConidsByExchange` (1 field),
  `GetTradingSchedule2` (1 field), and `ModifyFyiEmails` (1 field).
  Root cause: spec-patch produces `interface{}` with `omitempty` for optional
  non-pointer params; the codegen template did not guard against nil. The guards
  now live in `scripts/patch_gen.py`, so `make codegen` reproduces them and
  `make codegen-verify` passes.
- Fixed 8 REST wrapper/decode mismatches in `pkg/ibkr/rest.go` and
  `pkg/ibkr/rest_utilities.go`: `TaxDocuments.Generate`,
  `TaxVouchers.CreateRequests`, `ActiveCountries`, `AvailableYears`,
  `Dividends`, `Utilities.Enumerations`, `ComplexAssetTransferBrokers`,
  and `RequiredForms`.

[Unreleased]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.9...HEAD
[1.1.9]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.8...v1.1.9
[1.1.8]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.7...v1.1.8
[1.1.7]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.6...v1.1.7
[1.1.6]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.5...v1.1.6
[1.1.5]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.4...v1.1.5
[1.1.4]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.2...v1.1.3
[1.1.2]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/shing1211/ibkrapi4go/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.8...v1.1.0
[1.0.8]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/shing1211/ibkrapi4go/compare/v1.0.0...v1.0.1
[0.3.0]: https://github.com/shing1211/ibkrapi4go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/shing1211/ibkrapi4go/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/shing1211/ibkrapi4go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/shing1211/ibkrapi4go/releases/tag/v0.1.0
