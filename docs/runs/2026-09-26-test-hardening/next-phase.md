# Test Hardening - Next Phase

## Open Items

| Item | Severity | Notes |
|------|----------|-------|
| `TestWS_Resilience` flake cause unconfirmed | Medium | Leading hypothesis is a goroutine outliving `WSConn.Close`, caught by a per-test `goleak.VerifyNone(t)` that has no settle window. Reproduction is the next task; do not add a sleep before reading the leaked stack. |
| Per-test goleak attribution | Medium | Package-exit detection exists; naming the leaking test does not. |
| Coverage floor at 35% versus 43.3% measured | Low | 8.3 points of headroom. Deliberate ratchet, not a rescue. |
| Mutating model endpoints unverified against a real gateway | Low | Needs an FA-enabled paper account and an opt-in build tag. The mock cannot route the operation separately. |
| `cmd/ibkr` and `cmd/ibkr-mock-gateway` largely uncovered | Low | 318 statements in the denominator. Kept there by decision; only order validation and flag handling are worth testing. |

## Next Actions

1. Reproduce the `TestWS_Resilience` flake and read the leaked goroutine stack.
   Fix the cause in `Close` or the loop shutdown, not with a longer settle.
2. Add per-test `goleak.VerifyNone(t)` to `internal/ws_test.go`,
   `internal/session_test.go`, `pkg/ibkr/ws_test.go`, and
   `pkg/ibkr/managers_e2e_test.go`. `internal/session_test.go` has no cleanup
   sites at all and is the likeliest to surface a real leak.
3. Slice 1: cover `restrictions.go`, `rest_utilities.go`, `notifications.go`, and
   `trading_accounts.go` - 548 statements, about 8.7 points if fully covered.
   Re-measure and set the floor to the measured value, never above it.

## Carried Forward Unchanged

From `audit-remediation`:

- Extend `check_design` to the seven design documents it does not read
  (02, 04, 05, 06, 07, 08, 09). Its existing presence and order checks are sound.
- Unified typed streaming events, under ADR 0015.
- The mock gateway's inability to route `SubmitModelPortfolioOrder` separately.
  The request bodies are byte-identical, so no body predicate can discriminate;
  this is a property of the payloads, not a gap to close.
