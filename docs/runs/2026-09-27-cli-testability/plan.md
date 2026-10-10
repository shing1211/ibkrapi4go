# Plan: make the CLI testable, and raise the coverage floor

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `c09e01d` (`v1.1.13`)

## Objective

Close the last two account-independent gaps from the previous run:

1. Give `cmd/ibkr` the same testability the SDK already has, so the six remaining
   subcommands are covered by real tests rather than by inspection.
2. Raise the CI coverage floor from 58% to a figure the suite actually earns.

## Why this was left blocked until now

`cmd/ibkr` was deliberately not tested, and the reason was not effort. Every
subcommand read `os.Args` and `os.Stdout` directly, so the only way to reach one
from a test was to mutate process state - and the commands did that to each other.

`runOrders`, `runPortfolio`, and `runConfig` handed a subcommand's arguments down
by writing `os.Args = append(os.Args[:2], args[1:]...)`. That `append` writes into
the backing array `os.Args` itself points at, so dispatching a command **mutated
the real argument vector**. Two commands in one process corrupted each other, and
the test harness reads that same array. This is the same defect class as
`S1025` in the SDK: the process environment is global mutable state, and tests
cannot be written against it without being arranged around it.

Coverage was 62.0% while the floor sat at 58%, because the floor had been ratcheted
up in steps (35 -> 48 -> 58) against a suite that could not grow. Raising it is
only honest once the untestable package can actually be tested.

## Order, and why

1. **The environment seam first.** Every other step is unreachable without it.
2. **The tests second**, against the seam - not a fake client, the real dispatch.
3. **The floor last.** The number must come out of the measurement, not be picked
   first and justified afterwards.

## Constraints held

- Public API unchanged: `pkg/ibkr` and the module's exported surface were not
  touched, only `cmd/ibkr`, which is a binary.
- No new dependencies.
- `client/*.gen.go` untouched; the spec and `scripts/patch_spec.py` were not
  involved, so `codegen-verify` has no reason to move.
