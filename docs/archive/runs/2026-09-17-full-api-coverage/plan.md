# Plan: Full API Coverage (185/185 Operations)

## Goal
Implement all remaining CPAPI (ssoBearer) operations to achieve 100% coverage of the IBKR OpenAPI spec (185 operations total).

## Scope
- 70 IB REST (oauth2Bearer): ✅ Already complete
- ~81 CPAPI (ssoBearer): To implement

## Approach
Group by API section, create manager files following existing patterns.

## Task Breakdown

| Phase | Section | Ops | Files |
|-------|---------|-----|-------|
| A | Trading Accounts | 8 | `pkg/ibkr/accounts.go` |
| B | Trading Alerts | 6 | `pkg/ibkr/alerts.go` |
| C | Trading Contracts | 13 | `pkg/ibkr/contracts.go` |
| D | Trading Event Contracts + Scanner | 7 | `pkg/ibkr/events.go`, `pkg/ibkr/scanner.go` |
| E | Trading FA Allocation + Model Portfolios | 18 | `pkg/ibkr/allocation.go`, `pkg/ibkr/models.go` |
| F | Trading FYIs/Notifications + OAuth | 14 | `pkg/ibkr/notifications.go`, `pkg/ibkr/oauth1.go` |
| G | Trading Portfolio + Analyst + Watchlists | 11 | `pkg/ibkr/portfolio.go` (extend), `pkg/ibkr/watchlists.go` |
| H | Trading Session + verification | 1 | `pkg/ibkr/session.go` (extend) |

## Risks
- Some CPAPI operations may have complex request/response types
- OAuth 1.0a operations require different auth flow
- Need to verify each operation compiles and tests pass

## Verification
- `make check` (fmt, vet, money-check, tests)
- `make test-race`
- `make docs-check`
- `make license-check`
