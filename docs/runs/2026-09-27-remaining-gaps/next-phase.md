# Next phase

## Worth doing

### Make `cmd/ibkr`'s six remaining subcommands testable

`runAccounts`, `runOrders`, `runPortfolio`, `runStream`, `runPositions` and
`runConfig` still read `os.Args` and write to `os.Stdout`, and each builds a live
client, so none is reachable from a test. `run`, `parseGlobalFlags` and
`runCompletion` were parameterised in v1.1.9; the same treatment plus a
client-factory seam would make their argument validation testable without a
gateway. It is a refactor across six files rather than a finding, which is why it
did not belong in this run.

### Decide the coverage floor now that there is headroom

Coverage is 62.0% against a 58% floor. The floor was ratcheted twice by hand
during earlier runs; with four points of margin it can be raised to 60% so the
next slow drift is caught earlier. Raising it is a policy decision, and the honest
reason to raise it now is that the current figure is real rather than inflated by
idle waiting — the v1.1.12 audit removed several seconds of `time.Sleep` that had
been incidentally executing background goroutines.

## Settled, no action

- **The monetary `float32` fields are resolved.** 31 generated structs hold one;
  production decodes none of 25 of them, and the 3 that matter — deposit, withdraw
  and internal cash transfer amounts — are now `json.Number` and carry the
  caller's digits. The remaining 33 unevidenced fields sit in structs the SDK never
  decodes, and the v1.1.11 gate stops any of them reaching a caller through a
  float.
- **`unused` can no longer be fooled by a test.** CI runs a second lint pass with
  test files excluded. It found 3 real cases; `float32ToStr` had been surviving
  the same way for a release.
- **The model-portfolio configuration API has tests.** Eleven `ModelManager`
  methods had none at all.

## Still blocked, needs an account

- **D15** — one call to `listTaxDocumentsAvailable` with `year` omitted confirms
  the v1.1.8 spec fix against a real gateway.
- **Mutating model-endpoint dispatch** needs an FA-enabled paper account.

## Deliberately deferred, with the reason

- **`contextcheck` for the WebSocket layer** stays disabled. Satisfying it means a
  `ctx` parameter on the public `Subscription.Close()`, which is a breaking change
  and belongs in a minor version.
- **The error branches in `trading_accounts.go` and `notifications.go`** are the
  only gap in those files. Their functions are otherwise tested, and covering an
  `if err != nil` for its own sake produces tests that assert the language.
