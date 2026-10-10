# Todos: banking wire contracts and spec-level numeric fixes

- [x] Read the committed spec and generated client for D10–D15 before changing
      anything, so the wire type comes from the spec rather than assumption
- [x] D10: populate `Reason` in `CancelInstructionsBulk`
- [x] D11: emit `quantity` as a JSON number in the bulk V2 path; collapse the
      triplicated inline position type
- [x] D12: document `AssetTransferRequest.Quantity` as the V1 field
- [x] D14: add `patch_spec.py` defect 9 (`json.Number` for number-on-the-wire
      money fields)
- [x] D14: add `jsonNumberToStr` and switch the four tax-voucher call sites
- [x] D14: add a precision test using a value float32 actually corrupts
- [x] D15: add `patch_spec.py` defect 10 (make `year` optional for
      `listTaxDocumentsAvailable`)
- [x] Regenerate `client/` and confirm zero codegen drift
- [x] Mutation-check every behavioural change
- [x] Invert the four tests that asserted the old behaviour
- [x] Add `check_money.py` and `license-check` to CI, verified green
- [x] Correct `07-money-and-numbers.md`, whose stated reason for `json.Number`
      no longer covered the float32-rounding case
- [x] Full verification: build, vet, gofmt, tests, race, coverage, all checkers
- [x] Write run artifacts

## Deliberately not done

- [ ] `.golangci.yml` v2 migration and the 733 library findings — standing
      decision is to document and leave alone; it changes the security gate
- [ ] `cmd/ibkr` test coverage — needs an `os.Args` parsing seam first
- [ ] Live-gateway confirmation of D15
