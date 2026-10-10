# Report: banking wire contracts and spec-level numeric fixes

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `e741ba1` (`v1.1.7`)
- **Status**: implementation complete, uncommitted

## What changed

Six items. Five were code or spec defects; two of the six had been described
backwards in the previous run's notes and are worth stating plainly.

### D10 — bulk cancel silently dropped the caller's reason

`CancelInstructionsBulk` built each `client.CancelInstruction` with only
`InstructionId`. The generated type has carried `Reason string` all along and the
single-cancel path sets it, so a caller cancelling instructions in bulk had its
reason discarded before it left the process. This is a money-moving operation, so
a missing reason is not cosmetic.

### D11 — the bulk path sent a JSON string where the spec says number

The previous run's notes said ADR 0008 "sides with the bulk path" on the
`quantity` wire type. That was wrong. ADR 0008 governs the SDK's *public* types;
the *wire* type comes from the spec, and the spec's
`TradingInstrumentV2.Quantity` is `float32` — i.e. a number. The single-item path
was spec-compliant. The bulk path declared its own local position struct with
`Quantity string` and sent a quoted string, so the bulk endpoint received a
different type for the same field than the single endpoint did.

The bulk path existed in that shape because the spec types the bulk body's
`instructions` as `[]interface{}`, so the author hand-rolled a struct. The fix
keeps the hand-rolled struct but drops the duplicated inline position type
(three copies collapsed into one named type) and converts with the same
`strToDecimal` the single path uses.

### D12 — `AssetTransferRequest.Quantity` was correct all along

`Positions` is marked "For V2 only", which makes the top-level `Quantity` the V1
single-instrument field. The V2 paths ignoring it is right. Documented rather
than changed.

### D14 — float32 was silently rounding tax-voucher money

`TaxVoucherDTO.divAmount`, `withHeldAmount`, `fee` and `quantity` generated as
`*float32` and were rendered with `strconv.FormatFloat(..., 32)`. Reverting the
fix reproduces the loss exactly:

```
amount         = "12345679"      (input 12345678.91)
withheldAmount = "33554432"      (input 33554432.55)
```

The caller's public string showed rounded values with nothing to indicate a loss.
The repo had already patched 41 float ID fields and 16 money fields for exactly
this class of bug; these four were missed because the existing retyping targets
`string`, which cannot work here.

### D15 — the available-years call sent `year=`

`listTaxDocumentsAvailable` required a tax year, so the generated field was
non-pointer and the wrapper sent `year=` with an empty value while asking which
years exist. Now omitted.

### CI — two existing gates were not enforced

`scripts/check_money.py` (the ADR 0008 guard) and `make license-check` both
existed but neither ran in CI. Both are verified green and now run in the `docs`
job, so a future float money field or a missing SPDX header fails the build.

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l .` | clean |
| `go test ./...` | pass |
| `go test -race ./pkg/ibkr/ ./internal/...` | clean |
| coverage | 58.9% (floor 58) |
| `check_money.py` | OK, 55 files |
| `check_design` | 8 docs verified |
| `check_links.py` | resolve |
| `check_i18n.py` | 6 languages consistent |
| `check_spec_version.py` | no drift |
| `addlicense -check` | exit 0 |
| codegen drift | 0 lines |

All five behavioural changes were mutation-checked: the fix was reverted, the
assertion was confirmed to fail naming the specific defect, and the fix was
restored.

## Files

```
.github/workflows/ci.yml                   |  6 +++
client/client.gen.go                       | 41 ++++++----   (regenerated)
docs/design/07-money-and-numbers.md        | 10 ++-
pkg/ibkr/rest.go                           | 23 ++++-
pkg/ibkr/rest_banking.go                   | 41 ++++++----
pkg/ibkr/rest_banking_e2e_test.go          | 54 +++++++-----
pkg/ibkr/rest_reports_e2e_test.go          | 12 ++---
pkg/ibkr/rest_taxvouchers_e2e_test.go      | 54 ++++++++++++
scripts/patch_spec.py                      | 70 ++++++++++
```

Four existing tests asserted the old buggy behaviour and were inverted rather
than deleted: the bulk-cancel reason, the bulk-V2 quantity JSON type, and the
tax-document `year` query parameter. Each previously read as documentation of
current behaviour, so the corrected contract is now pinned in their place.

## Not addressed

- `.golangci.yml` is still schema-invalid and CI's lint job still fails with 733
  library findings. Left unchanged per the standing decision for this
  workstream; migrating it changes the security gate and needs its own call.
- `cmd/ibkr` has no tests; it needs an `os.Args` parsing seam first.
- D15 has not been checked against a live gateway. D14's direction is confirmed
  by the fixtures; D15 is a judgement about an incoherent spec requirement and
  should be revisited if the real endpoint turns out to require a year.
