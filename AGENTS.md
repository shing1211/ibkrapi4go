# AGENTS.md

Guidance for automated coding agents and AI assistants working in this
repository.

## Project

- **Module**: `github.com/shing1211/ibkrapi4go`
- **Public package**: `pkg/ibkr`
- **Generated code**: `client/*.gen.go` — **never edit by hand**
- **Go version**: 1.26+
- **License**: Apache-2.0. Every new source file needs the header:
  ```go
  // Copyright 2026 shing1211
  // SPDX-License-Identifier: Apache-2.0
  ```

## Commands

```bash
make help            # list targets
make check           # gofmt + go vet + tests
make codegen         # fetch/patch spec, regenerate client/
make codegen-verify  # fail if generated code differs from committed
make license         # apply SPDX headers
make license-check   # verify SPDX headers
make docs-check      # check markdown links + README translations
make mock-gateway    # run the standalone mock IBKR gateway
```

## Hard rules

1. **Do not edit `client/*.gen.go`.** Change `scripts/patch_spec.py` (spec
   defects) and regenerate. Committed generated files must match
   `make codegen-verify`.
   > **Note:** All codegen defects are fixed at the spec level, so no
   > post-generation step is needed. In particular, the nil-`interface{}` panic
   > class is fixed by `scripts/patch_spec.py` defect 4 (inline query params with
   > `type: null` are retyped to `type: string`). `scripts/patch_gen.py` is a
   > no-op kept for backward compatibility.
2. **Never auto-retry order mutations.** See
   [ADR 0009](./docs/adr/0009-no-auto-retry-orders.md).
3. **Money and quantities are `string`/`json.Number`, never `float64`.** See
   [ADR 0008](./docs/adr/0008-numeric-precision.md).
4. **No new dependencies without an ADR.** The dependency set is intentionally
   minimal. See [ADR 0004](./docs/adr/0004-minimal-dependencies.md).
5. **Do not commit secrets.** Credentials come from the environment; tokens are
   in-memory only.
6. **One canonical count.** Endpoint/schema numbers come from
   `docs/SPEC.md`. Do not hand-edit counts elsewhere.
7. **No dangling doc links.** If you add a reference to a doc, create it.
8. **README translations stay in lockstep.** Update the switcher in **every**
   `README*.md` and the `Last synced:` banner; run `make docs-check`. English is
   canonical. See [TRANSLATING.md](./TRANSLATING.md).

## Where things live

| Area | Location |
|------|----------|
| Endpoint index (canonical) | `docs/SPEC.md` |
| Design decisions | `docs/adr/` |
| Module contracts | `docs/design/` |
| Spec patching | `scripts/patch_spec.py` |
| Codegen | `scripts/codegen.sh`, `oapi-codegen.yaml` |
| Mock gateway | `internal/mockgateway/`, `cmd/ibkr-mock-gateway/`, `docs/MOCK-GATEWAY.md` |
| Translations | `README.*.md`, `TRANSLATING.md`, `scripts/check_i18n.py` |

## Before opening a PR

- `make check` passes.
- `make codegen-verify` passes (if you touched spec handling).
- Docs updated; numbers match `docs/SPEC.md`.
- Commit messages use Conventional Commits and are DCO-signed (`git commit -s`).
