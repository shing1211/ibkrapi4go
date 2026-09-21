// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command ibkr is a CLI tool that exposes the ibkrapi4go SDK operations as
// command-line commands. It is dependency-free, using only the Go standard
// library.
//
// Usage:
//
//	ibkr <command> [subcommand] [flags]
//
// Global flags:
//
//	-gateway    Override the Client Portal Gateway URL
//	-rest       Override the REST gateway URL
//	-account    Override the default account ID
//	-insecure   Skip TLS certificate verification
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// Version is set at build time.
var Version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ibkr: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmd := os.Args[1]
	switch cmd {
	case "--help", "-h", "help":
		if len(os.Args) > 2 {
			return printSubHelp(os.Args[2])
		}
		printUsage()
		return nil
	case "--version", "-v", "version":
		fmt.Printf("ibkr %s\n", Version)
		return nil
	case "completion":
		return runCompletion()
	case "config":
		return runConfig()
	case "accounts":
		return runAccounts()
	case "positions":
		return runPositions()
	case "orders":
		return runOrders()
	case "stream":
		return runStream()
	case "portfolio":
		return runPortfolio()
	default:
		fmt.Fprintf(os.Stderr, "ibkr: unknown command %q\n\n", cmd)
		printUsage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// newClient creates an ibkr.Client from global flags and config.
func newClient() (*ibkr.Client, error) {
	gateway, rest, account, insecure, _ := parseGlobalFlags()

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// Flags override config; config overrides defaults.
	if gateway != "" {
		cfg.GatewayURL = gateway
	}
	if rest != "" {
		cfg.RestGatewayURL = rest
	}
	if account != "" {
		cfg.AccountID = account
	}

	opts := []ibkr.Option{
		ibkr.WithGatewayURL(cfg.GatewayURL),
		ibkr.WithInsecureSkipVerify(insecure),
	}

	return ibkr.NewClient(opts...)
}

// globalFlagSet parses os.Args to extract global flags and the command index.
// Returns gateway, rest, account, insecure, and the index of the first
// non-global-flag arg.
func parseGlobalFlags() (gateway, rest, account string, insecure bool, cmdIdx int) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-gateway", "--gateway":
			if i+1 < len(args) {
				gateway = args[i+1]
				i++
			}
		case "-rest", "--rest":
			if i+1 < len(args) {
				rest = args[i+1]
				i++
			}
		case "-account", "--account":
			if i+1 < len(args) {
				account = args[i+1]
				i++
			}
		case "-insecure", "--insecure":
			insecure = true
		default:
			return gateway, rest, account, insecure, i + 1
		}
	}
	return gateway, rest, account, insecure, len(args)
}

// mustAccount returns the account ID from flag or config, or exits on error.
func mustAccount(flagValue string) (ibkr.AccountID, error) {
	if flagValue != "" {
		return ibkr.AccountID(flagValue), nil
	}
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.AccountID == "" {
		return "", fmt.Errorf("account ID required (use --account or set account_id in config)")
	}
	return ibkr.AccountID(cfg.AccountID), nil
}

// ctx returns a context with a reasonable timeout for CLI operations.
func ctx() context.Context {
	c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	_ = cancel // timeout handles cleanup
	return c
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `ibkr — Interactive Brokers CLI tool

Usage:
  ibkr <command> [subcommand] [flags]

Commands:
  accounts              List brokerage accounts
  positions             List positions for an account
  orders list           List open orders
  orders submit         Submit a new order
  orders cancel         Cancel an open order
  stream                Subscribe to market data snapshot
  portfolio summary     Portfolio summary
  portfolio ledger      Account ledger
  portfolio allocation  Asset allocation
  config show           Show current configuration
  config set            Set a configuration value
  completion            Generate shell completions

Global flags:
  -gateway  URL         Client Portal Gateway URL (default: https://localhost:5000)
  -rest     URL         REST gateway URL (default: https://api.ibkr.com)
  -account  ID          Default account ID
  -insecure             Skip TLS certificate verification

Examples:
  ibkr -gateway https://localhost:5001 accounts
  ibkr -account U1234567 positions
  ibkr orders list
  ibkr orders submit -conid 265598 -side BUY -qty 10 -type LMT -price 150.00
`)
}

func printSubHelp(sub string) error {
	switch sub {
	case "accounts":
		fmt.Fprintf(os.Stderr, "Usage: ibkr accounts\n\nList all brokerage accounts accessible in the session.\n")
	case "positions":
		fmt.Fprintf(os.Stderr, "Usage: ibkr positions [-account ACCOUNT]\n\nList positions for an account.\n")
	case "orders":
		fmt.Fprintf(os.Stderr, `Usage: ibkr orders <subcommand> [flags]

Subcommands:
  list      List open orders
  submit    Submit a new order
  cancel    Cancel an open order

Submit flags:
  -conid    INT     Contract ID (required)
  -side     SIDE    BUY or SELL (required)
  -qty      QTY     Order quantity (required)
  -type     TYPE    Order type: MKT, LMT, STP, STOP_LIMIT, MOC, LOC (default: LMT)
  -price    PRICE   Limit price
  -stop     PRICE   Stop price
  -tif      TIF     Time-in-force: DAY, GTC, IOC, OPG (default: DAY)
  -rth              Allow outside regular trading hours
  -coid     ID      Client order ID
`)
	case "stream":
		fmt.Fprintf(os.Stderr, `Usage: ibkr stream -conid CONID [-fields FIELDS]

Subscribe to a market data snapshot for a contract.

Flags:
  -conid    INT     Contract ID (required)
  -fields   LIST    Comma-separated field IDs (default: 31,84,86,87)
            Available: 31(last), 84(bid), 86(ask), 88(bid size), 85(ask size), 87(volume),
                       7295(open), 70(high), 71(low), 7296(close), 82(change), 83(change%%), 55(symbol)
`)
	case "portfolio":
		fmt.Fprintf(os.Stderr, `Usage: ibkr portfolio <subcommand> [-account ACCOUNT]

Subcommands:
  summary     Portfolio summary
  ledger      Account ledger by currency
  allocation  Asset allocation breakdown
`)
	case "config":
		fmt.Fprintf(os.Stderr, `Usage: ibkr config <subcommand>

Subcommands:
  show        Display current configuration
  set KEY VAL Set a configuration value

Keys:
  gateway     Client Portal Gateway URL
  rest        REST gateway URL
  account     Default account ID

Config file location: ~/.ibkr/config.json (or $IBKR_CONFIG)
`)
	default:
		return fmt.Errorf("no help available for %q", sub)
	}
	return nil
}
