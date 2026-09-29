# Plan: make the error budget recover the way its own contract says

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `f95f4c4` (`v1.1.23`)

## The question

`WithCircuitBreakerBudget(budget, size)` is documented as "when budget failures occur
within the last size outcomes the breaker opens". The implementation did not do that.
This run settles whether that is a bug worth fixing and, if so, fixes it.

## Why it was worth proving first

Two earlier runs reasoned about this from the source and got it wrong in the same
direction - both recorded the missing time window as the defect. Reading the code does
not settle it: the question is what the breaker does across a *sequence* of outcomes,
and a sequence is what a test or a probe produces. So this run starts by proving the
behaviour rather than by re-reading it.

## Steps

1. **Probe the current behaviour, do not change it.** Record a burst of failures, let
   the cooldown elapse, record a success, then one more failure. Read the window and
   the trip state at each step. Throwaway file, deleted after.
2. **Decide.** Fix the count-based budget as a bug, or design a new time-based public
   option. These are different pieces of work with different versioning.
3. **Write the regression test first** and watch it fail on the unfixed code. A test
   that has never been seen red is not evidence.
4. **Fix.** The window becomes a rolling window over the last `size` **outcomes**, with
   successes recorded so they push old failures out.
5. **Update the tests that encoded the defect.** Some pinned the old behaviour on
   purpose, and one is named for the opposite of what it asserts.
6. **Mutation-verify**, because a test that passes for the wrong reason is the failure
   mode this whole repository has been burned by repeatedly.
7. **Full gate sweep, then release.**

## What was deliberately not done

- No time-based eviction. That is a new public option with new semantics, not a bug
  fix, and it would need its own decision.
- No ADR. Nothing here adds a dependency or a retry policy. The public contract string
  is unchanged; the implementation is brought into line with it.
