# todos.md — Audit Remediation

Status values: `todo` · `doing` · `blocked` · `review` · `done`

## Correctness and tooling

| ID | Task | Status | Verification |
|----|------|--------|--------------|
| S1 | Repair the coverage gate parser and comparison | done | Real value parsed; 36.8% passes a 35% floor, 12.5% fails |
| S3 | Resubscribe before notifying a reconnect | done | Test fails with the old ordering |
| E3 | Make `check_design` diff code against the documented chain | done | Reordering `retry`/`rateLimit` in code fails the check |
| T1 | Fix `make codegen` and `make codegen-verify` on Windows | done | `codegen.sh` reproduces the committed client byte-for-byte |
| T3 | Fix `make docs-spec` on Windows | done | `gen_spec_index.py` reads and writes UTF-8 explicitly |
| F1 | Deliver field `6509` so `Update.Status` is reachable | done | End-to-end test asserts delayed/frozen/not-subscribed parsing |
| F2 | Add a line-ending policy | done | `.gitattributes` enforces LF |

## Feature gaps

| ID | Task | Status | Verification |
|----|------|--------|--------------|
| S4.1 | Add 8 mock operation constants, routes, and fixtures | done | 7 routed; the 8th is unroutable by construction |
| S4.2 | Add 8 public wrappers | done | 8 end-to-end tests against the mock |
| S4.3 | Regenerate `docs/SPEC.md` | done | 193 operations, 451 schemas |
| S4.4 | Update the canonical coverage counts | done | 123/70/193; all three assertions pass |
| T2 | Verify the `SubmitModelPortfolioOrder` decode | done | Full decode asserted via the shared-route fixture |

## Test coverage

| ID | Task | Status | Verification |
|----|------|--------|--------------|
| S2.1 | Multi-drop reconnect scripting in the mock | done | `DropConnections`, plus `AcceptedConnections` |
| S2.2 | Fix the misnamed three-drop test | done | Scripts 3 drops; fails at 1 |
| S2.3 | Late-dial discard | done | `ErrClosed`, nothing published, socket closed |
| S2.4 | Dispatch-level sequence gap | done | Exactly one `*WSGapError`, tick still delivered |
| S2.5 | `ForceRefresh` in-flight behaviour | done | Joins the flight; honours cancellation |
| S2.6 | `Invalidate` on a closed client | done | Documented and asserted idempotent |
| S2.7 | OAuth2 example helpers | done | 5 tests in `examples/live` |

## Documentation

| ID | Task | Status | Verification |
|----|------|--------|--------------|
| S5.1 | Correct `docs/TESTING.md` claims | done | Checked against the workflows |
| S5.2 | Correct `docs/STREAMING.md` claims | done | Checked against the delivery code |
| S5.3 | Correct `CHANGELOG.md` history | done | Counts re-measured |
| S5.4 | Correct README diagrams and drop pre-alpha wording | done | `check_i18n` passes |
| S5.5 | Restore the missing `## [1.1.0]` heading | done | Heading present |
| B1 | Mark E3 done in both run records | done | No `review` rows remain |
| B2 | Clean the duplicated headings and stale backlog | done | Single "Recommended next phase" |
| B3 | Create this run directory and index it | done | Row present in `docs/runs/index.md` |
