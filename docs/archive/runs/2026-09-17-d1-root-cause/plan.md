# Plan: D1 Root-Cause Fix — Retype Null Query Params

- **Run:** `2026-09-17-d1-root-cause`
- **Mode:** BUILD
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `5857f03`
- **Feature commit:** `b94e677`

## Goal

Close the last open correctness item from the correctness-closure run: eliminate
the nil-`interface{}` panic class in generated request builders **at the spec
level**, rather than papering over it with post-generation guards.

Phase 9 (`2026-09-17-correctness-closure`) fixed the immediate panics by patching
`client/client.gen.go` with 12 nil guards, and later recorded that `patch_spec.py`
could not express the fix. `2026-09-17-*` next-phase docs listed the root cause as
unfixed (severity Low). This run fixes it properly.

## Root cause (confirmed)

`patch_spec.py` operates on the OpenAPI JSON. The IBKR spec declares 22 inline
query parameters whose schema **omits `type`** (OpenAPI `type: null`), e.g.
`GetContractInfo`'s `sectype`/`month`/`exchange`/`strike`/`right`/`filters` and
`GetConidsByExchange`'s `assetClass`. `oapi-codegen` maps a typeless schema to a
bare `interface{}` in the generated params struct. When such a field is nil,
`runtime.StyleParamWithOptions` dereferences a nil interface and panics.

The values behind these params are plain strings in every case, so the correct
mapping is `type: string`.

## Scope

- `scripts/patch_spec.py` — new defect 4: retype inline `type: null` query
  parameter schemas to `type: string`.
- `scripts/patch_gen.py` — retire to a documented no-op (root cause handled
  upstream).
- `scripts/codegen.sh`, `scripts/validate_codegen.sh` — stop invoking
  `patch_gen.py`.
- `client/client.gen.go` — regenerate from the patched spec.
- `pkg/ibkr/contract.go`, `pkg/ibkr/notifications.go` — adapt the two SDK
  callers whose parameter Go types changed (`assetClass` → typed enum;
  `enabled` → string).
- Living docs: `AGENTS.md`, `docs/CODEGEN.md`, `CHANGELOG.md`.

## Non-goals

- No public API signature changes (`GetTradingScheduleBySymbol`,
  `ModifyFYIEmails` keep their existing signatures).
- No new dependencies.
- No change to the upstream spec; the patch remains deterministic and idempotent.

## Task breakdown

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|------------|
| D1 | Enumerate all `type: null` query params and confirm typeless→`interface{}` | codegen | — | 22 params listed; root cause reproduced |
| D2 | Add `patch_spec.py` defect 4 (retype to `string`); retire `patch_gen.py` | codegen | D1 | `codegen.sh`/`validate_codegen.sh` updated; `patch_gen.py` no-op |
| D3 | Regenerate `client/client.gen.go`; adapt SDK callers | backend | D2 | `go build ./...` clean; guards gone; enum constants present |
| D4 | Verify: build, vet, tests, codegen-verify, money/docs/license checks | qa | D3 | all pass |
| D5 | Docs sync (`AGENTS.md`, `CODEGEN.md`, `CHANGELOG.md`) + release | docs + release | D4 | `make docs-check` pass; pushed to both remotes |
| D6 | Run docs + index + next-phase | orchestrator | D5 | `report.md`, index line, `next-phase.md` |

## Verification (global)

`make check` (gofmt/vet/money/tests) · `make codegen-verify` · `make docs-check` ·
`make license-check`
