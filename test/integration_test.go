// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Integration tests connect to a real IBKR Client Portal Gateway using a
// paper account. They are gated by the IBKR_GATEWAY, IBKR_USERNAME, and
// IBKR_PASSWORD environment variables and are skipped when those are absent.
//
// Run with:
//
//	IBKR_GATEWAY=https://localhost:5000 IBKR_USERNAME=... IBKR_PASSWORD=... \
//	  go test --tags=integration ./test/...
//
// These tests are READ-ONLY: they never submit orders, modify positions, or
// initiate transfers. They validate that the SDK works against a real gateway
// and are not run in normal CI.
//
// Deliberately excluded, because they mutate state on the account:
//
//   - ModelManager.SubmitModelPortfolioOrder (submits orders)
//   - ModelManager.TwsInvestDivest and ModelManager.RebalanceTo* (move money
//     and rebalance model portfolios)
//
// Their response decoding is covered by the mock-gateway tests in pkg/ibkr. What
// only a real gateway can confirm is that IBKR dispatches each operationId to
// the right handler, which is server behaviour the mock cannot represent.
// Running these for real requires an FA-enabled paper account and a separate
// opt-in build tag, which this suite deliberately does not provide.
package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"go.uber.org/goleak"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func TestMain(m *testing.M) {
	if os.Getenv("IBKR_GATEWAY") == "" {
		fmt.Println("SKIP: IBKR_GATEWAY not set; integration tests require a running")
		fmt.Println("      Client Portal Gateway and paper account credentials")
		os.Exit(0)
	}

	code := m.Run()
	time.Sleep(50 * time.Millisecond)
	if code == 0 {
		if err := goleak.Find(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func testClient(t *testing.T) *ibkr.Client {
	t.Helper()

	gateway := os.Getenv("IBKR_GATEWAY")
	username := os.Getenv("IBKR_USERNAME")
	password := os.Getenv("IBKR_PASSWORD")

	if gateway == "" || username == "" || password == "" {
		t.Fatalf("IBKR_GATEWAY, IBKR_USERNAME, and IBKR_PASSWORD are required")
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return cli
}

func TestIntegration_SessionAndAccounts(t *testing.T) {
	cli := testClient(t)
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		t.Fatalf("Account.List: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("expected at least one account")
	}
	t.Logf("accounts: %+v", accounts)
}

func TestIntegration_PortfolioPositions(t *testing.T) {
	cli := testClient(t)
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		t.Fatalf("Account.List: %v", err)
	}

	positions, err := cli.Portfolio().Positions(ctx, accounts[0].ID)
	if err != nil {
		t.Fatalf("Portfolio.Positions: %v", err)
	}
	t.Logf("positions count: %d", len(positions))
	for _, p := range positions {
		t.Logf("  conId=%d  %s  qty=%s", p.ConID, p.ContractDesc, p.Quantity)
	}
}

func TestIntegration_MarketDataSnapshot(t *testing.T) {
	cli := testClient(t)
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}

	snapshot, err := cli.MarketData().Snapshot(ctx, []ibkr.ConID{8314}, []ibkr.Field{
		ibkr.FieldLastPrice,
		ibkr.FieldBidPrice,
		ibkr.FieldAskPrice,
	})
	if err != nil {
		t.Fatalf("MarketData.Snapshot: %v", err)
	}
	for _, s := range snapshot {
		t.Logf("conid=%d  last=%s  bid=%s  ask=%s", s.ConID,
			s.Fields[ibkr.FieldLastPrice],
			s.Fields[ibkr.FieldBidPrice],
			s.Fields[ibkr.FieldAskPrice])
	}
}

// TestIntegration_AllocationModels is read-only: it only lists the configured
// allocation models. It is an FA surface, so it skips when the account has no
// financial-advisor entitlement rather than failing.
func TestIntegration_AllocationModels(t *testing.T) {
	cli := testClient(t)
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}

	models, err := cli.Allocation().AllocationModels(ctx)
	if err != nil {
		var apiErr *ibkr.Error
		if errors.As(err, &apiErr) && (apiErr.HTTPStatus == http.StatusForbidden ||
			apiErr.HTTPStatus == http.StatusUnauthorized) {
			t.Skipf("skipping: account lacks FA entitlement for the allocation surface: %v", err)
		}
		t.Fatalf("Allocation.AllocationModels: %v", err)
	}
	if len(models) == 0 {
		t.Skip("skipping: no allocation models configured on this account")
	}
	for name, instruments := range models {
		t.Logf("model %s: %s", name, instruments)
	}
}
