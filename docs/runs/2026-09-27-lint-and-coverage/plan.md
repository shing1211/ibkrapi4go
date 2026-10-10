# Plan: lint gate, CLI tests, and design-doc coverage

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `860cafd` (`v1.1.8`)

## Starting position

`next-phase.md` from the banking-contracts run listed three remaining items and
two blocked decisions. All three are in scope here:

1. `.golangci.yml` is schema-invalid and 733 library findings are unenforced.
2. `cmd/ibkr` has no tests and needs an `os.Args` seam first.
3. `docs/design/07-money-and-numbers.md` is the one design document without a
   claim-checker.

Plus two questions worth answering rather than assuming: whether the 193-vs-191
fixture count is a coverage gap, and whether the D15 spec fix stands up.

## Approach

**Order matters here.** Migrating the lint config is not housekeeping - it is the
instrument that finds the other work. Everything else is downstream of it, so it
goes first, and the real defects it surfaces are fixed before any suppression is
considered.

The rule applied throughout: **a finding is fixed, or it is excluded with a written
reason.** Not silenced. The `.golangci.yml` comments are the record of which is
which, because a config that quietly turns linters off is worse than one that was
never valid - it looks like enforcement.

For each behavioural fix, a mutation check: revert the fix, confirm the assertion
fails with a diagnostic that names the defect, restore. A test that has never been
observed failing is not evidence.

## What the migration was expected to cost

The recorded 733 findings were never real - the config was rejected before any
analysis ran. The honest count after migration was 192, and the composition
predicted a large majority of noise: deferred `Close` calls, doc comments, and
bodyclose false positives through the SDK's decode helper. The plan assumed most
of it would be convention to document rather than code to change, and that a
handful would be real defects. That assumption turned out to be roughly right and
directionally wrong: fewer than a dozen were real, but they mattered more than the
noise did.

## Tasks

| # | Item | Disposition |
|---|------|-------------|
| 1 | Migrate `.golangci.yml` to v2 | done; also fixed revive's broken `exported` config |
| 2 | Triage 192 findings | done; every exclusion carries its reason |
| 3 | `cmd/ibkr` seam and tests | done; `run`, `parseGlobalFlags`, `runCompletion` parameterised |
| 4 | `07-money` claim-checker | done; 9 of 9 documents verified |
| 5 | 193-vs-191 fixture count | investigated; intentional, not a gap |
| 6 | D15 against a live gateway | not possible without an account; recorded |

## Ordering note

Configuration came before the test batch, which is the reverse of the previous
run's instruction. That is deliberate: the lint config is not a gate here, it is
the thing that finds the defects, so it had to exist before the fixes could be
identified. Everything else - tests, verification - came last.
