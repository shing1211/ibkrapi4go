# Next Phase — Planning (post mock-gateway run)

- **Run:** `2026-09-17-mock-gateway` (task T9, planner)
- **Base commit:** `f9cf1fc` (`feat(mockgateway): add in-repo 185-op mock gateway, CLI, examples, docs`)
- **Status of this document:** proposal only — no code, docs, or artifacts changed.

## 1. What was completed this run

This run delivered the in-repo mock IBKR gateway and exposed the SDK to every
operation it implements. `internal/mockgateway` serves both API surfaces on one
`http.Handler` — CPAPI `/v1/api/*` plus the WebSocket at `/v1/api/ws`, IB REST
`/gw/api/v1|v2`, and the OAuth2 token endpoint `/oauth2/api/v1/token` — with a
route and fixture registered for all 185 operations (115 CPAPI + 70 IB REST) and
a coverage guard (`coverage_test.go`) that parses `docs/SPEC.md` and fails on
drift. It adds scriptable faults (`Scenario`/`FaultPolicy`), request recording
(`Recorder`), a scripted WebSocket hub (`StreamHub`/`StreamScript`), and
shape-level session/OAuth handling, with no new dependency. The standalone
`cmd/ibkr-mock-gateway` binary, `examples/mock`, and `make mock-gateway` drive
the same code path, and `pkg/ibkr/endtoend_test.go` and `pkg/ibkr/ws_test.go`
were migrated onto it. ADR 0014 and `docs/MOCK-GATEWAY.md` record the decision
and usage; ROADMAP Phase 8, TESTING.md, and CHANGELOG `[Unreleased]` were
updated. All planned tasks T1–T8 (plus T7b/T7c) are `done`.

## 2. Gaps, tech debt, and deferred items

- **D1 — generated-client nil-`interface{}` panic (correctness bug).**
  `TradeManager.GetConidsByExchange` (`pkg/ibkr/contract.go:584`),
  `TradeManager.GetContractInfo` (`pkg/ibkr/contract.go:524`), and
  `FYIManager.GetAllFYIs` (`pkg/ibkr/notifications.go:173`) build generated
  `*Params` with optional `interface{}` fields left nil (`AssetClass`,
  `Sectype`, `Month`, `Exchange`, `Strike`, `Right`, `Filters`, `Include`, …).
  The generated builders pass those nil values to
  `runtime.StyleParamWithOptions`, which does `reflect.TypeOf(value).Kind()` and
  **panics on a nil interface** (`runtime@v1.7.0/styleparam.go:83-95`; example
  call site `client/client.gen.go:44409`). The wrappers therefore panic before
  any HTTP call. Per AGENTS.md rule 1 the fix belongs in
  `scripts/patch_spec.py` + regeneration, not in `client/*.gen.go`. Blocked
  locally: `oapi-codegen` is not installed (`make tools`), while CI installs
  `oapi-codegen@v2.8.0` (`.github/workflows/ci.yml`).
- **D2 — REST wrapper/model decode mismatches (8 operations).** Driving every
  REST wrapper against the mock surfaced pre-existing decode defects:
  `RESTTaxDocuments.Generate` (double-reads an already-consumed/closed
  `HTTPResponse.Body` — `pkg/ibkr/rest.go:300`, while
  `ParseCreateTaxDocumentsResponse` consumed and closed it at
  `client/client.gen.go:66023-66032`) and `RESTUtilities.Enumerations`,
  `RESTUtilities.ComplexAssetTransferBrokers`, `RESTUtilities.RequiredForms`,
  `RESTTaxVouchers.CreateRequests`, `RESTTaxVouchers.ActiveCountries`,
  `RESTTaxVouchers.AvailableYears`, `RESTTaxVouchers.Dividends`. The wrappers
  decode narrow shapes (`[]string`, bespoke `*Raw` structs) whereas the mock
  fixtures use the spec/generated-model shape — e.g. `OpGetEnumerations` returns
  `{"enumerationsType":...,"jsonData":null}` but `Enumerations` decodes
  `[]string`; `OpGetActiveCountryList`/`OpFetchDividends1`/`OpGetYears` return
  arrays but the wrappers decode `*Raw` objects. These are **wrapper fixes, not
  codegen**.
- **Run close-out is incomplete.** `docs/runs/2026-09-17-full-api-coverage/`
  contains only `plan.md` — no `todos.md` and no `report.md` — and
  `docs/runs/index.md` does not exist. The mock-gateway run's own T10
  (`report.md` + index line) is still `todo`.
- **No fixture-shape guard.** The mock's coverage guard asserts route *presence*
  and fixture existence, not that a fixture matches the generated response
  model. This is precisely how D1/D2-class drift went unnoticed; a shape check
  would have caught them.
- **Integration tier is still a stub.** `docs/TESTING.md` reserves `test/` with
  an `integration` build tag, but `test/` does not exist and no paper-account
  verification runs. The mock is explicitly not a conformance suite.
- **Stale context in the task brief / docs.** The brief asserted "No CI pipeline
  yet (no `.github/workflows/`)"; in fact `.github/workflows/` exists with
  `ci.yml` (fmt, vet, `go test -race`, docs guards, `codegen-verify`),
  `codeql.yml`, `govulncheck.yml`, and `release.yml`. The real backlog is a
  **coverage badge/upload, pre-commit hooks, `FUNDING.yml`, GitHub Discussions,
  and a docs website** (ROADMAP "Backlog"). Confirmed absent: `.pre-commit-config.yaml`,
  `.github/FUNDING.yml`, `test/`, any `Benchmark*`/`Fuzz*` functions, and any
  coverage badge reference.
- **Performance/robustness coverage is absent.** No benchmarks and no fuzz
  tests anywhere in the tree, despite a hot path (spec parsing/normalization,
  transport, rate limiting, decode) worth protecting.

## 3. Candidate next-phase items

### C1 — Fix D1: nil optional `interface{}` params in generated client
- **Objective:** stop the three wrappers from panicking when optional params are
  omitted, by teaching `scripts/patch_spec.py`/codegen to skip nil optionals and
  regenerating `client/*.gen.go`; add regression tests.
- **Why now:** these are shipped, documented operations that panic on their
  normal call path — a correctness bug, not a papercut.
- **Effort:** M (toolchain + spec-patch + regeneration + tests).
- **Dependencies:** `make tools` (`oapi-codegen@v2.8.0`); spec fetch; AGENTS.md
  rule 1 and `make codegen-verify`.
- **Risks:** patch fragility across spec refresh; large generated diff; must not
  hand-edit generated code; upstream spec may not mark the fields optional.

### C2 — Fix D2: REST wrapper/model decode mismatches (8 ops)
- **Objective:** make the eight REST wrappers decode the spec/generated-model
  shape (array vs object, field names, single vs double body read), with
  regression tests driving each wrapper against `internal/mockgateway`.
- **Why now:** bounded, well-enumerated defects; every REST consumer of these
  ops currently gets an error or wrong data.
- **Effort:** M.
- **Dependencies:** mock gateway is already in place; generated models for shape
  reference.
- **Risks:** per-op ownership decision (fixture vs wrapper) needs the spec as
  tie-breaker; possible upstream ambiguity in response shape.

### C3 — Fixture shape-conformance guard in the mock
- **Objective:** add a mock test that validates every fixture against its
  generated response model/shape (or a spec-derived JSON shape), so an op whose
  wire shape drifts fails loudly.
- **Why now:** directly prevents recurrence of D1/D2 and raises the mock from
  "route present" to "shape plausible" — without new runtime deps.
- **Effort:** L (introspection of generated types or a generated shape table).
- **Dependencies:** C1 + C2 (fixtures/types must first be correct); ADR 0004
  (no new dependency).
- **Risks:** generated types may be loose (e.g. `interface{}`), so "shape"
  coverage may be partial; scope creep into a full conformance suite.

### C4 — Close out run artifacts and create `docs/runs/index.md`
- **Objective:** backfill `todos.md`/`report.md` for
  `2026-09-17-full-api-coverage`, write the mock-gateway `report.md` (T10), and
  create `docs/runs/index.md` linking both runs; wire it into `docs-check`.
- **Why now:** traceability is broken today; it is cheap and blocks nothing.
- **Effort:** S.
- **Dependencies:** none (reconstruct from git history + existing plan/CHANGELOG).
- **Risks:** reconstructing unrecorded decisions; keep links valid for
  `make docs-check`.

### C5 — Integration/paper-account tier scaffold
- **Objective:** add `test/` with the `integration` build tag, an
  environment-gated skip (`IBKR_GATEWAY`), and read-only smoke coverage; update
  TESTING.md/ROADMAP.
- **Why now:** it is the only remaining proof of real-world interoperability
  after C1/C2; the tier is already specified but absent.
- **Effort:** M (scaffold) / L (real coverage).
- **Dependencies:** C1 + C2; paper-account credentials; ADR 0009 write-safety
  rules.
- **Risks:** never runs in CI; credential handling; must never issue live
  writes/transfers.

### C6 — Benchmarks, fuzz tests, and CI quality gates (backlog bundle)
- **Objective:** add `Benchmark*` for transport/ratelimit/decode/mock routing,
  `Fuzz*` for spec/path normalization and `patch_spec.py` inputs, a coverage
  badge/upload step, and optional pre-commit hooks; correct ROADMAP's CI note.
- **Why now:** these are long-standing backlog items and cheap to schedule
  alongside correctness work.
- **Effort:** M.
- **Dependencies:** none; CI already exists (workflow job additions only).
- **Risks:** benchmark noise and flakiness; fuzz corpus maintenance; keep
  ADR 0004 (no new tool deps) intact.

## 4. Recommended next phase — "Phase 9: Correctness & conformance closure"

Prioritize the two known SDK correctness defects (D1, D2), then install the
guard that prevents their class of drift, and close out the run/CI hygiene gaps.
Defer the integration tier and backlog bundle to a follow-on phase.

**Draft task breakdown**

| ID | Objective | Role | Acceptance criteria |
|----|-----------|------|---------------------|
| T9.1 | **D1 fix.** Patch `scripts/patch_spec.py` so nil optional `interface{}` params are skipped (or emitted safely) in generated request builders; regenerate. | backend / codegen | `GetConidsByExchange`, `GetContractInfo`, `GetAllFYIs` no longer panic when optionals are unset; new regression tests call all three against `internal/mockgateway`; `make codegen-verify`, `make check`, `make test-race` pass; `client/*.gen.go` regenerated, never hand-edited. |
| T9.2 | **D2 fix.** Correct the 8 REST wrapper decoders (`rest.go`, `rest_utilities.go`) to the spec/generated-model shape, including removing the second read of an already-closed body in `RESTTaxDocuments.Generate`. | backend | Each of the 8 ops decodes its mock fixture without error and yields the expected values; regression tests added; `make check` + `make test-race` pass. |
| T9.3 | **Fixture shape guard.** Add a mock test validating every fixture against its generated response shape/model. | backend / testing | Guard passes on the corrected fixtures; a deliberately mismatched fixture makes it fail; no new dependency (ADR 0004). |
| T9.4 | **Run close-out + index.** Backfill `2026-09-17-full-api-coverage/{todos,report}.md`, write the mock-gateway `report.md`, create `docs/runs/index.md`, wire it into `make docs-check`. | docs / orchestrator | Both run directories complete; index links resolve; `make docs-check` + `make license-check` pass. |
| T9.5 | **Integration scaffold.** Add `test/` with the `integration` build tag and env-gated skip; document in TESTING.md/ROADMAP. | backend / testing | Package compiles under the tag, skips cleanly without `IBKR_GATEWAY`, issues no writes; docs updated. |

**Order:** T9.1 and T9.2 first (independent of each other), then T9.3 (depends on
both); T9.4 independent and can run in parallel; T9.5 after T9.1/T9.2.
**Follow-on phase:** C6 (benchmarks/fuzz/coverage/pre-commit) and deeper C5
coverage.

## 5. Open questions for the human

1. **D1 toolchain:** is fetching/installing `oapi-codegen@v2.8.0` (`make tools`)
   and the upstream spec acceptable in this environment, or should D1 be staged
   as a patch-only change validated in CI (`codegen-verify`)?
2. **D2 ownership:** for each of the 8 ops, is the spec/generated model the
   authoritative shape (fix wrappers) or were some wrappers intentionally
   narrow (fix fixtures)? D2 currently presumes wrapper-side fixes.
3. **Shape guard strictness (C3):** match against generated response types
   (partial, many `interface{}`), or generate a JSON-shape table from the spec?
   Either way, confirm no new dependency is acceptable.
4. **Integration tier (C5):** are paper-account credentials available, and must
   the tier stay entirely out of CI? What read-only smoke set is sufficient?
5. **Scope/sequencing:** is correctness (D1/D2) the priority for the next phase,
   or should the backlog bundle (benchmarks/fuzz/coverage badge) share the slot?
6. **Docs index:** should `docs/runs/index.md` be a versioned artifact (as
   T10 implies) and is there a preferred template/format?
