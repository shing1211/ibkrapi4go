# Plan: a gate for unreferenced internal packages

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `2d11db4` (`v1.1.22`)

## Objective

Tenth request to "plan and implement remaining blocking items". The previous run
found `internal/fake`: seven files, no importer anywhere, no test, unreachable
externally, invisible to `unused`. The run ended by naming the prevention as the one
piece of work that needed no decision, and this is it.

## Why this gate and not the one that was withdrawn

`check_money.py`'s proposed precision gate was withdrawn two runs ago after being
measured: 27 false positives in one formulation, 16 false negatives in the other. A
gate with judgement calls in it gets ignored the first time it is wrong.

This one has no judgement calls. The rule is:

> Every package under `internal/` appears in at least one import somewhere in the
> module.

That is decidable, has no thresholds, no pattern matching and no judgement about
intent. A test-only importer counts, because a test-only importer is a legitimate
consumer. The only escape hatch is `ALLOWED`, keyed by package path, with a reason.

## What it found immediately

`internal/fake`, which is the package the previous run documented. It is on
`ALLOWED` with the reason it is orphaned, so the gate is green today and the
exception is visible in the output rather than hidden. If the maintainer's decision
goes the other way, the entry goes with it.

## Wiring

- `make internal-refs-check`, and into `make check`.
- A CI step beside the money-types check.
- AGENTS.md rule 8, because a rule nobody is told about is a rule that gets
  reintroduced.

## Verification approach

The self-test exercises the resolver with synthetic cases, which is not enough. A
gate that has only ever been run against a passing tree is exactly the kind of test
this sequence has learned not to trust. So it is also run against the real tree with
a synthetic package: unreferenced must fail and name it; referenced by a test must
pass; removal must restore green.
