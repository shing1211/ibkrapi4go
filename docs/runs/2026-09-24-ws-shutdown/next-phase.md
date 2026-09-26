# Next Phase: Post-WebSocket-Shutdown Directions

The WebSocket shutdown regression is fixed in `v1.0.4`.

## Completed since this run was written

- **P2** — `scripts/check_money.py` is struct-aware, and
  `python scripts/check_money.py` passes while still detecting a seeded exported
  `Money float64` field.
- **P3** — Done. `RESTSurface.ForceRefresh(ctx)` and `Invalidate()` are now
  available. Automatic refresh remains the default, rotated refresh tokens are
  preserved, invalidated in-flight results cannot repopulate the cache, and the
  live example no longer prints token material.
- **E3** — the design-checker accuracy work is complete. `check_design` now
  extracts the middleware order from the `ms = append(ms, ...)` calls in
  `internal/transport.go` and diffs it against the documented chain, so code and
  document drift fails the check. It had previously compared an always-empty
  order and could not detect anything.
- Scheduled upstream spec drift check — shipped as
  `.github/workflows/spec-drift.yml` (weekly, Monday 09:00 UTC).

## Recommended next phase

### P4 — Unified typed streaming events

**Status:** Not started.

- Add one entry point for account, portfolio, order, notification, and user
  events.
- Preserve the existing per-subscription channels and reconnect behavior.
- Document ordering and duplicate-delivery expectations.
