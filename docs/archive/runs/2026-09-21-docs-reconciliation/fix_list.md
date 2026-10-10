# Fix List — Docs Reconciliation

## Version Mismatches

| File | Line | Current Value | Should Be | Issue |
|------|------|---------------|-----------|-------|
| `pkg/ibkr/doc.go` | 19 | `Version = "v0.1.0"` | `Version = "v1.0.0"` | Version constant is stale |
| `pkg/ibkr/doc.go` | 9 | `v1 targets the CPAPI surface only` | `v1 covers both CPAPI and IB REST` | Stale scope comment; both surfaces are implemented |
| `README.md` | 9 | `Status-alpha-blue` badge | `Status-stable` or removed | Alpha badge is stale at v1.0.0 |
| `README.md` | 14 | `**⚠️ Unofficial & alpha.**` | `**⚠️ Unofficial.**` | "alpha" claim is stale at v1.0.0 |
| `README.md` | 56 | `Release \| ✅ v0.2.0` | `Release \| ✅ v1.0.0` | Release row is stale |
| `docs/ROADMAP.md` | 256 | `**v0.2.0**` | `**v1.0.0**` | Latest release reference is stale |
| `docs/RELEASING.md` | 13 | `The project is **alpha**` | Remove alpha claim | Project is v1.0.0 stable |
| `docs/RELEASING.md` | 14 | `Until \`v1.0.0\`` | `As of v1.0.0` | v1.0.0 is already released |
| `docs/ARCHITECTURE.md` | 4 | `Status: **Pre-alpha` | `Status: **Stable` | Pre-alpha is stale at v1.0.0 |
| `README.zh-Hans.md` | 58 | `发布 \| ✅ v0.2.0` | `发布 \| ✅ v1.0.0` | Release row stale |
| `README.zh-Hant.md` | 58 | `發佈 \| ✅ v0.2.0` | `發佈 \| ✅ v1.0.0` | Release row stale |
| `README.ja.md` | 59 | `リリース \| ✅ v0.2.0` | `リリース \| ✅ v1.0.0` | Release row stale |
| `README.ko.md` | 59 | `릴리스 \| ✅ v0.2.0` | `릴리스 \| ✅ v1.0.0` | Release row stale |
| `README.es.md` | 59 | `Lanzamiento \| ✅ v0.2.0` | `Lanzamiento \| ✅ v1.0.0` | Release row stale |

## Stale Feature Claims

| File | Line | Claim | Reality | Issue |
|------|------|-------|---------|-------|
| `pkg/ibkr/doc.go` | 9 | "v1 targets the CPAPI surface only" | Both CPAPI (115 ops) + IB REST (70 ops) are implemented | Scope is no longer CPAPI-only |
| `docs/ARCHITECTURE.md` | 4 | "Pre-alpha" | Project is v1.0.0 stable | Status label is stale |

## Other Issues

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `CHANGELOG.md` | 435 | `[Unreleased]` compare link still uses `v0.3.0...HEAD` | Update to `v1.0.0...HEAD` since v1.0.0 is the latest tag |

---

## Summary

**Files with issues:** 14 files

**Categories of issues:**
1. **Version constant stale** (1 file): `pkg/ibkr/doc.go` Version is v0.1.0
2. **Release row stale** (6 files): All README translations + README.md itself say v0.2.0
3. **Alpha/pre-alpha status** (3 files): README.md badge, README.md text, docs/ARCHITECTURE.md
4. **"v1 targets CPAPI only" stale** (1 file): `pkg/ibkr/doc.go` line 9
5. **RELEASING.md pre-1.0 rules** (1 file): Claims project is alpha and until v1.0.0
6. **ROADMAP.md latest release** (1 file): Says v0.2.0
7. **CHANGELOG compare link** (1 file): Unreleased still compares against v0.3.0

**Verification:** `make` is not available on this Windows environment, so `make check` and `make docs-check` could not be run. Manual code review of the files confirms the issues listed above.
