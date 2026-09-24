# todos.md — WebSocket Shutdown Reliability

| ID | Task | Status | Verification |
|----|------|--------|--------------|
| W1 | Reproduce the `_updated` nil-map panic and `Close` hang | done | Baseline timed out at `WSConn.Close` / `ActiveSubscriptions` |
| W2 | Initialize and safely update sequence state | done | `recordSequence` uses `defer` unlock and lazy initialization |
| W3 | Make explicit close interrupt active WebSocket I/O | done | `Close` cancels the owned context and uses `CloseNow` |
| W4 | Prevent late reconnect publication | done | `dial` discards a connection if shutdown started |
| W5 | Add regression and stress coverage | done | Focused tests pass repeatedly; full suite passes |
| W6 | Correct run metadata and release documentation | done | Changelog, roadmap, and run records updated |
| W7 | Publish `v1.0.4` to GitHub and Gitee | done | Tag and both remotes synchronized |
| W8 | Make `money-check` struct-aware and green | done | Scanner passes and retains the seeded-field self-test |

Status values: `todo` · `doing` · `blocked` · `review` · `done`
