# Code Generation

The SDK's types and HTTP client are generated from the official IBKR OpenAPI
spec with [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen). This
document records the pipeline and the measured state of generation.

## Source spec

| Property | Value |
|----------|-------|
| URL | `https://api.ibkr.com/gw/api/v3/api-docs` |
| Title / version | `IB REST API` / `2.39.0` |
| OpenAPI | `3.0.0` |
| Downloaded size | ~900 KB |
| Paths | 171 |
| Operations | 185 |
| Schemas | 443 |
| Tags | 27 |

The spec is **not committed**; it is fetched at build time and cached under
`specs/` (see `.gitignore`). This avoids redistributing IBKR's specification.

## Pipeline

```bash
./scripts/codegen.sh
```

1. **Fetch** the spec to `specs/ibkr_spec.json` (or use the provided path).
2. **Patch** it via `scripts/patch_spec.py` → `specs/ibkr_patched.json`.
3. **Generate** with `oapi-codegen` using `oapi-codegen.yaml` → `client/client.gen.go`.

All defects are handled in step 2. No post-generation fixing is required (see
[Post-generation fixups](#post-generation-fixups)).

The build-time toolchain is pinned in `go.mod` (tools) / `Makefile` (`make tools`).

## Spec defects and patches

The published spec does **not** generate cleanly. `patch_spec.py` applies three
generalized fixes. Measured against v2.39.0:

| # | Defect | Occurrences | Fix |
|---|--------|------------:|-----|
| 1 | Path parameter declared but absent from the path template (or name differs) | 27 endpoints | Reconcile: drop spurious, rename mismatched, add missing |
| 2 | Duplicate `operationId` (`getTradingSchedule`) | 1 | Append a unique suffix to the later occurrence |
| 3 | Go type-name collision after normalization (`ErrorResponse`/`errorResponse`, `User`/`user`) | 2 | Assign `x-go-name` to the later occurrence |
| 4 | Inline query parameter schema omits `type` (OpenAPI `type: null`), e.g. `GetContractInfo`'s `sectype`, `GetConidsByExchange`'s `assetClass` | 22 params | Set `type: string` (the values are plain strings in practice) |

Reconciliation detail (defect 1): **24** spurious params dropped, **3** renamed,
**1** added. The notable case is `/gw/api/v1/balances/query`, whose POST
`$ref`s `clientIdPathParam` even though the path has no `{client-id}` segment.

The patch script is idempotent and reports counts to stderr.

## Post-generation fixups

**None required.** Earlier revisions ran `scripts/patch_gen.py` to wrap
unguarded `runtime.StyleParamWithOptions(...)` calls for query parameters typed
as a bare `interface{}`. That class of panic is now fixed at the root by
`patch_spec.py` defect 4: mapping `type: null` to `type: string` makes
`oapi-codegen` emit a concrete `string` (or a named string enum), which is
nil-safe and needs no guard.

`scripts/patch_gen.py` is retained as a no-op for backward compatibility.

## Measured result

With `oapi-codegen` **v2.8.0** and the patched spec:

| Metric | Value |
|--------|------:|
| Generator exit code | `0` |
| Output | `client/client.gen.go` |
| Lines | 72,532 |
| Size | ~2.8 MB |
| `go build` | ✅ clean |

> This number replaces an earlier, unverified claim that "184/185 endpoints
> generate cleanly". The true story is: **all 185** generate, but only after four
> classes of spec patches.

## Generated files

- `client/client.gen.go` — schema types **and** the HTTP client, package `client`.
  Marked `DO NOT EDIT`.
- Generated code is committed so consumers don't need `oapi-codegen`.
- A deterministic SPDX header is prepended by `scripts/codegen.sh` (and by
  `scripts/validate_codegen.sh` before diffing). `client/` is excluded from
  `addlicense` so the two stay consistent.

## `oapi-codegen.yaml`

```yaml
package: ibkr
output: client/client.gen.go
generate:
  models: true
  client: true
output-options:
  skip-prune: false
```

## Do not edit generated code

Rules (enforced by `AGENTS.md` and CI):

1. Never hand-edit `client/*.gen.go`. Change `scripts/patch_spec.py` (spec
   defects) and regenerate.
2. `make codegen-verify` regenerates and fails if the committed output differs.

## Drift verification

```bash
make codegen-verify
```

This fetches the spec, patches, regenerates into a temp directory, and diffs
against `client/`. A non-empty diff fails CI.

A separate [`.github/workflows/spec-drift.yml`](../.github/workflows/spec-drift.yml)
workflow runs daily and opens a GitHub issue if the upstream spec version
advances past the version pinned in `docs/SPEC.md`. This catches spec drift
before it silently breaks the next `make codegen`.

## Regenerating `docs/SPEC.md`

`docs/SPEC.md` is the canonical endpoint index. Regenerate it from the patched
spec (surface + auth columns) rather than editing by hand; see the header of that
file. Counts elsewhere (README, ROADMAP) derive from it.
