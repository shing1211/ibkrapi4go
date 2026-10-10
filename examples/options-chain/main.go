// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command options-chain searches for a stock by symbol, then fetches its options
// chain for a given expiry month showing available call and put strikes.
//
// It requires a running IBKR Client Portal Gateway that you have already
// authenticated in a browser (there is no username/password login — see
// docs/GATEWAY-SETUP.md and docs/AUTH.md). Point it at a non-default gateway
// with the optional IBKR_GATEWAY variable:
//
//	IBKR_GATEWAY=https://localhost:5000 go run ./examples/options-chain [SYMBOL [MONTH]]
//
// SYMBOL defaults to AAPL. MONTH defaults to the next monthly expiry after today
// (e.g. 202506 for June 2025). Month must be in YYYYMM format.
//
// All operations are READ-ONLY: no orders are submitted and no positions are modified.
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
	opts := []ibkr.Option{ibkr.WithTickleInterval(time.Hour)}
	if gateway := os.Getenv("IBKR_GATEWAY"); gateway != "" {
		opts = append(opts, ibkr.WithGatewayURL(gateway))
	}

	cli, err := ibkr.NewClient(opts...)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	symbol := "AAPL"
	if len(os.Args) > 1 {
		symbol = os.Args[1]
	}

	month := nextMonthlyExpiry()
	if len(os.Args) > 2 {
		month = os.Args[2]
	}

	fmt.Printf("=== Symbol search: %s ===\n", symbol)

	summaries, err := cli.Trade().SearchContracts(ctx, symbol)
	if err != nil {
		log.Fatalf("Trade.SearchContracts: %v", err)
	}
	if len(summaries) == 0 {
		log.Fatalf("no contracts found for %q", symbol)
	}

	var stockConID ibkr.ConID
	for _, s := range summaries {
		if s.SecType == "STK" {
			stockConID = s.ConID
			break
		}
	}
	if stockConID == 0 {
		stockConID = summaries[0].ConID
	}
	fmt.Printf("  conId=%d\n\n", stockConID)

	fmt.Printf("=== Options chain: conId=%d, month=%s ===\n", stockConID, month)

	strikes, err := cli.Trade().Strikes(ctx, stockConID, "OPT", month)
	if err != nil {
		log.Fatalf("Trade.Strikes: %v", err)
	}

	if len(strikes.Call) > 0 {
		fmt.Println("  Calls:")
		for i, s := range strikes.Call {
			if i >= 10 {
				fmt.Printf("    ... and %d more\n", len(strikes.Call)-10)
				break
			}
			fmt.Printf("    %s\n", s)
		}
	}
	if len(strikes.Put) > 0 {
		fmt.Println("  Puts:")
		for i, s := range strikes.Put {
			if i >= 10 {
				fmt.Printf("    ... and %d more\n", len(strikes.Put)-10)
				break
			}
			fmt.Printf("    %s\n", s)
		}
	}
	if len(strikes.Call) == 0 && len(strikes.Put) == 0 {
		fmt.Println("  (no strikes returned — check the month format; use YYYYMM)")
	}
}

func nextMonthlyExpiry() string {
	now := time.Now()
	year, month, _ := now.Date()
	t := time.Date(year, month+1, 15, 0, 0, 0, 0, time.UTC)
	if t.Day() < 15 {
		t = time.Date(year, month, 15, 0, 0, 0, 0, time.UTC)
	}
	return t.Format("200601")
}
