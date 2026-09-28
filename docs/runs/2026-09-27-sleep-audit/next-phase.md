# Next phase

## Worth doing

### Restore coverage headroom

60.0% against a 58% floor is 2 points. One medium-sized feature lands and CI goes
red for an unrelated reason. The three weakest files are
`trading_accounts.go` (68.7%, 21 uncovered), `notifications.go` (73.4%, 25) and
`models.go` (83.9%, 23) — about 69 statements from real headroom.

### Retype the latent `float32` money fields in the generated client

198 monetary `float32` fields remain. None reaches a caller and the gate in
v1.1.11 stops one from doing so, but the generated types are still wrong. The
existing `patch_spec.py` splits the cases already: fields the gateway quotes become
`string`, fields it sends as bare numbers need `json.Number`. Deciding which of
the 198 are which needs per-field evidence, and the mock fixtures are the only
source available offline. Suggested first step: classify all 198 from the
fixtures and produce a report before changing any spec.

### Make `cmd/ibkr`'s remaining six subcommands testable

`runAccounts`, `runOrders`, `runPortfolio`, `runStream`, `runPositions` and
`runConfig` still read `os.Args` and write to `os.Stdout`, and each builds a live
client. Threading args and writers through, plus a client-factory seam, would make
their argument validation testable without a gateway.

## Settled, no action

- **The fixed-wait audit is complete.** 11 of 26 sites were defects and are now
  event-driven; the other 15 are legitimate and the reasoning is recorded in
  `2026-09-27-sleep-audit/report.md` so the next reader does not re-derive it.
- **`Session.Initialize` no longer sleeps a second before its first poll.** That
  was a real latency cost on every session start, found by noticing that the tests
  were slow for a reason the tests could not explain.
- **The float32 money path is closed** (v1.1.11).

## Still open from earlier runs

- **D15** (`listTaxDocumentsAvailable` no longer sends `year=`) needs one call
  against a live account.
- **Mutating model-endpoint dispatch** needs an FA-enabled paper account.
- **`contextcheck` for the WebSocket layer** is disabled rather than satisfied;
  making it pass means a breaking `Close(ctx)` signature.
- **The general `unused` blind spot** remains: `run.tests: true` still hides
  production code kept alive only by a test, outside the money shape that v1.1.11
  now covers.
