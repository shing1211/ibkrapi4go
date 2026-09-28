# Todos: audit every fixed wait in the test suite

- [x] Establish the classification rule: waiting for a guaranteed event is a
      defect; waiting for time to pass is not
- [x] Read all 21 `time.Sleep` sites and classify each
- [x] Find the 5 further sites that are `time.After` in a `select` with no event
      arm — invisible to a grep for `time.Sleep`
- [x] Convert the 6 session tests to drive the tickle loop directly
- [x] Add a manual `Clock` whose tickers each get their own channel, so a
      double-start is detectable
- [x] Convert the 5 WebSocket tests to wait on a delivery signal
- [x] Add notify channels to `fakeSink`, `fakeSystemSink` and
      `fakeOutOfOrderSink`
- [x] Make the wait helpers fail loudly on a nil channel instead of hanging
- [x] Prove `TestSession_StartTickle_Idempotent` was vacuous, then make it real
- [x] Find and fix the production defect: `Initialize` slept 1s before its first
      poll, taxing every session start
- [x] Remove the now-unused `Clock.Sleep` and its `sleepFunc` hook
- [x] Confirm the 15 remaining sleeps are legitimate, and record why
- [x] Re-measure coverage after a false 55.0% reading
- [x] Full verification and this run's artifacts

## Deliberately not done

- [ ] The 15 legitimate sleeps were left as they are. Changing them would make
      the suite slower without making it more truthful.
