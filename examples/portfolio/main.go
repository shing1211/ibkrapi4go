// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command portfolio queries the portfolio snapshot: accounts, positions,
// ledger, and summary. All calls target the mock gateway so no real account
// is needed.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/portfolio
//
// The gateway base URL comes from IBKR_GATEWAY_URL and defaults to the mock's
// default address, http://localhost:5001.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

const defaultGatewayURL = "http://localhost:5001"

func main() {
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	if gateway == "" {
		gateway = defaultGatewayURL
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatalf("Account.List: %v", err)
	}
	if len(accounts) == 0 {
		log.Fatal("no accounts returned")
	}
	acct := accounts[0].ID

	fmt.Println("=== Accounts ===")
	for _, a := range accounts {
		fmt.Printf("  %s  alias=%s\n", a.ID, a.Alias)
	}

	fmt.Println("\n=== Positions ===")
	positions, err := cli.Portfolio().Positions(ctx, acct)
	if err != nil {
		log.Fatalf("Portfolio.Positions: %v", err)
	}
	if len(positions) == 0 {
		fmt.Println("  (no positions)")
	}
	for _, p := range positions {
		fmt.Printf("  %s  %s  qty=%s  mktVal=%s  avgCost=%s\n",
			p.ContractDesc, p.AssetClass, p.Quantity, p.MktValue, p.AvgCost)
	}

	fmt.Println("\n=== Ledger ===")
	ledger, err := cli.Portfolio().Ledger(ctx, acct)
	if err != nil {
		log.Fatalf("Portfolio.Ledger: %v", err)
	}
	for ccy, l := range ledger {
		fmt.Printf("  %s  cash=%s  securities=%s  netLiq=%s\n",
			ccy, l.CashBalance, l.StockMarketValue, l.NetLiquidationValue)
	}

	fmt.Println("\n=== Summary ===")
	summary, err := cli.Portfolio().Summary(ctx, acct)
	if err != nil {
		log.Fatalf("Portfolio.Summary: %v", err)
	}
	showSummary := func(key string) {
		if v, ok := summary[key]; ok && !v.IsNull {
			fmt.Printf("  %s=%s %s\n", key, v.Amount, v.Currency)
		}
	}
	showSummary("netliquidation")
	showSummary("grosspositionvalue")
	showSummary("totalcash")
	showSummary("buyingpower")
}
