# v0.3.0 — Ergonomics & Type Safety Breaking Release (Report)

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** Complete
- **Commit:** `e8deb89`
- **Tag:** `v0.3.0` (annotated, unsigned — GPG key not available)

## Summary

Broke all 57 `Get*` prefix methods, fixed Go initialism casing (14 symbols), enforced `AccountID` type consistency (21 fields), fixed banking ID types (`float32` → `int64`), and cleaned up dead code — all in one breaking release while the user base is small.

## Files Changed

29 files modified, 439 insertions, 271 deletions.

## Breaking Changes

| Category | Count |
|----------|:-----:|
| Methods renamed (Get prefix removed) | 57 |
| Struct fields: `string` → `AccountID` | 21 |
| Struct fields: `float32` → `int64` (banking IDs) | 4 |
| Struct fields: `int64` → `ConID` | 4 |
| Initialism casing fixes | 14 |
| Abbreviation expansions | 3 |
| Deprecated aliases removed | 2 |
| Dead type removed | 1 |

## Verification

- `make check` — PASS
- `go build ./...` — PASS
- `go test -race -count=1 ./...` — PASS
- `check_money.py` — PASS

## Note

Tag is annotated but unsigned (`git tag -a` instead of `git tag -s`) because the GPG key `shing1211 <tchan@openclaw>` has no secret key available in this environment. CI `release.yml` should still create the GitHub Release from the changelog section.
