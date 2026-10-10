# Next phase

## Blocked on a live account

- **D15** - `listTaxDocumentsAvailable` with `year` omitted, against a real gateway.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Needs a decision

- **`internal/fake` - delete or rewire.** Now enforced as a visible `ALLOWED` entry in
  `scripts/check_internal_refs.py`, so the exception is printed on every run instead of
  being an oversight. Removing the package removes the entry; rewiring it means giving
  `internal.Clock` an exported constructor or making it an interface. On the evidence,
  the in-package `testClock` already covers what the fake was for.
- **The error budget has no time component.** `evict` is passed `now` and never reads
  it, so "5 failures in 60s" means "5 out of the last N". Changing when a trading client
  trips its breaker needs an ADR and a minor version.
- **`TestErrorBudget_EvictsOldEntries` is misnamed** - its body asserts entries are
  *not* evicted. One-line rename.
- **The coverage floor** has moved 58 -> 60 -> 62 across three releases while coverage
  went 63.5% -> 66.7%. Pin it, or commit to ratcheting per batch. Deferred for six
  runs.

## Available next, no credentials needed

- **The same class in other places.** `cmd/ibkr-mock-gateway` and `examples/` are
  shipped too, and the rule generalises: a `cmd/` binary nobody builds, or an example
  that no longer compiles against the SDK, is the same kind of rot. The
  `internal-refs-check` machinery would extend, though each needs its own judgement
  about what "used" means, which is exactly the part that made the money gate unsound.
- **`stream.go`'s remaining gap** - `serveWS` branches and the `handleSubscribe`
  family.
- **`cmd/ibkr` at 33%** - the largest number, and a design change: every non-help path
  builds a live client, so it needs a client-factory seam first.

## Process notes carried forward

- **A gate is only trustworthy once it has failed.** The self-test passed while the
  real scan was broken, twice, and only running the script against a real file showed
  it. Synthetic test strings without blank lines and with short paths hide exactly the
  shapes gofmt produces.
- **A probe that can emit invalid input makes a failing check meaningless.** The third
  "bug" in the checker was a missing closing quote in my own probe. Assert the probe's
  own input.
- **Exceptions should be loud.** `ALLOWED` entries print on every run, because the
  withdrawn money gate's problems started with exemptions nobody could see.
- **Measure the gate before recommending it.** The previous gate was proposed with a
  confident rationale and withdrawn on measurement. This one was measured first, which
  is why it survived - and why two bugs in it were caught.
