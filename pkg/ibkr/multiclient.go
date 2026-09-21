// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"sync"
)

// MultiClient manages multiple Client instances, enabling aggregate views across
// multiple IBKR accounts or gateways. It is useful for family offices, prop
// trading firms, or any setup where positions need to be aggregated across
// multiple accounts or gateways.
//
// MultiClient is safe for concurrent use. All aggregate operations fan out
// to all clients in parallel. Errors from individual clients are collected
// but do not abort the aggregate operation unless all clients fail.
type MultiClient struct {
	mu      sync.RWMutex
	clients []*Client
}

// NewMultiClient returns a MultiClient managing the given clients. The clients
// must already be initialized (Session.Initialize called) before being added.
// At least one client is required.
func NewMultiClient(clients []*Client) (*MultiClient, error) {
	if len(clients) == 0 {
		return nil, &ConfigError{Field: "clients", Message: "at least one client is required"}
	}
	return &MultiClient{clients: clients}, nil
}

// Add registers an additional client. It is safe to call while other
// aggregate operations are in progress.
func (m *MultiClient) Add(client *Client) {
	if client == nil {
		return
	}
	m.mu.Lock()
	m.clients = append(m.clients, client)
	m.mu.Unlock()
}

// Clients returns a snapshot of the registered clients.
func (m *MultiClient) Clients() []*Client {
	m.mu.RLock()
	out := make([]*Client, len(m.clients))
	copy(out, m.clients)
	m.mu.RUnlock()
	return out
}

// Accounts aggregates account listings across all clients.
func (m *MultiClient) Accounts(ctx context.Context) ([]Account, error) {
	clients := m.Clients()
	type result struct {
		accounts []Account
		err      error
	}
	results := make([]result, len(clients))

	var wg sync.WaitGroup
	for i, c := range clients {
		wg.Add(1)
		go func(idx int, cli *Client) {
			defer wg.Done()
			accounts, err := cli.Account().List(ctx)
			results[idx] = result{accounts: accounts, err: err}
		}(i, c)
	}
	wg.Wait()

	var out []Account
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		out = append(out, r.accounts...)
	}
	if len(errs) == len(results) && len(errs) > 0 {
		return nil, errs[0]
	}
	return out, nil
}

// Positions aggregates positions across all accounts of all clients.
// Each position's AccountID field is already set by the underlying
// Portfolio.Positions call.
func (m *MultiClient) Positions(ctx context.Context) ([]Position, error) {
	clients := m.Clients()

	type acctResult struct {
		accountID string
		positions []Position
		err       error
	}

	acctResults := make([]acctResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, c := range clients {
		accounts, err := c.Account().List(ctx)
		if err != nil {
			mu.Lock()
			acctResults = append(acctResults, acctResult{err: err})
			mu.Unlock()
			continue
		}
		for _, a := range accounts {
			wg.Add(1)
			go func(acct Account, cli *Client) {
				defer wg.Done()
				positions, err := cli.Portfolio().Positions(ctx, acct.ID)
				mu.Lock()
				acctResults = append(acctResults, acctResult{
					accountID: string(acct.ID),
					positions: positions,
					err:       err,
				})
				mu.Unlock()
			}(a, c)
		}
	}
	wg.Wait()

	var out []Position
	var errs []error
	for _, r := range acctResults {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		out = append(out, r.positions...)
	}
	if len(errs) == len(acctResults) && len(errs) > 0 {
		return nil, errs[0]
	}
	return out, nil
}

// Close closes all registered clients. It is safe to call even if some clients
// are already closed; nil clients are skipped.
func (m *MultiClient) Close() error {
	clients := m.Clients()
	var errs []error
	for _, c := range clients {
		if c != nil {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
