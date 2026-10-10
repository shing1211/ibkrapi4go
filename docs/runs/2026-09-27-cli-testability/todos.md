# Todos

- [x] Add an explicit `env` to `cmd/ibkr` carrying argv, writers, and a client
      factory, so no subcommand reads the process.
- [x] Convert `accounts`, `positions`, `orders`, `stream`, `portfolio`, `config`
      off `os.Args` / `os.Stdout`.
- [x] Delete the three `append(os.Args[:2], ...)` hand-offs.
- [x] Extract `dispatch(e, cmd)` from `run` so the routing table is reachable
      without going through argv parsing.
- [x] Test each leaf's help path, which must never reach a gateway.
- [x] Test each parent's unknown-subcommand error.
- [x] Test that the CLI never mutates the process argv - the regression test for
      the defect this run removed.
- [x] Test that a subcommand's own flags reach its validator.
- [x] Test consecutive flags in one command, so the value-skip cannot regress.
- [x] Delete `(*env).realStdout`, dead code the new `--tests=false` gate caught
      in the code written minutes earlier.
- [x] Resolve the two `gosec` G602 findings structurally rather than with
      `//nolint`, which silenced one lint profile and tripped `nolintlint` in the
      other.
- [x] Fix the `misspell` hit on a test fixture string.
- [x] Raise the CI coverage floor 58 -> 60, measured after the new tests.
- [x] Full verification: build, vet, gofmt, tests, race, both lint profiles,
      money, design, links, i18n, spec version.
- [ ] Commit, tag `v1.1.14`, push - **held for explicit approval**.
