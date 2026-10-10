# Plan: the breaker clock seam, and a package that cannot reach it

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `2bc985c` (`v1.1.21`)

## Objective

Ninth request to "plan and implement remaining blocking items". Two runs have now
said the offline backlog is empty. That claim was not verified, so this run verifies
it rather than repeating it.

Per-package coverage was the source of the earlier claim, and it listed
`internal/fake` at 0.0% - which was read as "not worth it" and skipped. A 0% package
is not a coverage gap, it is a different thing.

## What the check found

`internal/fake` is **completely unreferenced**: 7 files, nothing in the module
imports it, no test uses it, and `internal/` means nothing outside the module can
either. It has been that way since `internal/mockgateway` superseded it.

`CHANGELOG.md:1373` records it as a v1.0.0 feature - "internal/fake/ package with
five fake implementations for testing" - so it was deliberate, not an accident. Which
raises the question of whether it is still wanted.

The answer is more interesting than "dead code":

- `internal.Clock` is a **struct with unexported function fields** and **no exported
  constructor**.
- `Breaker.SetClock(c *Clock)` therefore accepts only a `*internal.Clock`, which
  only code **inside** `internal` can construct with a custom `nowFunc`.
- `internal/fake.Clock` is a **different type** whose doc comment claims it
  "implements internal.Clock" - but `internal.Clock` is a struct, not an interface,
  so there is nothing to implement and nothing to pass.

So `internal/fake` is not merely unused. It is **incompatible with the seam it was
written for**, because that seam moved from an interface to a struct with unexported
fields and the fake was never updated. A design change left it orphaned.

## What this run delivers

1. The deterministic breaker tests the seam was built for, driven in-package
   because that is the only place the clock can be constructed. These replace real
   `time.Sleep` waits.
2. The error-budget semantics pinned explicitly, including the part that
   contradicts the documentation.
3. `internal/fake` surfaced as a decision rather than deleted on my own judgement -
   the changelog records it as an intended feature, so removing it is the
   maintainer's call, not a cleanup.
