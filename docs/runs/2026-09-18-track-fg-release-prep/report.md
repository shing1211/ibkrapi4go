# Tracks F + G — Release Prep & Tag v0.2.0 (Report)

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** Complete
- **Release commit:** `5cb0141`
- **Tag:** `v0.2.0` (annotated, unsigned — GPG key not available)

## Summary

Prepared and released v0.2.0 with all changes from Tracks A–E. Created STABILITY.md, wrote the CHANGELOG entry, updated version references across all README translations, and tagged + pushed to both remotes.

## Files Changed

| File | Change |
|------|--------|
| `CHANGELOG.md` | Added v0.2.0 entry (Added/Changed/Deprecated/Fixed/Internal) |
| `docs/STABILITY.md` | New — user-facing stability contract from ADR 0015 |
| `docs/RELEASING.md` | Added STABILITY.md/ADR 015 refs, license-check in checklist |
| `docs/ROADMAP.md` | Updated latest release to v0.2.0 |
| `README.md` | Bumped version ref v0.1.1 → v0.2.0 |
| `README.zh-Hans.md` | Bumped version ref v0.1.1 → v0.2.0 |
| `README.zh-Hant.md` | Bumped version ref v0.1.1 → v0.2.0 |
| `README.ja.md` | Bumped version ref v0.1.1 → v0.2.0 |
| `README.ko.md` | Bumped version ref v0.1.1 → v0.2.0 |
| `README.es.md` | Bumped version ref v0.1.1 → v0.2.0 |

## Verification

- `make check` — PASS
- `git tag -l | grep v0.2.0` — PASS
- Both remotes (`origin` + `gitee`) updated with tag

## Note

Tag is annotated but unsigned (`git tag -a` instead of `git tag -s`) because the GPG key `shing1211 <tchan@openclaw>` has no secret key available in this environment. The CI `release.yml` should still create the GitHub Release from the changelog section.
