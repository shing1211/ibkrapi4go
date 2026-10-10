# v0.3.0 — Ergonomics & Type Safety Breaking Release

| ID | Task | Role | Status | Depends |
|----|------|------|--------|---------|
| A1a | Get prefix removal — Scanner/TradingAccount/Model/Session/Watchlist/Allocation/FYI | backend | done | — |
| A1b | Get prefix removal — TradeManager (13 methods) | backend | done | — |
| A1c | Get prefix removal — Performance/Alert/Forecast/Portfolio/RESTRequests | backend | done | — |
| B1 | Go initialism casing (14 symbols) + ReqAccessToken | backend | done | — |
| C1 | AccountID type consistency (21 fields) | backend | done | — |
| D1 | float32 → int64 for banking IDs (4 fields) | backend | done | — |
| E1 | ConID type consistency (4 fields) | backend | done | — |
| F1 | Remove dead RESTInstructions type | backend | done | — |
| G1 | Remove deprecated ErrStream* aliases | backend | done | — |
| H1 | Fix method/type name collisions | backend | done | A1b |
| I1 | Update all tests for renamed symbols | tester | done | A1a–H1 |
| I2 | Verify: make check + race tests | tester | done | I1 |
| I3 | CHANGELOG + commit + tag v0.3.0 + push | release | done | I2 |
