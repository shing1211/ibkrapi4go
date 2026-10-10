# Plan: drive the CLI subcommands that had never run

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `f95f4c4` (`v1.1.23`), plus the `v1.1.24` error-budget fix

## Where this came from

The `v1.1.24` backlog listed `cmd/ibkr` at 33% and said it "needs a client-factory
seam first". Checking that against the tree rather than carrying it forward showed the
seam had existed since 1.1.14, so the item was smaller than recorded: the work was using
an existing seam, not building one.

That correction is why this run exists, and it set the method: every claim gets checked
against the code before it is acted on, because the previous backlog also pointed at a
file that does not exist and named two functions that do not exist.

## Steps

1. **Confirm the seam is usable, not just present.** `env.newClient` is a func field, all
   9 call sites go through it, and `newClientFromArgs` honours `--gateway`. Verified
   `mockgateway.Handler()` returns an `http.Handler` so it drops into `httptest`, and
   that `pkg/ibkr/endtoend_test.go` builds a client with no credentials at all.
2. **Build a harness**, not a framework: one helper that isolates the config, starts the
   mock, and returns the env plus the global-flag arguments.
3. **Happy path and error paths per subcommand**, because the existing 20 tests already
   cover flag parsing and dispatch - the error paths are what the new code can get wrong.
4. **Treat every failure as a possible finding.** Two of the failures were real bugs and
   one of them was not in the subcommands at all.
5. **Full gate sweep, then release.**

## What was deliberately not done

- No new seam, no new abstraction over the subcommands. The existing one was enough.
- `--rest` is left inert on purpose, with a comment at the point of the omission.
- The 2s command latency was measured and filed, not root-caused. It is a different
  investigation and guessing at it here would have produced an unverified claim.
