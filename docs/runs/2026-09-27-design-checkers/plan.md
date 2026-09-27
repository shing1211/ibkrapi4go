# Design Checkers and Residual Gaps - Plan

## Objective

Extend `check_design` beyond the 2 of 9 design documents it currently verifies,
starting with the document whose subject matter this session proved to be
defect-prone. Clear the small residual gaps the last two runs recorded, and cover
`cmd/ibkr`'s validation helpers.

## Why this, now

The previous two runs found seven production defects and one mock-gateway defect
class, and a recurring theme ran through all of it: **a check that looked like it
verified something, but did not.**

- `internal/mockgateway/shape_test.go` set `DisallowUnknownFields` against
  `var js any`. It could never fire. Two live defects were instances of the class
  it existed to catch.
- `check_design` reads 2 of 9 design documents. The 7 unverified ones are prose
  that no gate reads, so drift in them is silent.

`06-errors-retries.md` is the highest-value target because its subject matter is
where this session's most user-visible defect lived. The document states the
`Error` struct's fields and the `RetryPolicy` defaults as fact; `wrapOp` was
returning an `*Error` with `Code: ""` and `HTTPStatus: 0`, which contradicts it,
and nothing failed. A checker over that document would have caught it.

## Tasks

| ID | Task | Role | Size | Blocking |
|----|------|------|------|----------|
| S1 | Add SPDX headers to `scripts/changelog-gen.sh` and `scripts/sbom-gen.sh` | docs | S | none |
| S2 | `check_design` check for `06-errors-retries.md` | backend | M | none |
| S3 | `check_design` check for `09-orders-and-confirmation.md` | backend | M | none |
| S4 | `check_design` checks for `02`, `04`, `05`, `08`, one document per commit | backend | M | S2 pattern |
| S5 | Decide whether `07-money-and-numbers.md` needs a doc check at all | architect | S | none |
| S6 | Cover `cmd/ibkr` order validation and flag handling | tester | M | extract parse loop |
| S7 | Re-measure coverage and ratchet the floor | orchestrator | S | S6 |
| S8 | Close-out | orchestrator | S | all |

### S2 detail — what is checkable in `06-errors-retries.md`

Read the document and pick the claims that are **facts about code** rather than
aspirations. Candidates already visible:

- The `Error` struct's field set and types, as listed in the document's code
  block, against the real struct in `pkg/ibkr`.
- `RetryPolicy` field defaults: `MaxAttempts` 3, `BaseDelay` 200ms, `MaxDelay`
  5s, `Jitter` true, `RetryOnStatus` 429/500/502/503/504 — against
  `internal/retry.go`.
- "Safe methods only: GET, HEAD, OPTIONS" against the retry middleware's method
  test.
- "Order/instruction mutations never retried" (ADR 0009) — the rule most worth
  enforcing, and the one whose violation would be most expensive.
- Circuit breaker "disabled by default" against `internal/breaker.go`.

Prefer a small number of checks that are each individually meaningful over a
diff-everything check. A checker that compares whole documents will rot and be
ignored; one that pins six load-bearing constants will be read.

## Method constraints

Carried forward from the last two runs, because each of these has cost time:

- **A check must be proven load-bearing.** Introduce the defect, show the check
  fails, revert, show it passes. A new check that has never been observed failing
  is not a check.
- **Do not infer behaviour from counts or patterns.** Quote the lines that
  establish each mechanism.
- **Resolve the operation's real type; do not scan for a name.** `executedAt` is
  a valid tag on a different operation, which is how an earlier version of a
  check blessed a bug.
- **Do not exempt uncertainty into a pass.** When a decode path or a doc claim
  cannot be established, report it and keep checking.
- **If a brief and the code disagree, the red test is the deliverable.** A green
  test that cannot fail is worse than a red one.
- Quote comma-bearing flags in PowerShell; unquoted `-coverpkg=a,b` splits and
  Go reports `[setup failed]`, which looks like a broken tree.
- `Get-Content` defaults to CP1252 and renders UTF-8 as mojibake. Use
  `-Encoding UTF8`. It is a reading artifact; never "fix" it.

## Risks

- **Extending the checker will surface pre-existing drift** in the seven
  documents, which then has to be fixed in the same change. One document per
  commit keeps each fix attributable. If a document turns out to be substantially
  wrong, stop and report rather than rewriting it under a check.
- **Over-broad checks rot.** A checker that fails on cosmetic reformatting gets
  disabled, which is worse than not having it.
- **S6 requires a small production refactor.** `runOrdersSubmit` reads
  `os.Args[2:]` directly (`cmd/ibkr/orders.go:82`); extracting the parse loop is
  the smaller change and makes the validation reachable without mutating globals.

## Out of scope

- **D10, D11, D12, D14, D15** — the deferred `rest_banking.go` wire contracts and
  the `float32` money precision. Each changes what goes on the wire for a
  money-adjacent operation and is blocked on a human decision or a real gateway.
  `2026-09-26-payload-contracts/next-phase.md` carries the open questions and the
  characterisation tests that make the eventual fixes provable.
- **Regeneration.** No spec change and no `make codegen` in this run; D14/D15
  need it and should be reviewed alone.
- No edits to `client/*.gen.go`. No new dependencies.

## Actuals vs. Plan

| Planned | Outcome |
|---------|---------|
| S1 SPDX headers on the two shell scripts | Done. `addlicense -check` now exits 0, verified with a control |
| S2 `check_design` for `06-errors-retries.md` | Done. 6 checks. The `Error` field set, the `RetryPolicy` defaults, the safe-method set, the ADR 0009 no-retry rule and the breaker default were all verified to agree with the code |
| S3 `check_design` for `09-orders-and-confirmation.md` | Done. 6 checks, and it found that the document's `Reply` block named a field that does not exist - a caller following it got a compile error |
| S4 `check_design` for `02`, `04`, `05`, `08` | Done. 12 checks. 6 further disagreements recorded, 2 of which turned out to be genuine document bugs |
| S5 decide whether `07` needs a doc check | Done: it does not. `check_money.py` already enforces its main claim, so a second checker would duplicate a live gate |
| S6 cover `cmd/ibkr` | **Deferred by choice.** Coverage margin is 1.0 point, and the human directed S3/S4 then close-out |
| S7 re-measure and ratchet | Done. 59.0% against the existing 58% floor. Not moved, because this run added no production statements |
| S8 close-out | Done |

### Where the plan was wrong

The plan's premise was that widening the checker from 2 documents to 9 was the
work. Verifying that premise first turned up something more urgent: the checker
was not run by CI and not in `make check`, so every document added would have
been an unrun gate. The human chose to enforce first, then widen, and the run
reordered itself around that.

The plan also under-scoped the lint problem. It recorded "6 pre-existing
findings" from a single uncapped run over `./scripts/`; the real figure across
CI's package set is 742 uncapped with 733 in library code, and the figure of 6
was itself an artifact of golangci-lint's default caps. `.golangci.yml` also
fails schema validation, so the author's linter settings have never applied.

### Unplanned work the run absorbed

Six tasks appeared that the plan did not anticipate:

- **S2b/S2e** - adding the `Metrics` field to the `06` document exposed that
  the new check could not express a field that is legitimately never assigned.
  The first fix considered was to assign `Metrics: nil` in production code; that
  was rejected as bending code to satisfy a checker, and the check was corrected
  instead.
- **S2c** - CI enforcement, the finding above.
- **S3b** - closing the enforcement hole S3 left once the document was corrected.
- **S4b** - re-anchoring harness cases after a document edit. Four were affected,
  not the two that failed: an anchor that no longer matches is a silent no-op, so
  one case was green while mutating nothing.
