# Next phase

## Blocked on a live account

- **D15 - `year` on `listTaxDocumentsAvailable`.** The client half is covered: the
  parameter is asserted absent from the wire. One live call confirms the gateway
  accepts it. Read-only endpoint, low risk, simply unreachable from here.

- **`submitModelPortfolioOrder` collision.** The SDK implements and tests the
  operation. What an FA-enabled paper account would confirm is that the real gateway
  exposes it at the same path as `submitNewOrder` - the assumption behind keeping it
  unrouted in the mock. `TestEveryOpIDIsRoutedOrExplained` enforces that decision, so
  changing it takes a deliberate deletion of the exception.

## Available next, and worth being honest about the value

The substantive offline backlog is **empty**. What is left is small, and the
distinction matters:

- **`options.go` at 25%** - seven one-line functional setters. Nearly free, and
  explicitly not insightful. It would move the number and prove nothing.
- **`stream.go` at 73%** - `closeAll` and `Broadcast`, which govern what a client
  still receives after cancelling its own subscription. That is real behaviour, and
  it is the most interesting remaining gap in the mock gateway.
- **`cmd/ibkr` at 33%** - the largest number, and the least tractable: every non-help
  path builds a live client, so covering it means a design change, not a test.

Chasing the percentage now would be the wrong instinct. The three previous runs each
found something coverage could not, and this one closed a class rather than adding
tests for their own sake.

## Two decisions still outstanding, both one line each

- **The coverage floor** has moved 58 -> 60 -> 62 across three releases while
  coverage went 63.5% -> 66.3%, making it a function of release timing. Either pin
  it and let coverage rise, or commit to ratcheting per batch. It is a number in
  `ci.yml` plus a comment saying which.
- **The `check_money.py` precision gate** stays withdrawn. Two formulations were
  measured and both were unsound - 27 false positives, then 16 false negatives. A
  sound version has to resolve the *type* of the field under assertion by following
  the `toPublic` mappings. Only worth writing if the class recurs.

## Process notes carried forward

- **An SDK op label is not the route it calls.** `Trade.ContractInfo` is labelled
  one thing and hits `getInstrumentInfo`. A fixture keyed off the label reads the
  default and returns plausible values, so only a distinctive assertion catches it.
- **Reuse the guard, not just the values.** `float32Loses` is now shared by both
  precision files, so a value a `float32` renders unchanged is rejected at authoring
  time in either.
- **Say when the backlog is empty.** Seven runs of the same prompt would be an
  argument for continuing to generate activity; the honest report is that the
  substantive offline work is done and the rest needs credentials or a decision.
