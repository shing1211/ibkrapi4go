# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** Client half covered: the
  parameter is asserted absent from the wire. Needs one live call. Read-only.

- **`submitModelPortfolioOrder` collision.** The SDK implements and tests the
  operation. An FA-enabled paper account would confirm the gateway exposes it at
  the same path as `submitNewOrder`, which is the assumption behind keeping it
  unrouted in the mock.

## Available next, no credentials needed

- **Finish the `json.Number` money fields.** 39 are now covered by this run. The
  remaining `json.Number` fields are in `marketdata.go` (6) and `contract.go` (5)
  - strikes, option quotes, conid. Lower value than the money fields, but the same
  class, and `contract.go:244` is a `[]json.Number` slice which is a distinct decode
  target worth one test.

- **The tax-voucher `fee` assertion.** It uses `0.007`, which is lossy in binary but
  renders back as `"0.007"`, so it carries no proof. Not wrong, just weaker than it
  looks next to two assertions that do. One literal's change.

- **An automated gate for the class itself.** This run found the defect by
  scanning; the next one will not. `check_money.py` already owns the "no float money
  in production" rule and could equally own "no money assertion uses a value a
  `float32` renders unchanged" - a rule that would have caught both the v1.1.9 and
  v1.1.13 blind spots at authoring time. This is the highest-leverage item on the
  list and the only one that prevents recurrence rather than fixing one instance.

- **`options.go` at 25%** - seven one-line setters. Nearly free, and explicitly not
  worth doing for insight.

- **`cmd/ibkr` at 33%** - the largest gap, but every non-help path builds a live
  client, so covering it is a design change rather than a test.

## A decision that keeps being deferred

The coverage floor has moved 58 -> 60 -> 62 across three releases while coverage
went 63.5% -> 66.3%. That makes the floor a function of release timing rather than
of the code, and it will keep moving every time a run lands. Either pin it and let
coverage rise, or commit to ratcheting on every batch - the current churn is neither
and nobody has actually chosen. It is one line in `ci.yml` and a comment explaining
the intent.

## Process notes carried forward

- **A defect class with two prior instances in this repository is worth more than
  the last few points of coverage.** Three consecutive runs raised the percentage;
  this one found 39 unproven money fields. Coverage was never going to find that.
- **Check detectability, not representability.** `0.007` is lossy in binary and
  renders back identically, so no string assertion can catch it. An assertion
  "proving" precision with such a value is decorative, and it sat next to two real
  ones in an existing test.
- **Prefer a gate over a fix when the class recurs.** Two of these bugs were
  prevented by a rule, not by a test, and neither rule exists yet.
- **When a recollection about a language behaviour is load-bearing, probe it.** I
  was wrong that a quoted string cannot decode into `json.Number`; a ten-line
  probe settled what an argument would have muddled.
