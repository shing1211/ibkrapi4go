// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"
)

func runPositions() error {
	args := os.Args[2:]

	var account string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-account", "--account":
			if i+1 < len(args) {
				account = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, "Usage: ibkr positions [-account ACCOUNT]\n\nList positions for an account.\n")
			return nil
		}
	}

	acct, err := mustAccount(account)
	if err != nil {
		return err
	}

	cli, err := newClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx()); err != nil {
		fmt.Fprintf(os.Stderr, "error: session initialize: %v\n", err)
		return err
	}

	positions, err := cli.Portfolio().Positions(ctx(), acct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list positions: %v\n", err)
		return err
	}

	if len(positions) == 0 {
		fmt.Println("no positions")
		return nil
	}

	// Print as table
	w := os.Stdout
	fmt.Fprintf(w, "%-12s %-8s %-30s %-6s %-8s %-12s %-12s %-12s %-12s\n",
		"ACCOUNT", "CONID", "DESCRIPTION", "CLASS", "QTY", "AVG_COST", "MKT_PRICE", "MKT_VALUE", "UNRL_PNL")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 112))
	for _, p := range positions {
		fmt.Fprintf(w, "%-12s %-8d %-30s %-6s %-8s %-12s %-12s %-12s %-12s\n",
			p.AccountID, p.ConID, truncate(p.ContractDesc, 30), p.AssetClass,
			p.Quantity, p.AvgCost, p.MktPrice, p.MktValue, p.UnrealizedPnL)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
