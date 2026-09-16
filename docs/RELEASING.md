# Releasing

## Versioning

- [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`.
- While `MAJOR == 0`, the public API may change in a **MINOR** release; breaking
  changes are called out explicitly in the changelog.
- Once `1.0.0` is released, breaking changes require a **MAJOR** bump and, for Go
  modules, a `/v2` import path.

## Pre-1.0 policy

The project is pre-alpha. Nothing is tagged yet. Until `0.1.0`:

- Public packages may not exist yet (see [ROADMAP.md](./ROADMAP.md)).
- Tags may be deleted or rewritten before `0.1.0`.

## Release checklist

1. Ensure `make check` and `make codegen-verify` pass on `main`.
2. Confirm docs match [SPEC.md](./SPEC.md) counts and CHANGELOG is current.
3. Update `CHANGELOG.md`: move `Unreleased` → the new version, add the date.
4. Tag: `git tag -s vX.Y.Z -m "vX.Y.Z"` (signed tag).
5. Push: `git push origin vX.Y.Z`.
6. CI `release.yml` creates the GitHub Release from the changelog section.
7. Mirror to Gitee (below).

## Changelog

[Keep a Changelog](https://keepachangelog.com/) format, maintained in
[CHANGELOG.md](../CHANGELOG.md). Every user-visible change gets an entry under
`Unreleased` in the PR that makes it.

## Commit & sign-off

- [Conventional Commits](https://www.conventionalcommits.org/) for messages.
- Every commit must be DCO-signed (`git commit -s`). See
  [CONTRIBUTING.md](../CONTRIBUTING.md).

## Generated code and releases

`client/*.gen.go` is committed. When the upstream spec changes, regenerate in a
dedicated PR so the diff is reviewable and `codegen-verify` passes.

## Gitee mirror

GitHub is the source of truth. The Gitee mirror tracks tags and releases:

```bash
git push origin main --tags
git push gitee  main --tags
```

Create the corresponding Gitee release from the same changelog section. Docs stay
in English except `README.zh-CN.md` and this section for Gitee users.

## Deprecation

- Mark deprecated exported symbols with a `// Deprecated:` comment naming the
  replacement.
- Keep deprecated symbols for at least one MINOR release before removal.
- Record removals in the changelog under `Removed`.

## Module path stability

`github.com/shing1211/ibkrapi4go` is stable. Major `v2+` versions will use a
`/v2` module path and a parallel tag.

## Support matrix

| Item | Supported |
|------|-----------|
| Go | latest two minor releases (currently 1.26+) |
| OS | Linux, macOS, Windows |
| Gateway | current Client Portal Gateway (spec v2.39.0) |
