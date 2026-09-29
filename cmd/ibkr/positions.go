// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
)

func runPositions(e *env, args []string) error {

	var account string
	for i := 0; i < len(args); i++ {
		switch argAt(args, i) {
		case "-account", "--account":
			if v, ok := argValue(args, i); ok {
				account = v
				i++
			}
		case "-h", "--help":
			_, _ = fmt.Fprintf(e.stderr, "Usage: ibkr positions [-account ACCOUNT]\n\nList positions for an account.\n")
			return nil
		}
	}

	acct, err := mustAccount(account)
	if err != nil {
		return err
	}

	cli, err := e.newClient(args)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Session().Initialize(ctx()); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: session initialize: %v\n", err)
		return err
	}

	positions, err := cli.Portfolio().Positions(ctx(), acct)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: list positions: %v\n", err)
		return err
	}

	if len(positions) == 0 {
		ibkrPrintln(e, "no positions")
		return nil
	}

	// Print as table
	w := e.stdout
	_, _ = fmt.Fprintf(w, "%-12s %-8s %-30s %-6s %-8s %-12s %-12s %-12s %-12s\n",
		"ACCOUNT", "CONID", "DESCRIPTION", "CLASS", "QTY", "AVG_COST", "MKT_PRICE", "MKT_VALUE", "UNRL_PNL")
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("-", 112))
	for _, p := range positions {
		_, _ = fmt.Fprintf(w, "%-12s %-8d %-30s %-6s %-8s %-12s %-12s %-12s %-12s\n",
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
