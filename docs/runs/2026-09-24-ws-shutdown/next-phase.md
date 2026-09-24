# Next Phase: Post-WebSocket-Shutdown Directions

The WebSocket shutdown regression is fixed in `v1.0.4`.

## Recommended next phase

P2 is now complete: `scripts/check_money.py` is struct-aware and
`python scripts/check_money.py` passes while still detecting a seeded exported
`Money float64` field.

### P3 — Expose explicit OAuth2 token refresh

**Status:** Done. `RESTSurface.ForceRefresh(ctx)` and `Invalidate()` are now
available. Automatic refresh remains the default, rotated refresh tokens are
preserved, invalidated in-flight results cannot repopulate the cache, and the
live example no longer prints token material.

## Recommended next phase

### P4 — Unified typed streaming events

- Add one entry point for account, portfolio, order, notification, and user
  events.
- Preserve the existing per-subscription channels and reconnect behavior.
- Document ordering and duplicate-delivery expectations.

## Later backlog

- Add a scheduled upstream spec drift check.
- Complete E3 design-checker accuracy work; it remains under review.
