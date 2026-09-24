# Next Phase: Post-WebSocket-Shutdown Directions

The WebSocket shutdown regression is fixed in `v1.0.4`.

## Recommended next phase

P2 is now complete: `scripts/check_money.py` is struct-aware and
`python scripts/check_money.py` passes while still detecting a seeded exported
`Money float64` field.

### P3 — Expose explicit OAuth2 token refresh

- Add an additive `ForceRefresh`/`Invalidate` surface on `RESTSurface`.
- Keep automatic refresh as the default behavior.
- Update the OAuth2 example and add lifecycle tests.
- Do not expose internal token values or credentials.

## Later backlog

- P4: add a unified typed streaming event entry point.
- Add a scheduled upstream spec drift check.
- Complete E3 design-checker accuracy work; it remains under review.
