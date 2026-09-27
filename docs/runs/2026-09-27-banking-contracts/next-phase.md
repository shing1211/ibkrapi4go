# Next phase

## Needs a decision

### The lint gate is red and its config is invalid

`.golangci.yml` is written for golangci-lint v1 (`version: "2"`,
`linters-settings`, `issues.exclude-rules`) and is rejected outright by the v2
schema, so `golangci-lint run` never gets as far as analysing anything. The
library code carries 733 findings underneath that. `scripts/` was cleaned to zero
in the design-checkers run; `pkg/`, `internal/` and `cmd/` were not.

This is a policy question rather than a mechanical one. The v2 migration either
rewrites the config to the new schema, which starts enforcing rules that have
never run, or it moves the `lint` job to a pinned v1 binary, which keeps today's
posture. Those are different security postures and the choice should be made
explicitly, not inherited from a broken file.

Recommended: migrate the config to v2, then triage the findings by linter rather
than fixing all 733 at once, so each linter's true cost is visible before
deciding what to exclude.

### D15 against a live gateway

`listTaxDocumentsAvailable` requiring a tax year is incoherent — the endpoint
reports which years exist — so the SDK now omits the parameter. That is a
judgement about a spec defect, not an observation. Worth one call against a real
account to confirm the endpoint behaves the same with the parameter absent.

## Ready to pick up

### `cmd/ibkr` has no tests at all

The CLI is the only package with zero test files. `orders.go` parses `os.Args`
inline, so nothing is reachable without extracting a `run(args []string) error`
seam. That refactor is small and would let the arg validation, the exit codes and
the unknown-command path be tested. Coverage is 58.9% against a 58% floor, so
there is almost no headroom for the next change that adds an uncovered branch.

### Design-doc coverage is 8 of 9 documents

`check_design` verifies every design document except
`07-money-and-numbers.md`, which is covered by `check_money.py` instead. That
split is deliberate and documented, but it means the money rules are enforced by
one regex-style check rather than the claim-checking harness the other eight
documents get. The tax-voucher case added in this run is the kind of thing a
claim-checker would catch automatically: it knows the SDK exposes no `float`
money field, and could assert that every money field which exceeds float32's range
is carried as `json.Number` or `string`.

### Nine operations are wired but not shape-checked

The fixture shape checker covers 191 of 193 operations. Worth confirming whether
the remaining two are intentional exclusions or gaps.
