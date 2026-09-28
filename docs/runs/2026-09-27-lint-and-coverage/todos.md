# Todos: lint gate, CLI tests, and design-doc coverage

- [x] Diagnose `.golangci.yml`: v1 schema against a v2 binary, rejected outright
- [x] Migrate the config to the v2 schema and confirm it validates
- [x] Fix revive's `exported` rule, which failed to configure and silently
      reported nothing
- [x] Get a true finding count (192, not the recorded 733)
- [x] Fix `SubmitDocument` discarding its `mimeType` argument
- [x] Apply `defaultMaxResponseBytes` unconditionally
- [x] Make `MaxBytes` return `ErrResponseTooLarge` instead of truncating silently
- [x] Exempt 1xx upgrade responses from the cap, restoring WebSocket dialing
- [x] Add four `MaxBytes` tests, each mutation-checked
- [x] Replace the silent panic swallow in the subscription goroutine
- [x] Fix `errorlint`: the WebSocket resilience test's `==` made it vacuous
- [x] Remove the dead-code empty branch in `ws_test.go`
- [x] Delete 4 dead wire structs, 2 dead helpers, 1 dead test helper
- [x] Write doc comments for all 32 exported symbols revive flagged
- [x] Set `MinVersion` explicitly in the mock gateway's TLS config
- [x] Annotate the 4 G101 false positives and the G115/G304/G404 sites inline
- [x] Remove 9 redundant conversions; clear the 4 remaining staticcheck findings
- [x] Migrate the models example off the deprecated `AllModels`
- [x] Make the 33 unchecked `fmt.Fprintf` and 2 `json.Unmarshal` explicit
- [x] Document every exclusion in `.golangci.yml` with its reason
- [x] Make the design-checker red tests line-number-agnostic
- [x] Investigate the 193-vs-191 fixture count; confirm it is intentional
- [x] Extract `run(args, stdout, stderr)` and parameterise `parseGlobalFlags`
- [x] Parameterise `runCompletion`, which was reading `os.Args`
- [x] Add the first tests in `cmd/ibkr`
- [x] Add `checkMoneyNumberFields` for the ninth design document
- [x] Add 5 red cases plus a control for the new checker
- [x] Full verification and this run's artifacts

## Deliberately not done

- [ ] Satisfy `contextcheck` in the WebSocket layer - needs a `ctx` parameter on
      the public `Close()`, a breaking API change
- [ ] Fix the `TestWS_DialAndSubscribe` 500ms flake - loosening the window would
      mask a real delivery regression
- [ ] Confirm D15 against a live gateway (carried from v1.1.8)
