# Plan: close the float32 money class

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `04162cb` (`v1.1.10`)

## Objective

v1.1.9 fixed the tax voucher's `float32` money fields — four fields on one DTO.
The same class of defect is reachable across 198 monetary `float32` fields in the
generated client, and nothing prevented the next occurrence. Turn the one-off fix
into a closed class.

## Approach

1. **Measure before fixing.** Establish how many `float32` fields exist, and —
      critically — whether any actually reach a caller as money. The answer
      decides whether this is a bug fix or a guard, and the report has to say
      which. Claiming a save that did not happen would be worse than doing
      nothing.
2. **Fix the gate, not the symptoms.** Retyping 198 fields is a separate, much
      larger project. The durable fix is to make the *path* from a binary float to
      a money string impossible to write quietly.
3. **Prove it bites.** A new gate that has never been observed failing is
      decoration. Reintroduce each vector and confirm the specific diagnostic.
4. **Correct the docs, including the parts that were already wrong.**

## Scope boundary

In: the enforcement rule, the dead code it exposes, the documentation.

Out: retyping the 198 generated fields; auditing the remaining test sleeps;
closing the general `unused` blind spot. Each is recorded in `next-phase.md`.

## Tasks

| # | Task | Outcome |
|---|------|---------|
| 1 | Measure `float32` exposure | 426 fields, 198 monetary, none reaching a caller |
| 2 | Diagnose the gate's blind spot | declaration rule; the defect was in a body |
| 3 | Reject `strconv.FormatFloat` in production `pkg/ibkr` | done |
| 4 | Reject float→string helpers | done; also catches unused-but-present ones |
| 5 | Extend the script self-test | six cases |
| 6 | Confirm both vectors caught | done, with diagnostics |
| 7 | Delete `float32ToStr` and its test | done |
| 8 | Correct `07-money-and-numbers.md` | done; it misdescribed the gate twice |

## Ordering note

The dead code came *after* the gate, not before. Deleting `float32ToStr` first
would have left the tree green and the class wide open — the point of the gate is
that it fails whether or not anything calls the helper.
