# Audit Remediation

- **Run:** `docs/runs/2026-09-26-audit-remediation/`
- **Base commit:** `c3a188a`
- **Mode:** BUILD
- **Status:** complete

## Objective

React to a full-codebase audit that compared the git history and the markdown
documentation against the implementation, then close every gap it found. The
audit ran after the `blueprint-hardening` and `ws-shutdown` runs and surfaced
two classes of problem: work that shipped without a changelog entry or a
matching document, and tests that were claimed but did not exist.

## Findings

| # | Finding | Severity |
|---|---------|----------|
| 1 | Coverage gate parsed an empty value, so the step never enforced anything | High |
| 2 | `codegen drift` failed every run because the committed generated client was CRLF while Linux CI generates LF | High |
| 3 | Eight spec operations had no public wrapper, contradicting the "all operations implemented" claim | High |
| 4 | Streamed `Update.Status` was unreachable because field `6509` was filtered before delivery | High |
| 5 | `check_design` compared an always-empty middleware order, so it could not detect drift | High |
| 6 | Seven implementation paths had no test | Medium |
| 7 | `make codegen`, `make codegen-verify`, and `make docs-spec` failed on Windows | Medium |
| 8 | Documentation made claims that did not match code or tooling | Medium |

## Tasks

| ID | Objective | Status | Acceptance |
|----|-----------|--------|------------|
| S1 | Repair the coverage gate | done | Parses a real percentage, compares as a float, enforces 35% |
| S2 | Close the seven untested paths | done | Each gap has a test that fails without its fix |
| S3 | Order resubscribe before the reconnect notification | done | Regression test fails under the old ordering |
| S4 | Implement the eight missing operations | done | 193 operations, 451 schemas, coverage counts 123/70/193 |
| S5 | Correct the inaccurate documentation | done | No claim contradicts the code or the CI configuration |
| E3 | Make `check_design` compare code against the document | done | Reordering middleware fails the check |
| T1 | Make the codegen scripts runnable on Windows | done | `make codegen` reproduces the committed client |
| T2 | Verify the `SubmitModelPortfolioOrder` response decode | done | Full snake_case decode asserted end to end |
| B | Reconcile run bookkeeping | done | No task left in `review`; index reflects the work |

## Constraints

- No breaking changes to existing exported API.
- No new dependencies.
- No generated-client edits by hand; all codegen defects fixed at the spec level.
- Money and quantity values stay `string`/`json.Number` per ADR 0008.
- The integration suite stays read-only.

## Verification

- `gofmt -s -l .` — clean, matching the CI step
- `go build ./...` and `go vet ./...` — pass
- `go test ./...` — all packages pass
- `go test -race` over `./internal/...`, `./pkg/ibkr/...`, `./examples/live/...` — pass
- `./scripts/validate_codegen.sh` — committed client matches a fresh generation
- `check_money`, `check_links`, `check_i18n`, `check_design` — pass
- `go vet -tags=integration ./test/...` — compiles
