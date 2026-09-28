# Plan: the gate I recommended last turn, tested and withdrawn

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `22051aa` (`v1.1.18`)

## Objective

The previous run ended by recommending a specific piece of work: extend
`check_money.py` so that a money assertion using a value a `float32` renders
unchanged fails the build. The reasoning was prevention rather than a third
instance of the fix.

This run builds it. It does not ship.

## Why the gate was withdrawn

Two formulations were prototyped and measured before either was written into
`check_money.py`.

**Formulation 1 - lint the literals.** "In a test file, a decimal literal on a
line that references money and compares it must change under a `float32`."

It flags **27 literals, and almost all of them are wrong**:

```
endtoend_test.go    L142  100.25   if sum.TotalCashValue != "100.25" {
managers_e2e_test.go L80  49.875   positions[0].UnrealizedPnL != "49.875"
```

These are string comparisons on `string`-typed public fields. A `float32` regression
there is a **compile error**, not a silent round - the type does the work, and the
literal is irrelevant. Worse, the rule **missed the one real case**: `0.007` in the
tax-voucher fixture is not on a comparison line, so a comparison-scoped rule never
sees it. It is simultaneously too noisy and not catching the target.

**Formulation 2 - key off the type.** "Every monetary `json.Number` field in a `*Raw`
decode struct must have a test asserting it with a `float32`-changing value."

This is the right invariant - 25 such fields exist - but the implementation is
where it breaks. Identifying "has a lossy-value assertion" needs to follow consts
and table-driven loops. A line-based reading reported **16 fields as missing that
are in fact covered**, because the assertions are table-driven and the literal lives
in a `const` on a different line. A gate that reports 16 false negatives gives false
assurance while looking rigorous.

## Why this is the right call

A lint with 27 false positives trains people to ignore it. A lint with false
negatives gives false assurance. Both are worse than no gate, and shipping either
would have consumed a reviewer's attention while looking like progress. The
recommendation was reasonable and the implementation is not available cheaply; the
useful outcome is knowing that, before writing it.

## What this run delivers instead

The gap itself, completed and verified: the remaining 7 of 28 monetary
`json.Number` fields on the three responses now have `float32`-changing assertions,
each mutation-checked. `money_precision_test.go` is now exhaustive over those
fields, and says so in a comment so the next person knows the set is meant to be
complete.

## What would make a gate viable

Not this, but recorded so the next attempt starts further along:

- The rule needs to resolve the *type* of the field under assertion, not the literal
  on the line. That means following `toPublic` mappings and the raw struct
  definitions - effectively a small amount of real analysis.
- A cheaper, sound alternative that does not exist yet: assert in Go, at test time,
  that the set of monetary `json.Number` fields is non-empty and that each public
  counterpart is asserted. Reflection can enumerate the raw structs, but it cannot
  see what a test asserts, so this still needs the explicit list this run now has.
  The value of the explicit list is that it is a single place to look, not that it
  fails on its own.

## Constraints held

- No credentials used.
- No production behaviour changed.
- No public API change.
- `check_money.py` is **not** modified.
