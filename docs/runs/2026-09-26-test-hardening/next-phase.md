# Test Hardening - Next Phase

## What This Run Completed

This run started as a coverage-and-test-hardening exercise and turned into a
defect hunt. Step 0 re-measured coverage with the CI command and found the
inherited 37.5% figure came from a `coverage.out` two days older than the tests
that produced it; the real number was **43.3% against a 35% floor**, an 8.3-point
margin, so the "thin margin, one feature away from a red gate" urgency argument
was unfounded. Step 2 then found a real leak rather than the flake that had been
blamed on it: `mockgateway.(*Server).serveWS`
(`internal/mockgateway/stream.go:334`) parked on
`c.Read(context.Background())` at `internal/mockgateway/stream.go:348`, a context
that is never cancelled, and `httptest.Server.Close` does not track hijacked
connections, so the handler outlived its server. `StreamHub.closeAll`
(`internal/mockgateway/stream.go:191`) plus a new `Server.Close` fixed it
(`9b9d671`, v1.1.3). Having built real coverage on the REST surface, the tests
then surfaced seven production defects
that no existing assertion would have caught: `omitempty` silently dropped
`isCompleted: false` on a `PATCH`, making "mark not-completed" inexpressible;
`internal.Timeout`'s `defer cancel()` fired before the body read and defeated
`cancelOnCloseBody`, so every REST request dialled a fresh connection
(**24 calls / 24 connections before, 24 / 0 after**); `internal/session.go`
discarded the logout `*http.Response`, dropping a connection per `Client.Close()`;
`wrapOp` re-wrapped an already-typed `*Error` in a fresh one, so the 79-plus
`>= 400` guards produced `Code: ""` and `HTTPStatus: 0` at the outer level and
`errors.As(err, &e); e.HTTPStatus` from `docs/ERRORS.md` read zero;
`RESTRequestInfo.ExecutedAt` was never populated because the 200 body is a
`oneOf` that the code tried to decode as a single type; `instructionSetId` was
rendered with `strconv.FormatFloat(float64(id), 'f', -1, 32)`, so the spec's own
example `1988905739` came back as `1988905700` and callers polling by that ID
addressed the wrong instruction; and dead code was deleted - `WSConn.waitForDone`
plus five unused `*Raw` response types and their `toPublic` converters. Coverage
went **43.3% -> 58.7%** and the CI floor
(`.github/workflows/ci.yml:76`) ratcheted **35% -> 48% -> 58%** across three
releases: v1.1.3 (`f3e062d`), v1.1.4, v1.1.5.

## Open Items

### Uncovered surfaces, kept in the denominator by decision

| Package | Uncovered stmts | Status |
|---|---|---|
| `cmd/ibkr` | part of 318 | In `-coverpkg` (`ci.yml:69`) on purpose. `main()` (`cmd/ibkr/main.go:32`) is not worth covering; only the validation and parsing helpers are. |
| `cmd/ibkr-mock-gateway` | part of 318 | Same decision. |

The testable parts, for whoever picks this up:

| Symbol | Location | Why it is testable |
|---|---|---|
| `runOrdersSubmit` required-flag checks | `cmd/ibkr/orders.go:150-158` | Pure validation, returns before any client is built |
| `runOrdersSubmit` order-type / TIF defaults | `cmd/ibkr/orders.go:187-192` | Pure, no I/O |
| `parseGlobalFlags` | `cmd/ibkr/main.go:108` | Reads `os.Args` only |
| `mustAccount` | `cmd/ibkr/main.go:137` | Validates an account-ID string |
| `parseAccountFlag` | `cmd/ibkr/portfolio.go:43` | Reads `os.Args` only |
| `parseFields` | `cmd/ibkr/stream.go:96` | Pure string split into `ibkr.Field` |
| `truncate` | `cmd/ibkr/positions.go:69` | Pure |
| `configPath` / `loadConfig` / `saveConfig` | `cmd/ibkr/config.go:22`, `:34`, `:55` | Filesystem-only, temp-dir testable |

Caveat for the test author: `runOrdersSubmit` reads `os.Args[2:]` directly
(`cmd/ibkr/orders.go:82`), so the parse loop has to be either extracted into a
helper or exercised by swapping `os.Args`. Extracting is the smaller change and
makes the validation reachable without global mutation.

### Deferred defects

None of these were fixed; each carries the reason.

| # | Defect | Location | Why deferred |
|---|---|---|---|
| D10 | `CancelInstructionsBulk` never populates `Reason`. The loop sets only `InstructionId`, while the single `CancelInstruction` sets `Reason: req.Reason`. | `pkg/ibkr/rest_banking.go:390-394` vs `pkg/ibkr/rest_banking.go:364` | A wire-contract change to a money-moving bulk operation. Needs a payload-pinning test first so the fix is provable, and the spec has to be consulted for whether bulk-cancel even accepts a reason. |
| D11 | V2 asset-transfer quantity wire type diverges: the single path emits a JSON **number** (`strToDecimal` into `client.TradingInstrumentV2.Quantity float32`), the bulk path emits a JSON **string** (`Quantity string`). | `pkg/ibkr/rest_banking.go:750` + `client/client.gen.go:16505` vs `pkg/ibkr/rest_banking.go:802`, `:820` | Two public methods on the same manager send the same field in two JSON types. Fixing it means deciding which one the gateway accepts, and that needs a real account, not the mock. |
| D12 | `AssetTransferRequest.Quantity` is ignored by both V2 paths, which read `req.Positions[].Quantity` instead. The V1 paths do use it. | `pkg/ibkr/rest_banking.go:748-752` and `:816-821` (ignored) vs `:660` and `:714` (honoured) | A public field silently dropped on two of four methods. Whether that is a bug or a V1-only field is a contract question for the human, not a code-reading question. |
| D13 | `TradeConfirmations.ListAvailable` acquires an OAuth2 token and passes it as the `Authorization` param, but `internal.Auth` overwrites the header unconditionally. The round trip is wasted. | `pkg/ibkr/rest.go:415`, `:418` overwritten at `internal/transport.go:135` | Low harm but a real per-call cost. Deferred only because it needs a test that counts token acquisitions, which does not exist yet. |
| D14 | Tax-voucher money arrives as `float32` in the generated client, so the ADR 0008 `string` fields hold already-rounded values. `float32ToStr(16777217)` is `"16777216"`, and the test pins it. | `client/client.gen.go:16344-16353` (`DivAmount`, `Fee`, `Quantity`, `WithHeldAmount` all `*float32`) read at `pkg/ibkr/rest.go:769-774`, formatted by `pkg/ibkr/rest.go:896`; rounding pinned at `pkg/ibkr/rest_float32_test.go:46` | Needs a spec change (`type: string`) plus a `patch_spec.py` defect entry plus `make codegen`. The only item that touches ADR 0008's intent and the codegen pipeline at once. |
| D15 | `ListTaxDocumentsAvailable` always sends `year=`. The generated param is a non-pointer `string` with no omitempty, and the request builder styles it unconditionally; the wrapper never sets it. | `pkg/ibkr/rest.go:281-283`; `client/client.gen.go:24789` (`Year TaxYearRequestParam`, non-pointer); `client/client.gen.go:24303` (`TaxYearRequestParam = string`); `client/client.gen.go:40142` (styled unconditionally) | Same shape as D14: making `year` optional in the spec yields a `*string` and regeneration. |
| D16 | The mock gateway's shared `OpGetRequestsStatus` fixture uses `executedAt`, a key the spec does not define for this operation. The generated `StatusResponse` has `dateSubmitted`; the other `oneOf` variant `AmRequestStatusResponse` has no time field at all. The default body is therefore inert. | `internal/mockgateway/routes_rest.go:268`; spec shape at `client/client.gen.go:16203` and `:11056`; consequence asserted at `pkg/ibkr/rest_reports_e2e_test.go:83-85`, populated case must override the fixture at `:105` | Small, but it should not be "fixed" by a test-only override: the fixture is shared and other suites may depend on the current body. |
| D17 | `RESTRequestInfo.ExecutedAt` is a documented misnomer. It carries the submission time from `dateSubmitted`, not an execution time. | `pkg/ibkr/rest.go:193-202` (misnomer note at `:199-201`), populated at `:230-234` | Renaming an exported field needs a deprecation cycle: add the correct name, keep the old one as a deprecated alias, remove at the next major. |
| D18 | `TradeConfirmationRequest.Gzip` is a permanently inert exported field. The generated body model has no `gzip` property and the operation takes no gzip query param, so there is nothing to forward it into. | `pkg/ibkr/rest.go:390`, rationale at `:448`; `RESTStatements` is the gzip-capable alternative | Needs a major to drop, or a spec change to make it real. Neither is a patch-level decision. |
| D19 | Three package-private helpers have no production caller and are reachable only from tests: `strToDecimalPtr`, `makeTradingInstrumentRef`, `f32PtrToInt64Ptr`. | `pkg/ibkr/rest_banking.go:34`, `:1378`, `:1415`; only callers are `pkg/ibkr/rest_banking_e2e_test.go:1277`, `:1306`, `:1342` | Not deferred for a reason so much as not worth the risk budget: three functions and three tests, zero production effect. It is the cheapest item on this list. |

### Unverified against a real gateway

The mutating model endpoints remain unverified. `SubmitModelPortfolioOrder`
(`docs/SPEC.md:246`, `POST /v1/api/iserver/account/{modelCode}/orders`) cannot be
routed separately by the mock because it and `PlaceOrder` send byte-identical
payloads, so no body predicate can discriminate between them. That is a property
of the payloads, not a gap in the mock. Verifying dispatch needs an FA-enabled
paper account and an opt-in build tag.

### `check_design` breadth

`scripts/check_design/main.go` reads exactly two of the nine design documents:
`01-transport.md` (`:138`) and `03-managers.md` (`:301`). Unverified: `02-client.md`,
`04-generated-wrapping.md`, `05-streaming.md`, `06-errors-retries.md`,
`07-money-and-numbers.md`, `08-concurrency.md`, `09-orders-and-confirmation.md`.

Its existing checks are sound and the gap is breadth, not strictness. Presence
is checked at `:240`, order at `:243`, reverse drift (code has it, doc does not)
at `:265`, and full-order divergence at `:290`. A replacement-shaped edit - delete
one middleware, add another - fails the presence check on the new layer, which is
the claim an earlier run got wrong twice.

## Process Lessons Worth Carrying Forward

### The failure mode: inferring behaviour from counts

Three separate false conclusions in this run came from reading a count or a
pattern instead of reading the code. Each was caught only by an independent
check, and none would have survived into a release unreviewed.

| False conclusion | How it was made | What the code actually said | Caught by |
|---|---|---|---|
| "`rest.go`'s `>= 400` guards leak the response body" - first phrased as "66 `*WithResponse` calls, 4 `Body.Close()`", later as 62 unclosed bodies | grep counts, without checking whether the type was even closeable | The `*WithResponse` methods call `Parse*Response(rsp)` unconditionally, and that does `io.ReadAll` + `defer Body.Close()` regardless of status. `Body` is `[]byte`. The neighbouring `errorFrom` guards that close explicitly are closing an already-closed body. | A sub-agent **refused** the prescribed fix because `resp.Body.Close()` would not compile on `[]byte` |
| "Stray unbalanced paren at `rest.go` ~708" | reading a fragment | That `})` correctly closes the `&client.FetchDividends1Params{...}` composite literal | A build |
| "Coverage is 37.5% against a 35% floor, so the margin is thin" | carrying a number forward | The `coverage.out` was two days older than the tests that produced it. Fresh measurement: 43.3%. | Re-running the CI command instead of reading the file |

The same habit produced two file-scoped false negatives: comparing the
error-wrapping strings (`const op = "Restrictions.Account"`) against route
registrations said all 35 operations were unrouted, and checking fixtures only in
`routes_rest.go` said most had none. Both were wrong.

Working rule, taken from this: **treat every grep-derived resource or coverage
claim as a hypothesis until a failing test confirms it.** The count that
matters is the count a test reproduces.

### Method notes that worked

- **Quote comma-bearing flags in PowerShell.** `-coverpkg=a,b` arrives as two
  arguments and Go resolves the fragments as package patterns.
- **Deduplicate coverage profile blocks per test binary before trusting a
  total.** A `-coverprofile` run holds one block set per binary; naive summing
  reported 14.4% instead of 43.3%. Treat a block as covered if any binary
  covered it, and require the deduped total to reproduce `go tool cover -func`
  exactly before planning any work from it.
- **Check routes and fixtures in BOTH `routes_cpapi.go` and
  `routes_rest.go`.** Operations are keyed by constants like `OpGetAccountOwners`,
  not by the `op` string, and the fixture sets are split across the two files.
- **Read `operationPath` out of `client/*.gen.go` rather than guessing paths.**
  For example `/gw/api/v1/tax-documents/available` is at
  `client/client.gen.go:40115`, not where the endpoint index suggests.
- **Prove a test fails before the fix.** `go test ./internal/ -count=2` failed on
  every attempt pre-fix, which is what turned a "flake" into a located leak.
  Similarly, 24 sequential REST calls opened 24 connections before T22 and 0
  after - a number, not an assertion about the code.
- **Register LIFO-sensitive cleanups first.** `t.Cleanup` and `defer` are both
  LIFO, so a goleak assertion must be registered *first* in order to run *last*.
  Wiring it into `newTestClient` rather than `newGateway` made it run before the
  gateway shut down, and it duly reported `httptest`'s own accept loop as a leak
  in six tests.
- **A refused task is a signal.** The T22 sub-agent declined the fix it was
  given because the premise was false. That refusal was correct and pointed
  straight at the real defect.
- **Re-check line numbers in an artifact before citing them.** `report.md` and
  `todos.md` both record the parked WebSocket read at
  `internal/mockgateway/stream.go:328`. That was true when written, and the fix
  added `closeAll` and `Server.Close` above it, so the read now sits at
  `internal/mockgateway/stream.go:348`. A citation written before the fix and
  read after it is wrong in a way that is easy to miss, because the surrounding
  prose still matches.

### Delegation

Sub-agents produced reliable work on the task they were briefed on: the T14a
agent even mutation-tested four of its own assertions, which is the right
instinct. **Incidental findings need checking one at a time.** In one report, two
of three incidental claims were false (the response-body leak, and a claim that
`fault_injection_test.go` had to reach into `.Err` - it never touches `wrapOp`).
The lesson is not that sub-agents are unreliable; it is that the brief's
incidental observations need the same independent check as the orchestrator's
own, and that check is cheap.

## Candidate Next Phases

### P1 - Fix the deferred `rest_banking.go` request-payload defects (D10, D11, D12)

**Objective:** Make the single and bulk V2 asset-transfer paths send the same
payload shape, carry `Reason` through bulk cancel, and resolve what
`AssetTransferRequest.Quantity` means on a V2 path.
**Why now:** These are money-adjacent and each is a caller-visible contract
question that has been open since v1.1.5 shipped. The tests that will pin them
are the same coverage work the last slice added, so the fixtures already exist.
**Effort:** S
**Dependencies:** Payload-pinning characterisation tests first, so each fix has
a before and an after. D11 needs a spec reading to pick the JSON type.
**Risks:** D12 cannot be resolved by reading the code - it is a contract
question. Splitting it out is safer than bundling. Changing what goes on the wire
for a transfer instruction must not be done speculatively.

### P2 - Resolve the `float32` money precision in the tax-voucher DTOs (D14, D15)

**Objective:** Get tax-voucher money onto string types end to end, so ADR 0008
holds in substance and not just in shape, and stop sending a bare `year=`.
**Why now:** It is the only outstanding item that touches both an ADR's intent
and the codegen pipeline. It is also the item most likely to be quietly left
open, because the public API *looks* compliant.
**Effort:** M
**Dependencies:** A spec change in `scripts/patch_spec.py`, then `make codegen`.
`make codegen-verify` must pass before merge.
**Risks:** Regeneration is a large diff. If the published spec cannot express
these as strings, the alternative is a request/response editor, which is more
code and has to be tested against the real gateway to be trusted.

### P3 - Extend `check_design` to the seven unverified design documents

**Objective:** Give `check_design` a check per design document, one document at a
time, so drift is a build failure rather than a stale paragraph.
**Why now:** This run found seven production defects by reading code and writing
tests. That is a good use of a run but an expensive one. A checker over
`07-money-and-numbers.md` in particular would have surfaced D14 from a document
that already states the rule.
**Effort:** S
**Dependencies:** None.
**Risks:** Extending the checker will surface pre-existing drift that then has to
be fixed in the same change. Do one document per commit so the drift is
attributable.

### P4 - Cover `cmd/ibkr`'s order validation and flag handling

**Objective:** Get the validation and parsing helpers in `cmd/ibkr` off 0% without
pretending `main()` is testable.
**Why now:** It is the last identified block of meaningful uncovered statements,
and the floor cannot rise much further while 318 statements of `cmd/*` sit in the
denominator.
**Effort:** S
**Dependencies:** `runOrdersSubmit` reads `os.Args` directly
(`cmd/ibkr/orders.go:82`); either extract the parse loop or the tests must mutate
globals.
**Risks:** Low, but a test that only exercises `os.Args` juggling is brittle.

### P5 - Deprecate, then remove, the two misnomers (D17, D18)

**Objective:** Add `SubmittedAt` alongside `ExecutedAt` and deprecate the old
name; decide whether `TradeConfirmationRequest.Gzip` is retired or made real.
**Why now:** Both are documented as wrong, and a documented-wrong public field
is worse than an undocumented one because callers trust the doc.
**Effort:** M
**Dependencies:** Requires a deprecation cycle and therefore a major-version
boundary. Not patch-level work.
**Risks:** Removing an exported field in a minor release would break the
`docs/STABILITY.md` promise. Keep both names through at least one minor.

### P6 - Delete the three dead package-private helpers (D19)

**Objective:** Remove `strToDecimalPtr`, `makeTradingInstrumentRef` and
`f32PtrToInt64Ptr` with their three tests.
**Why now:** Zero production callers; the only references are the tests that
exist solely to reach them. It is the smallest item here and it removes 40-odd
statements from the coverage denominator, which is worth noticing.
**Effort:** S
**Dependencies:** None. Confirm no production caller by grep plus a build, the
same way D5 was cleared.
**Risks:** Negligible. They are unexported, so removal is not an API change.

### P7 - Fix the mock gateway's `OpGetRequestsStatus` fixture (D16)

**Objective:** Change the shared fixture body to a spec shape (`dateSubmitted`
rather than `executedAt`) so the default response exercises the decode path.
**Why now:** A shared fixture that no production code path can read is a silent
gap: every test that wants a timestamp has to override it, and the default-path
test asserts a *nil* result as if it were correct.
**Effort:** S
**Dependencies:** None. Invert the assertion at
`pkg/ibkr/rest_reports_e2e_test.go:83-85` to require a populated `ExecutedAt`.
**Risks:** The fixture is shared. Check for other consumers before changing the
body, and keep the `AmRequestStatusResponse` variant covered by an explicit
override, as `TestRESTRequests_Status_NoTimestampVariant` does today.

## Recommended Next Phase

**P1 + P6 + P7 + P3: fix the deferred banking payload contracts, clear the dead
code and the inert fixture, then widen `check_design`.**

The reasoning is that these four share one justification: every one of them was
found by reading code, and each would have been caught earlier by a test or a
checker. They are also all small, none requires a spec change or a
regeneration, and together they close every deferred item that is not blocked on
a human decision or a real gateway.

P2 is deliberately excluded even though D14 is a genuine precision bug. It needs
a spec change and a regeneration, it touches ADR 0008, and the right answer
depends on a question only the human can answer (below). Running it inside a
phase that also touches twelve other files would make the regeneration diff
harder to review than it needs to be. P4 and P5 are worth doing but neither is
urgent: P4 buys coverage points, P5 needs a major boundary.

## Draft Task Breakdown

Task IDs are `N*` and are independent of the run's `T*` numbering.

| ID | Task | Role | Depends on | Acceptance |
|----|------|------|------------|------------|
| N1 | Characterise the four `rest_banking.go` request payloads with byte-level tests | tester | - | Tests assert the exact JSON key set and value types for `CancelInstruction`, `CancelInstructionsBulk`, `TransferV2`, `TransferBulkV2`. Each test fails if a key is added, removed, or changes JSON type. Passing on current code. |
| N2 | Carry `Reason` through `CancelInstructionsBulk` | backend | N1 | `pkg/ibkr/rest_banking.go:390-394` populates `Reason`; single and bulk payloads agree; the N1 characterisation test is inverted to require it, not weakened. `gofmt`/`vet`/suite/`-race` green. |
| N3 | Make the V2 asset-transfer quantity wire type consistent | backend | N1, N2 | The chosen JSON type is justified by the spec's `FopInstructionV2.positions[].quantity`; `pkg/ibkr/rest_banking.go:750` and `:820` emit the same type; `make codegen-verify` clean if the spec changed. |
| N4 | Resolve `AssetTransferRequest.Quantity` on the V2 paths | architect | N1 | Either both V2 paths honour it or the field's doc states it is V1-only. A written decision, not silence. No silent drop at `pkg/ibkr/rest_banking.go:748-752` or `:816-821`. |
| N5 | Delete the three dead package-private helpers | backend | - | `strToDecimalPtr` (`rest_banking.go:34`), `makeTradingInstrumentRef` (`:1378`), `f32PtrToInt64Ptr` (`:1415`) and their three tests removed. Zero references confirmed by grep and a build. Suite green. |
| N6 | Correct the `OpGetRequestsStatus` mock fixture | backend | - | `internal/mockgateway/routes_rest.go:268` body uses `dateSubmitted`; `pkg/ibkr/rest_reports_e2e_test.go:83-85` inverted to require a populated `ExecutedAt`; `TestRESTRequests_Status_NoTimestampVariant` still covers the other `oneOf` arm; no other suite depended on the old key. |
| N7 | Drop the wasted `Token(ctx)` fetch in `TradeConfirmations.ListAvailable` | backend | - | `pkg/ibkr/rest.go:415` removed or justified; a test asserts the `Authorization` header still arrives (set by `internal/transport.go:135`) and that no extra token acquisition occurs. |
| N8 | Add a `check_design` check for `02-client.md` | backend | - | `scripts/check_design/main.go` reads the document; deliberately editing it fails `make check`; both presence and order are verified, matching the `01-transport.md` pattern. |
| N9 | Add `check_design` checks for `04`-`09`, one document per commit | backend | N8 | Each document has a check that fails on a deliberate edit. Drift fixed in the same commit as the check that found it, so each is attributable. |
| N10 | Cover `cmd/ibkr` order validation and flag handling | tester | - | `cmd/ibkr/orders.go:150-158` and `:187-192` covered; `main.go:108`, `main.go:137`, `portfolio.go:43`, `stream.go:96`, `positions.go:69`, `config.go:22`/`:34`/`:55` covered; `main()` still 0%; **no `ci.yml` edit in this task**. |
| N11 | Re-measure coverage and ratchet the floor | devops | N5, N6, N10 | Fresh CI-flag measurement with deduplicated profile blocks; floor in `.github/workflows/ci.yml:76` set to the measured value or below, never above; the buffer is recorded with its reason. |
| N12 | Close-out | docs | all | `report.md` final, plan actuals recorded, `docs/runs/index.md` row for this run set to complete, `make docs-check` green. |

## Open Questions for the Human

1. **Should the coverage floor keep ratcheting, and at what cadence?** It has
   moved 35% -> 48% -> 58% in three releases, and `cmd/ibkr` plus
   `cmd/ibkr-mock-gateway` contribute 318 uncovered statements to the
   denominator. Diminishing returns are arriving. Options: ratchet every
   release, every N releases, or only when a slice lands; or trim `-coverpkg` to
   `./pkg/ibkr,./internal` and track `cmd/*` separately.
2. **Is `float32` money precision in the tax-voucher DTOs acceptable?** The public
   fields are `string`, so ADR 0008 is satisfied in shape, but the value was
   rounded at JSON decode - `float32ToStr(16777217)` is `"16777216"`. Fixing it
   means a spec change plus regeneration. Is the rounding a real risk at IBKR's
   actual dividend magnitudes, or is documenting it sufficient?
3. **Should the mutating model endpoints get their own opt-in build tag, or stay
   permanently mock-only?** The mock cannot route `SubmitModelPortfolioOrder`
   separately because the payloads are byte-identical, so mock coverage of that
   path is structurally impossible.
4. **For D10 and D11, which is authoritative - the published spec or the live
   gateway?** `CancelInstructionsBulk` may not accept a reason, and the two V2
   quantity JSON types cannot both be right. Picking wrong here means either a
   rejected bulk cancel or a rejected asset transfer. If the answer requires an
   FA account, these should stay open rather than be decided from the spec alone.
5. **Is `AssetTransferRequest.Quantity` on a V2 path a bug or a deprecation?**
   Only someone who knows the callers can say whether real code passes both it
   and `Positions`.
6. **Should `check_design` enforce the design documents, or are they prose that
   the ADRs and the code own?** Extending it from 2 of 9 to 9 of 9 makes design
   drift a build failure. That is a policy choice about which artifact is
   canonical, and it is worth making deliberately rather than by default.
