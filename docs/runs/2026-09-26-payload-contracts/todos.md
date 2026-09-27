# Payload Contracts and Design Checks - Todos

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| N1 | Characterise four `rest_banking.go` payloads with byte-level tests | tester | done | - | Exact JSON key set **and value types** pinned for `CancelInstruction`, `CancelInstructionsBulk`, `TransferV2`, `TransferBulkV2`. Fails if a key is added, removed, or changes JSON type. Passes on current code |
| N5 | Delete three dead package-private helpers | backend | done | - | `strToDecimalPtr` (`rest_banking.go:34`), `makeTradingInstrumentRef` (`:1378`), `f32PtrToInt64Ptr` (`:1415`) and their three tests removed. Zero references confirmed by grep and a build. Suite green |
| N6 | Correct the inert `OpGetRequestsStatus` mock fixture | backend | done | - | `routes_rest.go:268` body uses `dateSubmitted`; the test asserting a **nil** `ExecutedAt` on the default path is inverted to require a populated one; the `oneOf` arm without a timestamp stays covered by an explicit override; no other consumer depended on `executedAt` |
| N7 | Drop the wasted `Token(ctx)` fetch | backend | done | - | `rest.go` no longer acquires a token it discards; a test asserts `Authorization` still arrives (set by `internal/transport.go`) and that no extra acquisition occurs |
| N8 | `check_design` check for `02-client.md` | backend | deferred | - | Checker reads the document; a deliberate edit fails the check; presence and order both verified, matching the `01-transport.md` pattern |
| N9 | `check_design` checks for `04`-`09`, one document per commit | backend | deferred | N8 | Each document has a check that fails on a deliberate edit; drift fixed in the same commit as the check that found it |
| N10 | Cover `cmd/ibkr` order validation and flag handling | tester | deferred | - | `orders.go:150-158` and `:187-192`, `main.go:108`/`:137`, `portfolio.go:43`, `stream.go:96`, `positions.go:69`, `config.go:22`/`:34`/`:55` covered. `main()` stays 0%. **No `ci.yml` edit** |
| N11 | Re-measure coverage and ratchet the floor | orchestrator | done | N5,N6,N10 | Deduplicated profile that reproduces `go tool cover -func`; floor set at or below measured; buffer and its reason recorded |
| N12 | Close-out | orchestrator | done | all | `report.md` final, plan actuals recorded, `docs/runs/index.md` row set to complete |

## Blocked — awaiting a human decision

Not started. Each would change what goes on the wire for a money-adjacent
operation, and the previous run's own plan gates them on questions only a human
or a real gateway can answer.

| ID | Blocked on | Question |
|---|---|---|
| D10 | open question 4 | Does bulk-cancel accept a `Reason` at all? `CancelInstructionsBulk` (`rest_banking.go:390-394`) omits it while the single path (`:364`) sets it |
| D11 | open question 4 | Which V2 quantity JSON type does the gateway accept? Single emits a number (`:750`), bulk emits a string (`:802`, `:820`) |
| D12 | open question 5 | Is `AssetTransferRequest.Quantity` on a V2 path a bug or a V1-only field? Silently dropped at `:748-752` and `:816-821` |
| D14/D15 | open question 2 | Is `float32` money precision acceptable? Needs a spec change plus `make codegen` and touches ADR 0008's intent |

N1 exists precisely so that, once these are answered, each fix has a before and
an after rather than being decided from the spec alone.

## Baseline

Carried from the previous run's close-out, re-measured at the start of this run
before any change.

- Coverage: **58.7%** (CI flags, deduplicated profile)
- CI floor: **58%** (`.github/workflows/ci.yml:76`)
- `HEAD`: `aa43fdb`, tags `v1.1.3`, `v1.1.4`, `v1.1.5` on GitHub and Gitee
- Working tree: clean

## Method notes carried forward

- Quote comma-bearing flags in PowerShell: `-coverpkg=a,b` splits into two args.
- Deduplicate coverage profile blocks per test binary before trusting a total.
- Check routes and fixtures in **both** `routes_cpapi.go` and `routes_rest.go`.
- Re-check a cited line number before using it; the previous run's own fix moved
  one of them.
- Prove a test fails before the fix.
- Verify each incidental sub-agent finding individually — two of three were false
  last run.

## Added during the run

| ID | Task | Role | Status | Acceptance |
|----|------|------|--------|------------|
| N7a | Correct the inert createSsoSessions fixture | backend | done | Fixture used camelCase where the generated type tags snake_case; a new test decodes every field and fails pre-fix |
| N13 | Make the fixture shape check real | tester | done | `DisallowUnknownFields` was inert against `any`. Now resolves each op's real response type by reflection and flags wrong keys. Caught both historical bugs; 88 failures surfaced |
| N14 | Stop the check producing false positives | tester | done | 88 -> 12. Decode path derived from `pkg/ibkr` call sites via `go/ast`; 110 ops exempt with a counted reason and `file:line`; 12 unresolved stay key-checked |
| N15 | Correct the 12 remaining fixtures | tester | done | 12 -> 0, check now stricter (key-checked 50 -> 59 ops, union 1 -> 2). No test needed inverting |

N8/N9/N10 are deferred: the shape-check work expanded to fill the run, and
N10 is deferred by choice since the coverage margin is adequate at 0.9 points.
