# Todos

- [x] Audit each carried-forward blocked item against the code, not the notes.
- [x] Establish whether `submitModelPortfolioOrder` is unimplemented or merely
      unrouted in the mock gateway. It is the latter.
- [x] Measure the SPEC-opId versus route-table gap empirically rather than by hand.
- [x] Correct the misleading 1.1.14 changelog entry, inline.
- [x] Add `TestEveryOpIDIsRoutedOrExplained`: every distinct SPEC operation is
      routed, or carries a verified exception.
- [x] Verify each exception rather than trusting it - stale excuse, unrouted
      shadow, and false collision reason all fail.
- [x] Add `sharedOpIDs` for the reverse case, with stale-entry detection.
- [x] Mutation-verify all six branches, including deleting the entry itself.
- [x] Add a 1.1.15 changelog entry.
- [x] Full verification sweep.
- [ ] Commit, tag `v1.1.15`, push to GitHub and Gitee - **held for explicit
      approval**.

## Deliberately not done

- [ ] D15 live verification - needs a real account.
- [ ] Confirm the `submitModelPortfolioOrder` collision against a real gateway -
      needs an FA-enabled paper account.
- [ ] `Subscription.Close(ctx)` for WebSocket `contextcheck` - breaking API change,
      maintainer decision.
- [ ] Cover the remaining `if err != nil` branches in `trading_accounts.go` and
      `notifications.go` - previously judged to assert the language rather than
      behaviour, and left that way.
