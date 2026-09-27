# Design Checkers and Residual Gaps - Report

Status: complete. `check_design` went from verifying **2 of 9** design documents
to **8 of 9**, and from running only when a maintainer typed `make docs-check`
to running on **every push**. No production code changed.

## The finding that reordered the run

The plan was to widen the checker from 2 documents to 9. Verifying the premise
first turned up something more urgent:

| Claim | Verified |
|---|---|
| `check_design` is not run by CI | **true** — `ci.yml` has `setup-go` in five places and never invokes it |
| `check_design` is not in `make check` | **true** — `Makefile:46` is `check: fmt vet money-check test`; it appears only at `Makefile:63` inside `docs-check` |
| The `lint & security` job is red on `main` | **true**, and far worse than first reported |

That last one needed correcting in both directions. I first said the lint job
reports "6 pre-existing issues". That figure was an artifact of golangci-lint's
default caps — `max-issues-per-linter: 50` and `max-same-issues: 3` are unset, so
repeated gosec findings were truncated.

| | Count |
|---|---|
| What CI displays | 154 |
| Uncapped, same package set | **742** |
| In `pkg/` + `internal/` + `cmd/` | **733** |

And `.golangci.yml` **fails schema validation**: a v1 file carrying `version: 2`,
whose `linters-settings` and `issues.exclude-*` keys are all rejected.
golangci-lint silently discards the whole `issues` block, so the author's intent —
revive limited to the `exported` rule, `exclude-rules` silencing errcheck in
`_test.go` — has never applied. That is why 45 test-file errcheck findings appear
in a security gate.

Widening a gate nothing runs would have multiplied an unrun checker, so the human
chose to enforce first, then widen. The lint state was documented and left
unchanged: the only migration that validates would *activate* those exclusions and
silently drop roughly 95 findings from a security gate, which is a reviewed
decision rather than a hygiene fix.

## Delivered

| Task | What | Result |
|------|------|--------|
| S2 | `check_design` for `06-errors-retries.md` | 6 checks: `Error` field set, `RetryPolicy` defaults, safe-method set, the ADR 0009 no-retry rule, the breaker default, the public alias |
| S2b | `RetryPolicy.Metrics` documented | Resolved the doc/code disagreement in the code's favour, as the human directed |
| S2c | `check_design` in CI, and in `make check` | New `design-check` target; `check` is now `fmt vet money-check design-check test`. Workflow parsed with PyYAML and structurally asserted, not eyeballed |
| S2d | 6 lint findings in `scripts/` | **8 → 0** uncapped in tracked code. Every `//nolint` carries a site-specific reason; exactly 4 added, because `nolintlint` makes a gratuitous one an error |
| S1 | SPDX headers on the two shell scripts | `addlicense -check` exits 0, verified live with a control |
| S3 | `check_design` for `09-orders-and-confirmation.md` | 6 checks: flow endpoints, the public order API, the hand-built wire body, explicit confirmation, no-retry routing, error mapping |
| S3b | `Reply` and `OrderRequest` enforced both directions | Closed a hole S3 had deliberately left, after the disagreement was resolved |
| S4 | `check_design` for `02`, `04`, `05`, `08` | 12 checks: client composition, import boundary, exported-type boundary, streaming surface and limits, redaction, insecure-skip-verify warning, OAuth generations |

Final state: **8 documents verified**, 8 of 9. Only `07-money-and-numbers.md` is
unverified, deliberately — `check_money.py` already enforces its main claim, so a
second checker there would duplicate a live gate.

## Document corrections

Five false statements in design documents, all found by the checkers and all
resolved in the code's favour, which is the precedence the human set twice:

- **`09-orders-and-confirmation.md`** documented `Reply.Message string` where the
  code declares `Messages []string` (`pkg/ibkr/trade.go:100`). A caller following
  the document got a **compile error**. Its `OrderRequest` block also listed
  `TimeInForce` before `StopPrice` and omitted `ParentID` and `IsSingleGroup`.
- **`02-client.md`** documented `WithOAuth2JWTKey(key []byte)` and
  `WithOAuth2JWTKeyPath(path string)`. Neither compiles: the real signature takes
  `*rsa.PrivateKey`, and the path form is `WithOAuth2JWTKeyFile`. Its struct
  completeness note also claimed to omit two fields the block lists directly
  above it.
- **`06-errors-retries.md`** omitted `RetryPolicy.Metrics`, which the struct
  declares and which drives real retry metrics.

## Proof discipline

Every check was observed failing, and the harness that proves it grew from
nothing to **232 subtests** across five suites, with **7 top-level control
tests**. Three details worth recording:

- **Negative controls.** Two of them replace a strict branch with a naive one and
  assert the *specific* cases go red — so a future edit cannot quietly weaken a
  check. Another proves the harness itself is not inert by breaking two documents
  at once and requiring both messages from one run.
- **A harness caught a real reader bug.** `checkClientOptions` originally read and
  filtered option lines in one pass, so `WithStreamingLimits(StreamingLimits)`
  would have been silently skipped. Found, fixed to two passes, and pinned.
- **A wrong-reason catch.** Re-anchoring a case after the note edit initially
  substituted one backticked name where the new note has only one, so the check
  reported the *real* `release` field instead of the bogus one. Fixed by adding
  the bogus name rather than replacing, with the expectation tightened to name it.

## Process lessons

Sub-agents corrected the orchestrator's briefs **eleven times** across the three
runs in this series, and every correction was right. The recurring ones:

- Claims about the brief that did not exist in the code (an assertion the brief
  said was already present; a caller count off by four; a field declaring one key
  where the brief said two)
- A hazard class under-specified by roughly 100 sites
- An orchestrator figure that was a tooling artifact rather than a fact — the
  "6 lint issues" claim

The pattern held and sharpened: **the brief's incidental claims need the same
independent check as its main deliverable.** Two agents also reported that their
briefs were self-contradictory and chose to hand over red tests rather than green
ones that could not fail. Both times that refusal is what surfaced the real
problem instead of baselining it away.

## Verification

- `go run ./scripts/check_design` — 8 documents verified
- `go test ./scripts/check_design/ -count=1` — 232 subtests, all pass
- `go test ./... -count=1` — pass
- `go test -race ./internal/... ./pkg/ibkr/...` — pass
- `gofmt -s -l .`, `go vet ./...` — clean
- `golangci-lint` over `./scripts/...` uncapped — **0 issues** (was 8)
- `check_money`, `check_links`, `check_i18n`, `check_spec_version` — pass
- Coverage 59.0% against a 58% floor, 1.0-point margin. The floor was not moved:
  this run added no production statements, so there is nothing new to ratchet to.

## Known limitations

- `07-money-and-numbers.md` is unverified by `check_design`, deliberately,
  because `check_money.py` covers its main claim. Its other claims — that the
  `amounts` helpers do not yet exist — are not checked by anything.
- The `lint & security` job remains red with 742 uncapped findings, 733 of them in
  library code, and `.golangci.yml` remains schema-invalid. Documented, unchanged.
- `license-check` is still not a CI step, which is why the two missing SPDX
  headers survived.
- Six design-document claims are recorded as disagreements and left unenforced by
  the specific check that found them, so a human can decide authority. All are
  listed in `next-phase.md` with both sides quoted in the checkers' header
  comments.
- `02-client.md`'s Options list is curated and omits seven exported options. The
  reverse direction is not enforced because the document makes no exhaustiveness
  claim.
