# Next phase

## Worth doing

### Retype the latent `float32` money fields in the generated client

198 monetary `float32` fields remain. None reaches a caller today, and after this
release none can. But the generated types are still wrong, and anything that
decodes through them and exposes a value inherits the rounding.

This is a project, not a patch, because the fix differs per field. The existing
`patch_spec.py` already splits the two cases: fields the gateway *quotes* become
`string` (defects 7 and 8), and fields it sends as a *bare number* need
`json.Number` (defect 9, added for the tax voucher). Deciding which of the 198 are
which needs per-field evidence, and the fixtures are the only source of that
evidence available offline.

Suggested first step: classify the 198 by what the mock fixtures actually contain
— quoted, bare number, or absent — and produce a report. Only then change the
spec.

### Audit the 21 remaining `time.Sleep` sites in tests

The one I fixed in v1.1.10 was a real defect: a sleep standing in for a wait on an
event that was already guaranteed. I did **not** verify the rest. The 250 ms sites
in `session_test.go` are the most suspicious. Cheap to audit, and it is the same
judgement call each time — is this sleep waiting for an event, or waiting for time
to pass? Only the first is a defect.

### Close the `unused` blind spot generally

`run.tests: true` makes `unused` count a function as used when only a test
references it. `float32ToStr` survived that way for a release, kept alive by a
test asserting the bug it caused. The money rule now covers the money shape;
nothing covers the general case. Options: set `tests: false` for the `unused`
linter specifically, or add a check for production symbols referenced only from
test files.

## Settled, no action

- **The float32 money path is closed.** `check_money.py` rejects both
  `strconv.FormatFloat` in production `pkg/ibkr` and any float→string helper
  there, confirmed by reintroducing each. The dead helper and its test are gone.
- **`float32ToStr` was dead code, not a live bug.** It had no production callers
  after v1.1.9, and its only test asserted the rounding loss it existed to cause.

## Still open from earlier runs

- **D15** (`listTaxDocumentsAvailable` no longer sends `year=`) needs one call
  against a live account.
- **Mutating model-endpoint dispatch** needs an FA-enabled paper account.
- **`contextcheck` for the WebSocket layer** is disabled rather than satisfied;
  making it pass means a breaking `Close(ctx)` signature.
- **`cmd/ibkr`'s six remaining subcommands** are still untestable - they read
  `os.Args` and build a live client.
