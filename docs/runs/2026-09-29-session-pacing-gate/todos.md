# Todos

- [x] Reject the wall-clock version of this idea before implementing it
- [x] Find an exact, non-flaky signal: `ibkr.ratelimit.waits`
- [x] Reuse `NewInMemoryMetrics` rather than hand-rolling a second sink
- [x] Pair the cases: default auth pacing, and opted out
- [x] Guard against a vacuous pass with a request-count assertion
- [x] Discover that `MetricHTTPRequests` carries attributes, so a bare series key matches nothing
- [x] Sum across attribute-bearing keys and assert each of the four paths
- [x] Mutation-verify: restore `/logout` to the auth set, confirm failure, restore source
- [x] Correct the test's own comment: the opted-out case does not catch the regression
- [x] Reorder the cases so the regression-catching one runs first
- [x] Re-verify the mutation still fails after the comment and order edits
- [x] Full gate sweep including race and coverage
- [x] Add no CI step
