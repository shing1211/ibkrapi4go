# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** Client half covered; the live
  call is not reachable from here. Read-only endpoint.

- **`submitModelPortfolioOrder` collision.** SDK implemented and tested. An
  FA-enabled paper account would confirm the gateway exposes it at the same path as
  `submitNewOrder`.

## Withdrawn, and why not to retry it naively

**The `check_money.py` precision gate.** Two formulations were prototyped. Scanning
literals produced 27 flags that are almost all false positives - string-typed
assertions, where a `float32` regression is a compile error - while missing the one
real case, which is on a fixture line. Keying off the `json.Number` type produced the
right invariant and 16 false negatives, because table-driven assertions keep their
literal in a `const` on another line.

Do not re-attempt either as a grep. A sound version must resolve the *type* of the
field being asserted, which means following the `toPublic` mappings and the raw
struct definitions. That is a small analysis, and it would be worth writing only if
the recurrence rate justifies it - two real bugs in this repository's history is the
whole case for it.

What exists instead is `pkg/ibkr/money_precision_test.go`, which is exhaustive over
the monetary `json.Number` fields of the three responses. It cannot fail on its own
when a field is added, but it is one place to look, and a new raw money field is
visibly absent from it.

## Available next, no credentials needed

- **The remaining `json.Number` fields outside money.** `marketdata.go` (6) and
  `contract.go` (5) carry `json.Number` for strikes, option quotes and conid. The
  same reasoning applies - `contract.go:244` is a `[]json.Number` slice, a distinct
  decode target - and neither has a precision test. Smaller than the money fields,
  but the last of the class.

- **The tax-voucher's `fee` assertion.** `0.007` is lossy in binary but renders back
  as `"0.007"`, so no string comparison can detect a regression there. Not wrong,
  just decorative next to two assertions that carry proof. One literal's change.

- **`options.go` at 25%** and **`cmd/ibkr` at 33%.** The first is seven one-line
  setters, nearly free and not insightful. The second needs a live client per
  subcommand, so it is a design change rather than a test.

## Still a decision, not a note

The coverage floor has moved 58 -> 60 -> 62 across three releases while coverage
went 63.5% -> 66.3%, making it a function of release timing. One line in `ci.yml`
plus a comment saying which policy it is.

## Process notes carried forward

- **A recommendation is a hypothesis.** Last turn's gate was stated confidently and
  was wrong on inspection. So were two premises in this run: that string-typed
  assertions were vacuous, and that the type-keyed rule was implementable as a scan.
  Measure a proposed rule against the tree before proposing it again.
- **27 false positives and 16 false negatives are two ways to ship nothing.** Both
  were found only by running the candidate rule over the real code. A rule that has
  never been run on the corpus is not a rule.
- **Prefer an explicit list you can read over a heuristic that guesses.** The
  precision test is exhaustive by hand and says so; the automated alternative is
  exhaustive only on paper.
