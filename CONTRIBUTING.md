# Contributing to ibkrapi4go

Thank you for your interest in contributing!

- [Code of Conduct](./CODE_OF_CONDUCT.md)
- [Security Policy](./SECURITY.md)
- [Governance](./GOVERNANCE.md)
- [Support](./SUPPORT.md)

---

## Getting Started

### Prerequisites

- Go 1.26+
- [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) v2 (for code generation)
- A running IBKR Client Portal Gateway (for integration tests)

### Setup

```bash
git clone https://github.com/shing1211/ibkrapi4go
cd ibkrapi4go

# Install tools used by the Makefile
make tools

# Regenerate types from the OpenAPI spec (optional; generated code is committed)
make codegen

# Format, vet, test
make check
```

---

## Repository Layout

```
.
├── client/          # Generated code — DO NOT EDIT MANUALLY
├── pkg/ibkr/        # Public SDK — edit here for API changes
├── internal/        # Internal implementation — breaking changes OK
├── cmd/             # Standalone binaries (ibkr-mock-gateway)
├── examples/        # Runnable examples (mock)
├── docs/            # Reference, design, ADRs
├── scripts/         # Codegen + validation scripts
└── specs/           # Cached OpenAPI spec (gitignored)
```

---

## How to Contribute

### 1. Fork and Branch

```bash
git checkout -b feat/your-feature-name
# or
git checkout -b fix/your-bug-fix-name
```

Branch naming: `feat/`, `fix/`, `docs/`, `test/`, `chore/` prefixes.

### 2. Make Changes

- **New API endpoint**: update the spec handling if needed, regenerate types,
  then wrap in the appropriate manager. Never hand-edit `client/*.gen.go`.
- **Bug fix**: add a failing test first, then fix.
- **Design change**: open an ADR under `docs/adr/` first (see GOVERNANCE.md).
- **Documentation**: update the relevant `.md` files and godoc comments.

### 3. Run Checks

```bash
make check        # gofmt + go vet + tests (recommended)

# Or manually:
gofmt -s -w .
go vet ./...
go test ./...
```

### 4. Commit

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(account): add List with pagination support
fix(trade): handle nil pointer on order response
docs(readme): add WebSocket example
test(portfolio): add integration test for position snapshot
```

### 5. Sign your commits (DCO)

All commits **must be signed off** under the
[Developer Certificate of Origin 1.1](https://developercertificate.org/):

```bash
git commit -s -m "feat(account): add List with pagination support"
```

This adds a `Signed-off-by: Your Name <you@example.com>` trailer. By signing off
you certify that you have the right to submit the contribution under the
project's license ([Apache-2.0](./LICENSE)). Commits without a sign-off will not
be merged.

### 6. Pull Request

- Fill out the pull-request template.
- Reference the related issue (if any).
- For API changes: include the operation ID from [docs/SPEC.md](./docs/SPEC.md).
- Ensure CI is green, including `make codegen-verify` if you touched spec handling.

---

## Codegen Workflow

When the IBKR OpenAPI spec changes:

```bash
# 1. Fetch + patch + generate (writes specs/ and client/)
make codegen

# 2. Review the diff
git diff client/

# 3. Confirm reproducibility
make codegen-verify

# 4. Update the canonical endpoint index
#    (regenerate docs/SPEC.md — see docs/CODEGEN.md)
```

Known spec defects and how `scripts/patch_spec.py` addresses them are documented
in [docs/CODEGEN.md](./docs/CODEGEN.md).

---

## Code Style

- **Go formatting**: `gofmt` (enforced by CI).
- **Error handling**: always handle `error`; never ignore it with `_`.
- **Context**: all public functions accept `context.Context` as the first arg.
- **No globals**: all state lives in `Client`, injected via options.
- **Money**: never `float64`; use `string`/`json.Number`
  ([ADR 0008](./docs/adr/0008-numeric-precision.md)).
- **Minimal dependencies**: prefer the standard library; new deps need an ADR.

---

## Architecture Tour

The SDK has three layers:

```
pkg/ibkr/         Public SDK — edit here for API changes
     ↓
internal/         Internal implementation — breaking changes OK
     ↓
client/           Generated OpenAPI code — DO NOT EDIT MANUALLY
```

**Public surface** (`pkg/ibkr/`) is the only surface external packages may depend
on. Managers (`AccountManager`, `TradeManager`, etc.) are accessed via
`Client` accessors. All public functions accept `context.Context` as the first
argument. Never import `internal/` from outside the module.

See [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) for the full design.

## Testing Guidelines

| Test Type | Location | Runs In CI | Needs Account |
|-----------|----------|------------|---------------|
| Unit tests | `*_test.go` alongside source | ✅ | No |
| Integration tests | `test/` (`//go:build integration`) | On demand | Paper account |
| Codegen validation | `scripts/` | ✅ | No |

See [docs/TESTING.md](./docs/TESTING.md).

## Code Review Criteria

Before merging, consider:

- **No float64 for money or quantities.** Use `string` or `json.Number`
  ([ADR 0008](./docs/adr/0008-numeric-precision.md)).
- **No auto-retry on order mutations.** Write operations (`Submit`, `Modify`,
  `Cancel`) must not be retried automatically
  ([ADR 0009](./docs/adr/0009-no-auto-retry-orders.md)).
- **Context on all public functions.** Every exported function must accept
  `context.Context` as its first parameter.
- **Minimal dependencies.** New runtime dependencies require an ADR
  ([ADR 0004](./docs/adr/0004-minimal-dependencies.md)).
- **Never hand-edit generated code.** Changes to `client/*.gen.go` go through
  `scripts/patch_spec.py` + `make codegen`; see
  [docs/CODEGEN.md](./docs/CODEGEN.md).
- **Backwards compatibility.** Pre-1.0, breaking changes are permitted in minor
  releases but must be called out in the changelog; after 1.0 they require a
  major bump.

---

## Reporting Issues

Bug reports welcome. Please include:

- Go version (`go version`)
- ibkrapi4go version (git commit or tag)
- IBKR OpenAPI spec version (check [docs/SPEC.md](./docs/SPEC.md) header)
- Gateway version and paper/live designation
- Minimal reproduction case
- Full error output, with secrets redacted

---

## Translations

Translations of the [README](./README.md) are welcome and follow
[TRANSLATING.md](./TRANSLATING.md):

- English is canonical; translations are best-effort.
- Add/update the language switcher in **every** `README*.md`.
- Keep the translation banner and its `Last synced:` commit current.
- Do **not** translate legal text (`LICENSE`, `DISCLAIMER.md`).
- Run `make docs-check`; CI enforces `scripts/check_i18n.py`.

Current languages: English, 简体中文, 繁體中文, 日本語, 한국어, Español.

---

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](./LICENSE), and you certify the DCO sign-off above.
