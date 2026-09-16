# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Project documentation set: `SPEC.md`, `ARCHITECTURE.md`, `AUTH.md`,
  `SESSIONS.md`, `ERRORS.md`, `RATE-LIMITING.md`, `STREAMING.md`, `CODEGEN.md`,
  `TESTING.md`, `RELEASING.md`, `CONFIG.md`, `GLOSSARY.md`, `ROADMAP.md`.
- Architecture Decision Records (`docs/adr/0001`–`0010`).
- Per-module design contracts (`docs/design/01`–`09`).
- `scripts/patch_spec.py`: generalized OpenAPI spec patcher (path-parameter
  reconciliation, operation-ID de-duplication, Go type-name collision fixes).
- `scripts/validate_codegen.sh`: reproducible codegen validation.
- Full Apache License 2.0 text, `NOTICE`, `THIRD_PARTY_NOTICES.md`,
  `DISCLAIMER.md`.
- Code of Conduct, Security Policy, Support and Governance documents.

### Verified

- OpenAPI codegen against IBKR Web API v2.39.0 succeeds after four classes of
  spec patches; generated client is ~72k LOC and compiles cleanly. See
  [docs/CODEGEN.md](./docs/CODEGEN.md).

### Notes

- The public SDK (`pkg/ibkr`) and `internal/` packages are **not yet
  implemented**. See [docs/ROADMAP.md](./docs/ROADMAP.md).

[Unreleased]: https://github.com/shing1211/ibkrapi4go/commits/main
