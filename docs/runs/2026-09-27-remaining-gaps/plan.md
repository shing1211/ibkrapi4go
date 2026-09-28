# Plan: close the remaining engineering gaps

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `ddaf9b5` (`v1.1.12`)

## Objective

Finish the three items the previous run left open, and fix anything the work
uncovers.

## Order, and why

The float32 classification goes **first**, not because it is the biggest item but
because it is the one whose premise might be wrong. The previous run recommended
"classify all 198 from the fixtures and produce a report before changing any
spec" — and the report could plausibly have said *don't*. Doing it first means the
rest of the run is planned against a fact rather than an assumption.

Coverage and the `unused` blind spot follow, and both turned out to be cheaper
than estimated: the coverage work and the untested-public-API work are the same
work.

## Approach

1. **Measure before changing.** A field count is not a defect. The classification
   asks whether each field is *reachable* and what the gateway actually sends;
   only then is a change justified.
2. **Prefer the existing tool.** The `unused` blind spot could be closed with a
   hand-rolled static-analysis script, which would be a new thing to maintain and a
   new source of false positives. `golangci-lint run --tests=false` uses the
   linter that already understands Go.
3. **Mutation-check anything new.** Every test added here is observed failing
   against the defect it targets, because the alternative is a test that documents
   a guess.

## Tasks

| # | Task | Outcome |
|---|------|---------|
| 1 | Classify the monetary `float32` fields | 44 names, 31 structs, **0** bare-number cases |
| 2 | Narrow to the production surface | 25 of 31 never decoded; **3** request types are real |
| 3 | Fix the request-direction rounding | `moneyToNumber`; mutation shows `12345679` |
| 4 | Strengthen the amount assertions | 3 float32-exact values replaced with literal comparison |
| 5 | Cover 11 untested `ModelManager` methods | pkg/ibkr 58.9% → 61.7% |
| 6 | Close the `unused` blind spot | second lint pass, 3 findings resolved |
| 7 | Wire the example's dead helpers in | `handleAuthError` used; `httpStatusCheck` deleted |

## Outcome

The classification changed the plan: there was no mass retyping to do, and a live
money bug in the request direction instead. That is the opposite of what the
previous run's `next-phase.md` expected, and it is why the classification ran
first.
