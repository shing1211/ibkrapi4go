# Next phase

## Needs a live account, and it gates the biggest open question

- **Does IBKR actually rate-limit auth endpoints, and at what rate?** ADR 0018 is
  Accepted with this open. The 1 rps default is a client-side precaution; the spec is
  silent, the mock imposes no limit, and no ADR states it. If auth is not specially
  limited, the right change is to **drop** the bucket rather than raise its rate, and
  `TestLimiter_AuthPathsAreSlow` goes with it. This is the same trip that would unblock
  D15, so it is worth doing once rather than twice.
- **D15** - `listTaxDocumentsAvailable` with `year` omitted.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Needs a decision

- **The `config` subcommand is at 0%.** `saveConfig`, `runConfigShow` and `runConfigSet`
  are all uncovered and they write the user's real config file. This is the true
  remaining `cmd/ibkr` gap - not `completion.go`, which sits at 100%, which is what the
  `v1.1.25` backlog wrongly claimed. It has the same smell this run just closed for
  pacing: an unexercised path that nothing measures.
- **The limiter construction guard.** `internal.NewLimiter` is only called when
  per-endpoint or global limiting is enabled, so disabling both silently removes auth
  pacing too. `WithAuthRateLimit` can relax pacing but not add it. Documented on the
  option; restructuring it changes behaviour for callers who disable limiting today.
- **`--rest` is inert and now says so.** `cfg.RestGatewayURL` is parsed, stored, printed
  and settable, and `pkg/ibkr` has no option to receive it. Either add
  `WithRestGatewayURL` or remove the flag, the field and the `ibkr config` output.
- **`internal/fake` - delete or rewire.** Unchanged for five runs. The in-package
  `testClock` covers what the fake would have been for; it is still on `ALLOWED`.
- **The coverage floor** has moved 58 -> 60 -> 62 across seven releases while coverage
  went 63.5% -> 69.6%. Pin it, or ratchet per batch. Deferred for nine runs.
- **`parseGlobalFlags` returns `cmdIdx` one past** the first non-flag, against its own
  doc comment and its test's comment. No production caller uses it. The only item on
  this list with no judgement in it.
- **`portfolio ledger` emits Go field names** (`NetLiquidationValue`) because
  `LedgerCurrency` has no json tags, unlike every other JSON command. Changes output a
  script may already parse.

## Available next, no credentials needed

- **`cmd/ibkr-mock-gateway` at 0%.** A thin `main` over `internal/mockgateway` (89.4%).
  The `cmd/ibkr` harness shows the pattern: build the package once, mount its handler.
- **`ibkrPrintln` and `truncate` at 66.7%.** Small, and both are on the paths the
  `v1.1.25` run added, so the gap is a branch or two each.
- **A wait-count assertion for the CLI layer.** This run's test covers the SDK, not
  `newClientFromArgs`. Probably not worth it - the CLI's subcommand tests already cover
  that wiring - but recorded so the omission is a decision rather than an oversight.

## Process notes carried forward

- **Check for an existing signal before adding machinery.** This test needed no new
  infrastructure at all: `ibkr.ratelimit.waits` and `InMemoryMetrics` already existed. I
  had planned to hand-roll a metrics sink before checking whether one was there.
- **A counter is not a clock.** A wall-clock gate is flaky and a flaky gate gets muted.
  This repository has retracted `check_money.py` twice for that reason. Asserting on an
  exact counter is both stricter and cheaper.
- **Verify which case catches the regression, not which case should.** I asserted the
  opted-out case would catch the logout defect. It does not - a burst of 10 absorbs both
  auth calls. Only running the mutation revealed it, and a "passing" gate that catches
  nothing is worse than none.
- **Check the metric's attributes before keying a snapshot.** `SeriesKey(name)` with no
  attributes matches only observations that carried none. `MetricHTTPRequests` carries
  three, so the obvious assertion silently matches nothing - and the tempting fix is to
  loosen it until it passes.
- **Re-run the mutation after editing the test.** The comment and ordering changes came
  after the first mutation run, so the mutation was re-verified to confirm the pass was
  not an artefact of them.
- **Assert on the flow, not the helper.** `TestLimiter_LogoutIsNotAuthPaced` proved the
  classification; this run's test proves the real client takes that path. The defect
  existed because only the first kind of test existed.
