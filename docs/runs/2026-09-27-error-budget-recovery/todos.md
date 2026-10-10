# Todos

- [x] Probe current behaviour with a throwaway test, changing no production code
- [x] Decide scope: bug fix now, time-based option is separate work
- [x] Write the regression test first and confirm it fails on unfixed code
- [x] Make the window a rolling window over the last `size` outcomes
- [x] Record successes as well as failures
- [x] Record a failure even when the consecutive threshold already tripped
- [x] Drop the unused `now` parameter and the `evict` method
- [x] Fix the public doc for `WithCircuitBreakerBudget` (`size<=0` also disables)
- [x] Rename `TestErrorBudget_EvictsOldEntries`, which asserted the opposite of its name
- [x] Delete `TestErrorBudget_CountsFailuresNotElapsedTime`, which pinned the defect
- [x] Widen `TestBreaker_SetErrorBudgetTripsTheBreaker` to keep its original intent
- [x] Delete the throwaway probe
- [x] Mutation-verify all three load-bearing parts of the fix
- [x] Full gate sweep, including race and coverage
- [x] Correct the severity claim in the changelog and the test comments
