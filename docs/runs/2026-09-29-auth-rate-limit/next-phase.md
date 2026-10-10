# Next phase

## Needs a live account, and it is the biggest item here

- **Does IBKR actually rate-limit auth endpoints, and at what rate?** Recorded as an
  open question in ADR 0018. The 1 rps default is a client-side precaution; the spec is
  silent, the mock imposes nothing, and no ADR states it. If auth is not specially
  limited, the right change is to **drop** the bucket rather than raise its rate, and
  `TestLimiter_AuthPathsAreSlow` goes with it. This also unblocks the observation behind
  D15, so it is worth one trip to a real gateway.
- **D15** - `listTaxDocumentsAvailable` with `year` omitted.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Needs a decision

- **The limiter construction guard.** `internal.NewLimiter` is only called when
  per-endpoint or global limiting is enabled, so disabling both silently removes auth
  pacing too. Documented on `WithAuthRateLimit`; restructuring it changes behaviour for
  callers who disable limiting today, so it wants its own decision.
- **`--rest` is inert and now says so.** `cfg.RestGatewayURL` is parsed, stored, printed
  and settable, and `pkg/ibkr` has no option to receive it. Either add
  `WithRestGatewayURL` or remove the flag, the field and the `ibkr config` output.
  Leaving a commented dead assignment is honest but not finished.
- **`internal/fake` - delete or rewire.** Unchanged for four runs. The in-package
  `testClock` covers what the fake would have been for, and it is still on `ALLOWED`.
- **The coverage floor** has moved 58 -> 60 -> 62 across six releases while coverage
  went 63.5% -> 69.5%. Pin it, or ratchet per batch. Deferred for eight runs.
- **`parseGlobalFlags` returns `cmdIdx` one past** the first non-flag, against its own
  doc comment and its test's comment. No production caller uses it. Cheapest item on
  this list and the only one with no judgement in it.
- **`portfolio ledger` emits Go field names** (`NetLiquidationValue`) because
  `LedgerCurrency` has no json tags, unlike every other JSON command. Changes output a
  script may already parse.

## Available next, no credentials needed

- **The `config` subcommand is at 0%.** `saveConfig`, `runConfigShow` and `runConfigSet`
  are all uncovered, and they write the user's real config file. This is the true
  remaining `cmd/ibkr` gap - not `completion.go`, which is at 100%, which is what the
  `v1.1.25` backlog wrongly claimed.
- **`cmd/ibkr-mock-gateway` at 0%.** A thin `main` over `internal/mockgateway` (89.4%).
  The `cmd/ibkr` harness shows the pattern: build the package once, mount its handler.
- **A CLI latency check in CI.** This run found a 2s regression-class defect that no
  test, gate or linter would ever have caught, because nothing asserted on wall clock.
  A single test that an Initialize-then-Close cycle completes well inside a second
  would have caught it at any point in the last two releases.

## Process notes carried forward

- **Check `docs/` before claiming something is undocumented.** I told the user the 1 rps
  auth default had no recorded rationale. `docs/RATE-LIMITING.md` had the figure and
  the reason. I had grepped `docs/adr/` and `docs/design/` and stopped. That was the
  third wrong claim in this session, and all three came from asserting before checking.
- **A measured symptom with no cause is a lead, not a finding.** The previous backlog
  said "the 2.0s latency, cause unexplained" and it was treated as understood. It was
  not, and the cost of that was a whole run.
- **Separate a wait before a call from a wait after a failed one.** Phase timings said
  "1.0s in Initialize"; per-request timings with inter-request gaps said "0.998s before
  the request went out". The second measurement ruled out the poll loop that the first
  one pointed at.
- **Let the code name its own bug.** The client already logs `ibkr.ratelimit wait` with
  a duration. Four probes of reading and timing were replaced by one probe of listening.
- **Check the repo's own policy before escalating a version judgement.** Strict semver
  would call a new public option a minor; ADR 0015 line 69 lists "adding new exported
  symbols" as non-breaking. The repo's contract settles it, and it is citable.
- **When a change contradicts a doc, fix the doc in the same change.** `RATE-LIMITING.md`
  listed `/logout` as an auth path. Left alone it would have taught the next reader the
  opposite of what the code does.
