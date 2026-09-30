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
make check           # gofmt + go vet + tests + checker gates
make codegen         # fetch/patch spec, regenerate client/
make codegen-verify  # fail if generated code differs from committed
make license         # apply SPDX headers
make license-check   # verify SPDX headers
  make docs-check      # check markdown links + README translations
  make managers-table  # regenerate the generated manager table in docs/design/03-managers.md
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
   `docs/SPEC.md`. Do not hand-edit counts elsewhere. The manager method-count
   table in `docs/design/03-managers.md` is the one other count in the tree, and
   it is generated rather than written: after adding, removing or renaming a
   manager method, run `make managers-table` and commit the result. Editing it by
   hand fails `make design-check`, by design. A manager's **scope** is prose and
   lives in `managerScopes` in `scripts/check_design/main.go` — not in the table.
7. **No dangling doc links.** If you add a reference to a doc, create it.
8. **No unreferenced packages under `internal/`.** A package there that nothing
   imports is dead code shipped in the module, and `internal/` is not importable
   from outside it, so no user can reach it. `unused` will not report it, because
   every identifier it declares is exported. `make internal-refs-check` fails the
   build; a deliberate exception goes in `ALLOWED` in
   `scripts/check_internal_refs.py` with a reason.
9. **README translations stay in lockstep.** Update the switcher in **every**   `README*.md` and the `Last synced:` banner; run `make docs-check`. English is
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

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **ibkrapi4go** (6037 symbols, 17095 relationships, 520 execution flows).

> Index stale? Run `node .gitnexus/run.cjs analyze --index-only` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? Bootstrap with `npx`, `bunx`, or `pnpm dlx` — e.g. `bunx gitnexus@latest analyze` (npm 11 npx crash; #1939).

## Always Do

- **MUST run impact before editing.** Use `impact({target: "symbolName", direction: "upstream"})` or `node .gitnexus/run.cjs impact "symbolName" --direction upstream --repo .`; report callers, processes, and risk. Never substitute grep for graph analysis.
- **MUST analyze graph changes before committing.** Use `detect_changes({scope: "all"})` (MCP) or `node .gitnexus/run.cjs detect-changes --scope all --repo .` (CLI fallback). `partial: true` or `truncated: true` is not a clean check — a zero means unseen, not unaffected; re-run it. For regression review: `detect_changes({scope: "compare", base_ref: "main"})` or `node .gitnexus/run.cjs detect-changes --scope compare --base-ref "main" --repo .`.
- MUST warn on HIGH/CRITICAL `risk` pre-edit; never use `riskSharedAxes` to waive a HIGH/CRITICAL `risk` warning. Compare File/symbol: MCP File omits axes; Graph-RAG expands File.
- **MUST treat `risk: UNKNOWN` as unresolved, not as low.** An empty caller set is not evidence the symbol is unused — it can also mean the callers are not resolvable by the index (plain-object property access, dynamic dispatch, cross-language calls). `impact` pairs `UNKNOWN` with a `riskNote` saying so. Confirm with a text search before treating the symbol as safe to change or delete; do not proceed on the strength of a zero.
- **MUST use `query({search_query: "concept"})` for concepts/flows, `context({name: "symbolName"})` for a named symbol, or `impact` for blast radius, on read-only callers, dependencies, imports, or execution flow.** Graph first; text search only for empty/`UNKNOWN`/literals.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method before MCP/CLI impact analysis.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis, and never read `UNKNOWN` as an all-clear — it means the walk could not answer, which is the one verdict that requires confirming by other means.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit before MCP/CLI graph change analysis.

## Resources

| Resource | Use for |
| --- | --- |
| `gitnexus://repo/ibkrapi4go/context` | Codebase overview, check index freshness |
| `gitnexus://repo/ibkrapi4go/clusters` | All functional areas |
| `gitnexus://repo/ibkrapi4go/processes` | All execution flows |
| `gitnexus://repo/ibkrapi4go/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
| --- | --- |
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
