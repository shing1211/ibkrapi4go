// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func runPortfolio() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return runPortfolioSummary()
	}

	switch args[0] {
	case "summary":
		os.Args = append(os.Args[:2], args[1:]...)
		return runPortfolioSummary()
	case "ledger":
		os.Args = append(os.Args[:2], args[1:]...)
		return runPortfolioLedger()
	case "allocation":
		os.Args = append(os.Args[:2], args[1:]...)
		return runPortfolioAllocation()
	case "-h", "--help", "help":
		fmt.Fprintf(os.Stderr, `Usage: ibkr portfolio <subcommand> [-account ACCOUNT]

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

func parseAccountFlag() (string, error) {
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
			return "", nil
		}
	}
	return account, nil
}

func runPortfolioSummary() error {
	account, err := parseAccountFlag()
	if err != nil {
		return nil
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

	summary, err := cli.Portfolio().Summary(ctx(), acct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: portfolio summary: %v\n", err)
		return err
	}

	w := os.Stdout
	fmt.Fprintf(w, "Portfolio Summary — %s\n", acct)
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 60))
	for k, v := range summary {
		val := v.Amount
		if val == "" {
			val = v.Value
		}
		if v.Currency != "" {
			val += " " + v.Currency
		}
		fmt.Fprintf(w, "  %-30s %s\n", k, val)
	}
	return nil
}

func runPortfolioLedger() error {
	account, err := parseAccountFlag()
	if err != nil {
		return nil
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

	ledger, err := cli.Portfolio().Ledger(ctx(), acct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: portfolio ledger: %v\n", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(ledger)
}

func runPortfolioAllocation() error {
	account, err := parseAccountFlag()
	if err != nil {
		return nil
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

	alloc, err := cli.Portfolio().Allocation(ctx(), acct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: portfolio allocation: %v\n", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(alloc)
}
