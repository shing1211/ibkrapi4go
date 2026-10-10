# Payload Contracts and Design Checks - Report

Status: complete. Coverage **58.9%** against a **58%** floor.

## What this run did

It began as the recommended phase from `2026-09-26-test-hardening/next-phase.md`
(P1 + P6 + P7 + P3) and grew a sixth item once a check that had been silently
inert for the life of the mock gateway was made real.

The P1 work was deliberately cut down. The previous run's own plan gated the
`rest_banking.go` wire-contract fixes (D10, D11, D12) on questions only a human
or a real gateway can answer, and this run did not guess at them. Instead N1
pinned the current payloads with byte-level tests, so whichever of those the
human later approves has a before and an after rather than a guess.

## Delivered

| Task | What | Result |
|------|------|--------|
| N1 | Characterised four banking payloads | 383 insertions, **0 deletions**. Exact key set and JSON type per key |
| N5 | Deleted three dead helpers | `strToDecimalPtr`, `makeTradingInstrumentRef`, `f32PtrToInt64Ptr`, plus an orphaned test helper |
| N6 | Corrected the inert `requests-status` fixture | `executedAt` -> `dateSubmitted`; a nil-expecting assertion inverted |
| N7a | Corrected the inert SSO fixture | `accessToken`/`tokenType` -> `access_token`/`token_type` |
| N7b | Dropped a wasted token fetch | `TradeConfirmations.ListAvailable` no longer acquires a token `internal.Auth` discards |
| N13-N15 | Made the fixture shape check real, then made it correct | 88 failures -> 12 false positives removed -> 0, with the check **stricter** than when it was naive |

### The shape check

`internal/mockgateway/shape_test.go` set `DisallowUnknownFields` on a decoder
whose target was `var js any`. `any` has no fields, so the option could never
fire and the function reduced to a JSON well-formedness test. It could not detect
the one class of defect it existed to catch, and that class was live twice: the
two inert fixtures above both passed it.

Making it real immediately flagged **88 fixtures**. That number was mostly wrong,
and being precise about why mattered more than being fast:

- The naive version compared every fixture against the operation's generated
  response type. But **113 of the 184 operations `pkg/ibkr` reaches are never
  decoded through that type** — production calls the bare generated method, gets
  the `*http.Response`, and `decodeJSON`s into its own struct. For those, the
  generated type is irrelevant and comparing against it is invalid.
- There were **two** bypass classes, not the one first identified: 17
  `json.Unmarshal(resp.Body, ...)` sites, and roughly 100 `decodeJSON(resp, ...)`
  sites reached through the non-`WithResponse` generated methods.
- The resolution is derived from `pkg/ibkr` call sites by `go/ast`, not from
  operation names. An op is exempt only when **every** wrapper reaching it
  decodes the raw body and none reads the generated type. 110 ops are now exempt
  with a counted reason and a `file:line`; 12 whose path cannot be established
  stay key-checked, because uncertainty should not produce a pass.

Final state: 180 passed, 4 skipped, **0 failed** of 191 fixtures, and the
enforcement got *tighter* than the naive version — key-checked against a single
response type rose from 50 to 59 ops, and per-key union checking from 1 to 2.
Both historical bugs are still caught, pinned by
`TestValidateShapeRejectsMismatchedKeys`.

## Findings recorded and not acted on

- **The `…_1` / `…_2` opId naming gap.** `fetchDividends_1`,
  `createExternalAssetTransfers_2` and three others cannot get a decode path at
  all, because oapi-codegen renames the method and the fixture layer files it
  under the old name. A codegen divergence, not a fixture bug.
- **Two spec/codegen divergences of the deferred `quantity` family.**
  `forecastMarketResponse.payout` is `number/double` in the spec but generates as
  `*string`; `orderCancelSuccess.order_id` is `integer/int64` while its sibling
  `orderSubmitSuccess.order_id` is `string`. The fixtures now match their
  generated types, but the spec inconsistency remains.
- **One item not delivered as specified.** N7b was asked for a test counting token
  acquisitions. Measuring first showed the count is 1 before and 1 after, because
  the redundant call hits the same cache the middleware uses and issues no
  separate fetch — so a count-based test cannot fail pre-fix. No such test was
  written. The removal's only effect is one fewer in-process cache hit, which is
  not externally observable.
- **`pkg/ibkr/forecast.go` declares a manager with no methods**, so five forecast
  operations are unreachable from the SDK. Their fixtures are now spec-correct
  regardless.

## Blocked, still awaiting a human

Unchanged from the previous run and still not started, because each changes what
goes on the wire for a money-adjacent operation:

- **D10** — `CancelInstructionsBulk` drops `Reason`. Does bulk-cancel accept one?
  N1 now pins that `reason` ships as `""` for every element.
- **D11** — V2 quantity is a JSON **number** on the single path and a **string**
  on the bulk path. Both cannot be right. ADR 0008 sides with the bulk path,
  since the single path routes a `float32` through `strToDecimal`, whose own
  comment concedes "Precision loss is accepted for now".
- **D12** — `AssetTransferRequest.Quantity` is ignored on both V2 paths.
- **D14/D15** — `float32` money precision and the forced `year=`, both needing a
  spec change plus `make codegen`.

## Process note

This run's sub-agents corrected the orchestrator's briefs **six** times, and every
correction was right: claimed assertions that did not exist, a caller count that
was 5 rather than 1, a field that declares one key rather than two, a framing
that said "query param" where it is a header, and a hazard class I had
under-specified by listing only the `json.Unmarshal(resp.Body)` sites and missing
the ~100 `decodeJSON` ones. The pattern from the previous run held and sharpened:
the brief's incidental claims need the same independent check as the main
deliverable.

One agent also correctly reported that its own brief was self-contradictory — told
not to change fixtures, not to suppress failures, and to leave the suite green,
with 88 pre-existing defects in the way — and chose to hand over a red test
rather than a green one that could not fail. That was the right call and it is
why the 88 got looked at properly instead of being baselined away.

## Verification

- `go test ./...` - pass
- `go test -race ./internal/... ./pkg/ibkr/...` - pass
- `go test ./pkg/ibkr/ -count=3` - pass, goleak stable
- `TestFixtureShapeConformance` - 180 passed, 4 skipped, **0 failed**
- `gofmt -s -l .`, `go vet ./...` - clean
- `check_money`, `check_links`, `check_i18n`, `check_spec_version` - pass
- Coverage 58.9%, floor 58%, 0.9-point margin
