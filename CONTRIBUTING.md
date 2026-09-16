# Contributing to ibkr-sdk

Thank you for your interest in contributing!

- [Code of Conduct](./CODE_OF_CONDUCT.md)
- [Security Policy](./SECURITY.md)

---

## Getting Started

### Prerequisites

- Go 1.26+
- [oapi-codegen](https://github.com/deepmap/oapi-codegen) for code generation
- A running IBKR Client Portal Gateway (for integration tests)

### Setup

```bash
git clone https://github.com/shing1211/ibkrapi4go
cd ibkr-sdk
go mod download

# Generate types from latest spec
./scripts/codegen.sh

# Run unit tests
go test ./...

# Run integration tests (requires Client Portal Gateway)
IBKR_GATEWAY=https://localhost:5000 \
IBKR_USERNAME=your_username \
IBKR_PASSWORD=your_password \
go test ./test/... -tags=integration -v
```

---

## Repository Layout

```
.
├── client/          # Generated code — DO NOT EDIT MANUALLY
│   ├── client.gen.go
│   └── types.gen.go
├── pkg/ibkr/        # Public SDK — edit here for API changes
├── internal/        # Internal implementation — breaking changes OK
├── docs/            # Architecture docs
├── scripts/         # Codegen + patching scripts
├── test/            # Integration tests
└── examples/        # Usage examples
```

---

## How to Contribute

### 1. Fork and Branch

```bash
git checkout -b feat/your-feature-name
# or
git checkout -b fix/your-bug-fix-name
```

Branch naming: `feat/`, `fix/`, `docs/`, `test/` prefixes.

### 2. Make Changes

- **New API endpoint**: Add to spec first, then regenerate types, then wrap in the appropriate `*Manager`
- **Bug fix**: Add a failing test first, then fix
- **Documentation**: Update relevant `.md` files and godoc comments

### 3. Run Checks

```bash
# Using Makefile (recommended)
make check

# Or manually:
go fmt ./...
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

### 5. Pull Request

- Fill out the PR template (appears automatically)
- Reference the GitHub issue (if any)
- For API changes: include the operation ID from [docs/SPEC.md](./docs/SPEC.md)
- For new endpoints: add entry to [docs/SPEC.md](./docs/SPEC.md)

---

## Codegen Workflow

When the IBKR OpenAPI spec is updated:

```bash
# 1. Fetch latest spec
curl -s https://api.ibkr.com/gw/api/v3/api-docs -o specs/ibkr_v2_40.json

# 2. Patch spec bugs
python3 scripts/patch_spec.py specs/ibkr_v2_40.json > specs/ibkr_patched.json

# 3. Regenerate
./scripts/codegen.sh specs/ibkr_patched.json

# 4. Review generated diff
git diff client/client.gen.go client/types.gen.go

# 5. Update SPEC.md endpoint index
# (re-run the analysis script from PLAN.md)
```

---

## Code Style

- **Go formatting**: `gofmt` (auto-enforced by CI)
- **Error handling**: Always handle `error`, never `_`
- **Context**: All public functions accept `context.Context` as first arg
- **No globals**: All state in `Client` struct, injected via options
- **Minimal dependencies**: Prefer stdlib, add deps only when necessary

---

## Testing Guidelines

| Test Type | Location | Runs In CI | Needs Account |
|-----------|----------|------------|---------------|
| Unit tests | `*_test.go` alongside source | ✅ | No |
| Integration tests | `test/` | ✅ (with env vars) | Paper account |
| Spec compliance | `scripts/` | On spec update | No |

---

## Reporting Issues

Bug reports welcome! Please include:
- Go version (`go version`)
- ibkr-sdk version (git commit or tag)
- IBKR API spec version (check [docs/SPEC.md](./docs/SPEC.md) header)
- Minimal reproduction case
- Full error output

---

## License

By contributing, you agree that your contributions will be licensed under the
Apache License 2.0, same as the project. See [LICENSE](./LICENSE).
