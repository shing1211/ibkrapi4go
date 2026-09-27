# Design Checkers and Residual Gaps - Todos

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| S1 | SPDX headers on the two shell scripts | docs | todo | - | `addlicense -check` reports no missing headers |
| S2 | `check_design` check for `06-errors-retries.md` | backend | todo | - | The `Error` field set, `RetryPolicy` defaults, the safe-method list, the ADR 0009 no-retry rule and the breaker default are each verified. Every check observed failing once, with output |
| S3 | `check_design` check for `09-orders-and-confirmation.md` | backend | todo | S2 | Same standard. Targets the acknowledgement and confirmation claims |
| S4 | `check_design` checks for `02`, `04`, `05`, `08` | backend | todo | S2 | One document per commit; each check observed failing once |
| S5 | Decide whether `07-money-and-numbers.md` needs a doc check | architect | todo | - | A written decision. `check_money.py` may already cover its main claim; if so, say so rather than adding a duplicate |
| S6 | Cover `cmd/ibkr` order validation and flag handling | tester | todo | - | `orders.go:150-158` and `:187-192`, `main.go:108`/`:137`, `portfolio.go:43`, `stream.go:96`, `positions.go:69`, `config.go:22`/`:34`/`:55` covered. `main()` stays 0%. **No `ci.yml` edit** |
| S7 | Re-measure coverage and ratchet the floor | orchestrator | todo | S6 | Deduplicated profile reproducing `go tool cover -func`; floor at or below measured; buffer and reason recorded |
| S8 | Close-out | orchestrator | todo | all | `report.md` final, plan actuals recorded, `docs/runs/index.md` row complete |

## Resolved this run, recorded because they were silent

| Item | Finding |
|------|---------|
| `scripts/changelog-gen.sh`, `scripts/sbom-gen.sh` | No SPDX header, which `AGENTS.md` requires of every new source file. Pre-existing, last touched in `4f12df0`, and not caught by CI because `license-check` is not a CI step and there is no pre-commit config. That second fact is itself worth noting: the rule is unenforced |

## Carried forward, still blocked

Unchanged from `2026-09-26-payload-contracts/next-phase.md`. Each changes what
goes on the wire for a money-adjacent operation.

| ID | Blocked on |
|----|------------|
| D10 | Whether bulk-cancel accepts a `Reason` at all |
| D11 | Which V2 quantity JSON type the gateway accepts; ADR 0008 sides with the bulk path |
| D12 | Whether `AssetTransferRequest.Quantity` on a V2 path is a bug or a deprecation |
| D14, D15 | Whether `float32` money precision is acceptable; needs a spec change plus `make codegen` |

## Baseline

- `HEAD`: `f9345dd`, tag `v1.1.6` on GitHub and Gitee
- Working tree: clean
- Coverage: **58.9%** (CI flags, deduplicated profile)
- CI floor: **58%** (`.github/workflows/ci.yml:76`), 0.9-point margin
- `check_design` currently verifies **2 of 9** design documents:
  `01-transport.md` (middleware chain) and `03-managers.md` (manager method
  counts). Unverified: 02, 04, 05, 06, 07, 08, 09.

## Added during the run

| ID | Task | Role | Status | Acceptance |
|----|------|------|--------|------------|
| S2b | Document `RetryPolicy.Metrics` | orchestrator | done | Human ruled the code authoritative; document corrected. The checker gained an explicit `optional;` no-default marker so a legitimately unassigned field can be expressed |
| S2c | Enforce `check_design` in CI and `make check` | devops | done | `docs` job gained `setup-go` + the step, validated by PyYAML parse and 5 structural asserts. New `design-check` target; `check` is now `fmt vet money-check design-check test` |
| S2d | Fix the `scripts/` lint findings | devops | done | 8 -> 0 uncapped. Each `//nolint` carries a site-specific reason |
| S2e | Let the check express an unassigned field | backend | done | The check had required every documented field to be assigned; bending production code to satisfy it was rejected as the wrong fix |
| S3b | Enforce `Reply` and `OrderRequest` both directions | backend | done | S3 left a hole because the document disagreed. The human resolved it in the code's favour and the hole was closed, with 11 new red cases and 2 stale green cases replaced |
| S4b | Re-anchor harness cases after a document fix | backend | done | 4 cases re-anchored, not the 2 that failed - a sweep found 2 more, one of them green-but-anchored-to-nothing |

## Outcome

- `check_design`: **2 of 9 documents -> 8 of 9**, and from manual-only to running on every push.
- 18 checks added, each observed failing. Harness: **232 subtests**, 7 top-level controls including 2 negative controls.
- 5 false statements in design documents found and corrected, the worst being a documented `Reply.Message string` that gave callers a compile error.
- `scripts/` lint: 8 -> 0. Coverage 58.9% -> 59.0%; floor unchanged at 58% since this run added no production statements.
- **No production code, no generated code, no dependency change.**
