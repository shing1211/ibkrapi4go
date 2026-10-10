# Tasks

## 1. Execution Integrity Foundation

- [ ] 1.1 Add idempotency key generation utility in `internal/idempotency.go` and verify unit test confirms uniqueness across 10,000 generations
- [ ] 1.2 Implement sequence tracking in `ClientTransport` and verify gap detection triggers an immediate reconnect and logs the incident
- [ ] 1.3 Add microsecond-precision audit logging to all mutation endpoints and verify log timestamp precision is within 1ms of system clock

## 2. Transport Resilience Foundation

- [ ] 2.1 Implement stateful circuit breaker with CLOSED/OPEN/HALF-OPEN states in `internal/transport.go` and verify state transitions via unit test simulating error spikes
- [ ] 2.2 Implement jittered exponential backoff retry logic and verify retry count does not exceed 5 and total backoff time does not exceed 30s via test
- [ ] 2.3 Implement adaptive rate limiter with 50% reduction on 429 response and verify rate recovery to 100% after 5 successful requests via test

## 3. Client Gateway Safety

- [ ] 3.1 Implement global `SafetyState` struct in `pkg/ibkr/client.go` with `EngageKillSwitch()` and `DisengageKillSwitch()` and verify all subsequent trading operations are rejected with `ErrSafetyEngaged`
- [ ] 3.2 Add HMAC-SHA256 signing to mutation requests exceeding a configurable size threshold (default: $100,000) and verify signature is attached to request headers via test
- [ ] 3.3 Add deterministic failure handler for critical integrity errors (unknown fills, price bound violations) and verify Kill Switch is engaged immediately and error is logged

## 4. Integration & Verification

- [ ] 4.1 Write integration test for full request middleware chain and verify Rate Limiter -> Circuit Breaker -> Signer -> Sender flow works correctly under load
- [ ] 4.2 Write end-to-end test for Kill Switch engagement and verify all open orders are cancelled and connection is severed within 100ms
- [ ] 4.3 Update `README.md` with new production readiness features (Kill Switch, Idempotency, Audit Logging) and verify `make docs-check` passes

## Workflow follow-up

- Archive the change after review requirements are satisfied: `openspec archive production-sdk-readiness`
- Verify the archived result: `openspec validate --change production-sdk-readiness`
