# P6 — Multi-Gateway / Session Pool

- **Run:** `2026-09-21-multi-gateway`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Allow a `MultiClient` to manage sessions across multiple IBKR accounts/gateways simultaneously, enabling aggregate portfolio views across families of accounts.

## Design

`MultiClient` wraps multiple `*Client` instances. Each `*Client` may point to a different gateway (e.g., different paper trading accounts). `MultiClient` provides aggregate operations that fan out to all clients in parallel.

### New file: `pkg/ibkr/multiclient.go`

```go
type MultiClient struct { clients []*Client }

func NewMultiClient(clients []*Client) *MultiClient
func (m *MultiClient) Add(client *Client)
func (m *MultiClient) Accounts(ctx) ([]Account, error)   // fan-out
func (m *MultiClient) Positions(ctx) ([]Position, error)  // aggregate across all accounts
func (m *MultiClient) Close() error

// Per-account access:
func (m *MultiClient) Clients() []*Client
```

### Tasks

| ID | Task | Size |
|----|------|------|
| M1 | `pkg/ibkr/multiclient.go` — MultiClient with aggregate Positions/Accounts | M |
| M2 | `pkg/ibkr/multiclient_test.go` — tests | M |
| M3 | Example `examples/multi-account/main.go` | S |
| M4 | Update OBSERVABILITY.md or add entry | S |
| M5 | Commit + push | S |
