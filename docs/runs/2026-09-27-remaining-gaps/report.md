# Report: close the remaining engineering gaps

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `ddaf9b5` (`v1.1.12`)
- **Scope**: the three items the previous run's `next-phase.md` listed as
  remaining, plus the live defect the classification turned up.

## The float32 classification found a live bug

The open question was whether the 198 monetary `float32` fields in the generated
client were worth retyping. The answer changed the plan.

An earlier count of 198 was a loose regex. A tighter one found **44 unique field
names across 31 structs**. Classifying each by what the mock fixtures actually put
on the wire gave:

- **0** arrive as a bare JSON number
- **11** appear quoted
- **33** have no fixture evidence at all

So the case for a mass retyping largely dissolved: there was no evidenced bare
number left to fix. But name-based matching is unreliable — `balance` appears in
several structs — so the question that actually mattered was narrower: **is the
struct decoded by production code at all?**

31 structs hold a monetary `float32`. **25 are never decoded by production**; the
SDK reads them through `rawToString`, which uses `json.Number`. Of the 6 that
matched, three turned out to be local SDK types shadowing the generated names.

That left **three**, and all three are request types:

| Type | Field | Filled from |
|------|-------|-------------|
| `DepositFundsInstruction` | `amount` | `strToDecimal(req.Amount)` |
| `WithdrawFundsInstruction` | `amount` | `strToDecimal(req.Amount)` |
| `InternalCashTransferInstruction` | `amount` | `strToDecimal(req.Amount)` |

The spec declares each as `type: number, multipleOf: 0.01`, so the wire really is
a JSON number and `string` would be wrong to send. But the caller's amount arrives
as a decimal string and was parsed into a float32 — so **a caller moving
`"12345678.91"` put `12345679` on the wire**. That is the v1.1.9 tax-voucher defect
in the request direction, on three money-moving operations.

Fixed by extending `patch_spec.py`'s `NUMBER_MONEY_SCHEMAS` to the three request
types (`number_money_patched` 4 → 7) and replacing the parse with `moneyToNumber`,
which emits a `json.Number`: a bare number as the spec describes, carrying the
caller's digits.

Confirmed by reverting `moneyToNumber` to a float32 parse:

```
rest_banking_amount_test.go:63: amount on the wire = 12345679;
    want 12345678.91 - the caller's digits must survive unchanged
```

The three existing amount assertions used `250.75`, `500.25` and `1000.50` — all
exactly float32-representable, so they passed whether or not the amount had been
rounded. They now read the raw token and compare the literal. A new test drives a
value float32 actually corrupts, for both deposit and withdraw, and
`moneyToNumber` maps an empty amount to `0` because a zero-value `json.Number` is
the empty string, which `encoding/json` refuses to marshal.

## Coverage: 58.9% → 62.0% for pkg/ibkr, 60.0% → 62.0% overall

The three weakest files held 152 uncovered statements, and the cause was not
partially-covered code — it was **17 `ModelManager` methods with no test at all**.
The four existing model tests covered rebalance, invest/divest, the cash analyzer
and the portfolio-order collision; the entire model-portfolio *configuration* API
was never exercised.

Eleven are now covered: `ModelPresets`, `SetModelPresets`, `AccountsInModel`,
`SetAccountInvestmentInModel`, `InvestedAccountsInModel`, `AllModels`,
`ModelsPager`, `AllModelPositions`, `ModelSummarySingle`, `SetModelTargetPositions`
and `SubmitModelOrders`.

One is a cross-check rather than a smoke test: `ModelsPager` must agree with
`AllModels`, which it wraps. The models example was migrated to the pager in
v1.1.9, so a divergence there is user-visible.

Three of the new tests were mutation-checked — dropping the preset accounts,
sending the wrong model name, and routing the amount through a float — and each
was caught with a specific diagnostic. A fourth mutation attempt of mine was inert
(`_ = accts` changes nothing) and is recorded here because the first result would
otherwise have looked like a test gap.

`trading_accounts.go` and `notifications.go` turned out to be *better* covered than
the percentage suggested: their gaps are the `if err != nil` returns of functions
that are otherwise tested. Left alone — chasing them would have meant tests for
their own sake.

## The `unused` blind spot, closed without a hand-rolled checker

`run.tests: true` makes `unused` count a function as used when only a test
references it. That is how `float32ToStr` — a lossy money formatter — survived a
release, held open by a test that asserted the rounding it caused.

The obvious fix is a custom static-analysis script, which would be a new thing to
maintain and a new source of false positives. Instead: `golangci-lint run
--tests=false`, using the linter that already knows. It found **3** findings with
no false positives, all in one example.

Two were worth keeping and are now wired into the example's flow:
`handleAuthError` and `isAuthError` demonstrate recovering from an expired token,
and step 4 of that example previously just printed the error. `httpStatusCheck`
was deleted: it took a raw `*http.Response`, which the SDK never returns — a
reader copying it would reach for something they do not get, when `Error.HTTPStatus`
is the actual mechanism. The deletion also broke the example's own test, which was
the only remaining reference — the blind spot visible in the act.

CI now runs both passes, so the class is closed rather than the instance.

## Verification

| Check | Result |
|-------|--------|
| `golangci-lint run` | 0 issues |
| `golangci-lint run --tests=false` | 0 issues |
| `go build ./...` / `go vet ./...` / `gofmt -l .` | clean |
| `go test ./...` | pass |
| `go test -race` | clean |
| coverage | **62.0%** (was 60.0%) |
| `check_money.py` | OK, 55 files |
| `check_design` | 9 of 9 |
| codegen drift | 0 lines |
| links / i18n / SPDX | pass |

## Not addressed

- **`cmd/ibkr`'s six remaining subcommands** are still untestable — they read
  `os.Args` and build a live client. Left for a dedicated run: threading args,
  writers and a client-factory seam through six files is a refactor, not a
  finding.
- **The 33 unevidenced monetary `float32` fields** are untouched. They are in
  structs production never decodes, and guessing whether the gateway quotes them
  would be exactly the kind of change that breaks a decode. The v1.1.11 gate stops
  one reaching a caller through a float in any case.
