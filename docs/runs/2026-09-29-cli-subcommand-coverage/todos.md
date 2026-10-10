# Todos

- [x] Verify the client-factory seam exists and is usable before planning around it
- [x] Verify `mockgateway.Handler()` drops into `httptest`
- [x] Verify the SDK client can be built with no credentials, as `endtoend_test.go` does
- [x] Build a harness that isolates the config and starts the mock
- [x] Drive all 9 `newClient` call sites: accounts, positions, portfolio x3, orders x3, stream
- [x] Add the error paths: gateway unreachable, 401, unusable account id
- [x] Fix `runPortfolio` panicking with no arguments
- [x] Fix `orders cancel` writing to the process stdout
- [x] Fix global flags never being read
- [x] Fix `orders submit`/`cancel` ignoring `-account`
- [x] Pin the flag behaviour with tests so it cannot silently regress
- [x] Delete the throwaway probes
- [x] Full gate sweep including race and coverage
- [x] File, rather than fix, the 2s command latency and the cmdIdx doc discrepancy
