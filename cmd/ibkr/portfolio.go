// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func runPortfolio(e *env, args []string) error {
	if len(args) == 0 {
		// args[1:] on an empty slice is a bounds panic, and runOrders' equivalent
		// branch passes nothing. The default subcommand is summary.
		return runPortfolioSummary(e, nil)
	}

	switch args[0] {
	case "summary":
		return runPortfolioSummary(e, args[1:])
	case "ledger":
		return runPortfolioLedger(e, args[1:])
	case "allocation":
		return runPortfolioAllocation(e, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintf(e.stderr, `Usage: ibkr portfolio <subcommand> [-account ACCOUNT]

Subcommands:
  summary     Portfolio summary
  ledger      Account ledger by currency
  allocation  Asset allocation breakdown
`)
		return nil
	default:
		return fmt.Errorf("unknown portfolio subcommand %q", args[0])
	}
}

func parseAccountFlag(args []string) (string, error) {
	var account string
	for i := 0; i < len(args); i++ {
		switch argAt(args, i) {
		case "-account", "--account":
			if v, ok := argValue(args, i); ok {
				account = v
				i++
			}
		case "-h", "--help":
			return "", nil
		}
	}
	return account, nil
}

func runPortfolioSummary(e *env, args []string) error {
	account, err := parseAccountFlag(args)
	if err != nil {
		return nil
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

	summary, err := cli.Portfolio().Summary(ctx(), acct)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: portfolio summary: %v\n", err)
		return err
	}

	w := e.stdout
	_, _ = fmt.Fprintf(w, "Portfolio Summary — %s\n", acct)
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("-", 60))
	for k, v := range summary {
		val := v.Amount
		if val == "" {
			val = v.Value
		}
		if v.Currency != "" {
			val += " " + v.Currency
		}
		_, _ = fmt.Fprintf(w, "  %-30s %s\n", k, val)
	}
	return nil
}

func runPortfolioLedger(e *env, args []string) error {
	account, err := parseAccountFlag(args)
	if err != nil {
		return nil
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

	ledger, err := cli.Portfolio().Ledger(ctx(), acct)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: portfolio ledger: %v\n", err)
		return err
	}

	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(ledger)
}

func runPortfolioAllocation(e *env, args []string) error {
	account, err := parseAccountFlag(args)
	if err != nil {
		return nil
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

	alloc, err := cli.Portfolio().Allocation(ctx(), acct)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: portfolio allocation: %v\n", err)
		return err
	}

	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(alloc)
}
