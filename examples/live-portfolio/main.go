// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command live-portfolio queries the live portfolio snapshot using paper trading
// credentials: accounts, positions, ledger, and summary.
//
// This example requires a running IBKR Client Portal Gateway and paper trading
// credentials. Set the environment variables before running:
//
//	IBKR_GATEWAY=https://localhost:5000 \
//	IBKR_USERNAME=yourpaperusername \
//	IBKR_PASSWORD=yourpaperpassword \
//	  go run ./examples/live-portfolio
//
// All operations are READ-ONLY: no orders are submitted, no positions are modified,
// and no transfers are initiated.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	gateway := os.Getenv("IBKR_GATEWAY")
	username := os.Getenv("IBKR_USERNAME")
	password := os.Getenv("IBKR_PASSWORD")

	if gateway == "" || username == "" || password == "" {
		log.Fatal("IBKR_GATEWAY, IBKR_USERNAME, and IBKR_PASSWORD are required")
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	fmt.Printf("=== Accounts (%d total) ===\n", len(accounts))
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
