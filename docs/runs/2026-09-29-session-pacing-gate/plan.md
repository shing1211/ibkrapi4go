# Plan: pin session pacing so a 2s defect cannot ship again

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `46d4535` (`v1.1.26`)

## Where this came from

The `v1.1.26` run found that every CLI invocation cost ~2.0s, root-caused it to the
hardcoded auth rate-limit bucket, and fixed it. What it could not do was stop the same
class of defect from shipping again: that 2s passed every gate in this repository for two
releases, because nothing in the repo asserted on pacing.

The `v1.1.26` backlog proposed "a single test that an Initialize-then-Close cycle
completes well inside a second". That proposal was wrong and this run exists to replace
it before anyone implements it.

## Why the proposal was wrong

A wall-clock assertion is flaky on a loaded CI runner, and a gate that is wrong often
gets muted rather than fixed - the same fate as the withdrawn `check_money.py` gate,
which this repository has now retracted twice. Adding a flaky gate would have been worse
than adding no gate, because it would have looked like coverage.

## The approach

The rate limiter already records `ibkr.ratelimit.waits` whenever a bucket appears empty
(`internal/ratelimit.go:130-133`), and `InMemoryMetrics` already exists with a
`Snapshot()` reader used by `pkg/ibkr/metrics_test.go`. So the assertion can be exact
and free of wall clock, using machinery the repo already has.

## Steps

1. **Assert on the counter, not the clock.** Count rate-limit waits across an
   Initialize → List → Close cycle against a mock gateway.
2. **Pair it.** One case for the default auth pacing, one for a caller that opts out.
   Same shape as the paired limiter tests from the previous run.
3. **Guard against a vacuous pass.** A zero-wait assertion on a run that made no requests
   is trivially true, so both cases also assert the four requests happened.
4. **Mutation-verify.** Put `/v1/api/logout` back into the auth set and confirm the test
   fails. A test never seen red is not evidence.
5. **No CI step.** It belongs in the normal `go test ./...` sweep; a gate that exists
   only in one CI job is a gate someone eventually drops.

## What was deliberately not done

- No wall-clock assertion, and no tolerance window.
- No CLI-shaped variant. `pkg/ibkr` builds the client directly, so it does not exercise
  `newClientFromArgs`; covering that too would couple the test to the command layer for
  little gain.
- No new gate script. This is a test, and tests are the gate.
