# Run: banking wire contracts and spec-level numeric fixes

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `e741ba1` (`v1.1.7`)
- **Scope**: the five unconfirmed contracts (D10–D15) carried out of
  `2026-09-26-payload-contracts` and `2026-09-27-design-checkers`, plus the two
  CI gates that existed as Make targets but were never enforced.

## Goal

Settle every remaining item flagged as "needs live-gateway evidence" without a
live gateway, by reading the checked-in spec and the generated client, and fix
what the code contradicts. Do the coding first; batch configuration and
verification at the end.

## Findings that changed the plan

Two of the six open questions were not open at all — they were decidable from
the committed spec, and one of them had been framed backwards:

- **D11 was inverted.** The generated `TradingInstrumentV2.Quantity` is
  `float32`, so the *spec* puts `quantity` on the wire as a JSON **number**.
  ADR 0008 governs the SDK's *public* types, not the wire type. The single-item
  path was already spec-compliant; the **bulk** path hand-rolled a local struct
  with `Quantity string` and sent a string. The bulk path was the bug.
- **D10 was a plain omission.** `CancelInstruction` already carries
  `Reason string` and both cancel paths use the same generated type. The bulk
  path simply never set it, so a caller's reason was silently dropped.
- **D12 was already correct.** `Positions` is marked "For V2 only", so
  `Quantity` is the V1 field and the V2 paths are right to ignore it.

## Tasks

| # | Item | Action | Evidence |
|---|------|--------|----------|
| D10 | bulk cancel dropped `reason` | populate `Reason` from the caller's request | mutation-proven |
| D11 | bulk V2 sent `quantity` as a string | emit a JSON number via `strToDecimal`, matching the single path and the spec | mutation-proven |
| D12 | top-level `Quantity` looked wrong | document it as the V1 field; no behaviour change | code-authoritative |
| D14 | `float32` money on tax vouchers | `patch_spec.py` defect 9 retypes the four fields to `json.Number`; new `jsonNumberToStr` helper | mutation-proven |
| D15 | forced `year=` on the available-years call | `patch_spec.py` defect 10 makes `year` optional for that one operation, so it is omitted | mutation-proven |
| CFG | `check_money.py` and `license-check` not enforced in CI | added to the `docs` job | both verified green |

## Why D14 needed a new patch kind

`patch_spec.py` defects 7 and 8 retype `type: number` to `type: string`. That is
correct for the money fields the gateway already quotes (`"balance":"1000.00"`,
`"buyingPower":"4000.00"` — verified in the fixtures). It is wrong for the tax
voucher, which the gateway sends as a bare number (`"divAmount":12.5`): Go cannot
unmarshal a number into a string field, so retyping would break the decode
outright rather than merely losing precision. Those fields need `json.Number`,
which accepts the number and keeps the gateway's own digits.

A `float32` mantissa is 24 bits, so it rounds anything above 2^24 (16777216).
A mutation check on the old code path produced `"12345679"` for an input of
`12345678.91` and `"33554432"` for `33554432.55` — the cents were gone with no
indication in the SDK's public string.

## Why D15 is a spec defect

`listTaxDocumentsAvailable` marks `TaxYearRequestParam` as `required: true`, so
`oapi-codegen` emitted a non-pointer `Year string` and the wrapper had no way to
omit it: calling the operation to *discover* which years exist sent a
present-and-empty `year=`. The requirement is incoherent — an endpoint that lists
the available years cannot require the caller to know one. `required` is a
sibling of `$ref`, so the parameter is inlined for this operation only, leaving
the shared component intact for operations that genuinely need a year.

## Verification

Every behavioural change carries a mutation check: the fix was reverted and the
new assertion was confirmed to fail with a diagnostic naming the defect, then
restored. The D14 test additionally asserts what float32 *would* have produced,
so the expectation cannot silently stop proving anything.

`make codegen-verify` reports zero drift, so the committed `client/` matches a
fresh generation from the committed spec.

## Not done, and why

- **`.golangci.yml` is still schema-invalid and the lint job still fails**
  (733 findings in library code). The prior decision for this workstream was to
  document the state and change nothing; migrating the config alters the
  security gate and needs its own decision, so it was left alone rather than
  folded in silently.
- **`cmd/ibkr` still has no tests.** The CLI's `os.Args` parsing needs a seam
  first; that is a refactor, not a config change, and it was not part of the
  contracts this run was chartered to settle.
- **No live-gateway verification.** D14 and D15 rest on the committed spec plus
  the gateway fixture shapes. D14's direction is confirmed by the fixtures (the
  voucher fields really are bare numbers). D15's direction is a judgement call
  about an incoherent requirement; if the real gateway in fact requires a year
  on that endpoint, the fix should be revisited.
