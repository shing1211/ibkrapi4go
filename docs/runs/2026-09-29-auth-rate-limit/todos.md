# Todos

- [x] Time each HTTP request and the gap between them, to separate a pre-call wait from a post-failure wait
- [x] Enable the client's DEBUG log against a mock gateway to name the cause
- [x] Confirm the recorder does not de-duplicate, so the request count is real
- [x] Rule out the session poll ticker, the zero-value `Clock`, and the API layer
- [x] Write the failing test for the best-effort logout (failed at 1.0005s)
- [x] Remove `/v1/api/logout` from `isAuthPath`
- [x] Guard the nil auth bucket in `needsWait` and `wait`
- [x] Add `Limiter.SetAuthRateLimit` following the `SetClock`/`SetMetrics` convention
- [x] Add `WithAuthRateLimit` with the default unchanged at 1 rps / burst 1
- [x] Wire it through all three `NewLimiter` call sites
- [x] Confirm `TestLimiter_AuthPathsAreSlow` is untouched and still passes
- [x] Check the repo's stability ADR before choosing the version number
- [x] Correct the claim that the rationale was undocumented
- [x] Update `RATE-LIMITING.md`, which listed `/logout` as an auth path
- [x] Write ADR 0018
- [x] Full gate sweep including race and coverage
- [x] Delete the throwaway probe
