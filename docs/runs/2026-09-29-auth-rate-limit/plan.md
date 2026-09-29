# Plan: stop a one-shot client paying 2s for a precaution

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `e1fe64d` (`v1.1.25`)

## Where this came from

The `v1.1.25` backlog listed, as its headline item, that every CLI command costs ~2.0s
of fixed latency and that the cause was unknown. It was written from a measurement
(`Session.Initialize` 1.007s, `cli.Close` 995ms) and a list of things already ruled
out. A recorded measurement with no cause is a lead, not a finding.

## Steps

1. **Find the 2s, not describe it again.** Time each HTTP request separately, and
   record the gap *between* one response and the next request, so a wait before a call
   is distinguishable from a wait after a failed one.
2. **Ask the code to say so.** The client has a DEBUG log line for rate-limiter waits.
   Turning it on with a mock gateway and a CLI-shaped call sequence names the culprit
   in one run, with its duration.
3. **Separate the two waits.** They are not the same problem: one is a deliberate
   precaution that may be right, the other blocks process exit for a call whose result
   is discarded.
4. **Change as little as the finding supports.** The unconditional half is a
   classification change. The configurable half adds an option and changes no default.
5. **Update the doc that the change contradicts**, and check the repo's own stability
   contract before choosing a version number.

## What was deliberately not done

- The auth default was not raised. See the ADR: the premise is unverified, and
  guessing trades a 2s CLI for production 429s.
- The limiter's construction guard was not restructured, only documented.
- The open question - whether IBKR actually rate-limits auth - is recorded, not
  answered. It needs a live gateway.
