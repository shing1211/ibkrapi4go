# Todos: close the remaining engineering gaps

- [x] Classify the monetary `float32` fields by what the fixtures actually contain
- [x] Find the real production surface: 31 structs hold one, only 6 are referenced
- [x] Discover 3 bank-instruction request amounts were rounded on the way out
- [x] Extend `patch_spec.py` `NUMBER_MONEY_SCHEMAS` to the three request types
- [x] Replace the `strToDecimal` parse with `moneyToNumber`
- [x] Regenerate `client/`; confirm zero codegen drift
- [x] Strengthen the 3 amount assertions that used float32-exact values
- [x] Add a precision test proving the fix; mutation-checked
- [x] Cover the 11 untested `ModelManager` methods
- [x] Add a cross-check that `ModelsPager` agrees with `AllModels`
- [x] Mutation-check 3 of the new model tests
- [x] Close the `unused` blind spot with `golangci-lint run --tests=false`
- [x] Wire `handleAuthError` into the example it demonstrates
- [x] Delete `httpStatusCheck`, which taught a pattern the SDK makes obsolete
- [x] Add the `--tests=false` pass to CI
- [x] Full verification and this run's artifacts

## Deliberately not done

- [ ] `cmd/ibkr`'s six remaining subcommands: threading args, writers and a
      client-factory seam through six files is a refactor, not a finding.
- [ ] The 33 unevidenced monetary `float32` fields: they sit in structs
      production never decodes, and guessing whether the gateway quotes them would
      be the kind of change that breaks a decode.
- [ ] The error branches in `trading_accounts.go` and `notifications.go`: their
      functions are otherwise tested, and covering them would mean tests for their
      own sake.
