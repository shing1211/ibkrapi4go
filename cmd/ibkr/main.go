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
	"io"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// Version is set at build time.
var Version = "dev"

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ibkr: %v\n", err)
		os.Exit(1)
	}
}

// run is the testable entry point. It takes its arguments and output streams as
// parameters rather than reading os.Args and os.Stdout directly, and hands every
// subcommand an explicit env, so no command below this line touches the process.
func run(args []string, stdout, stderr io.Writer) error {
	e := newEnv(args, stdout, stderr)
	if len(args) < 2 {
		printUsage(stderr)
		return nil
	}
	return dispatch(e, args[1])
}

// dispatch routes a command name to its implementation. It is separate from run
// so a test can reach the dispatch table with an env it built, without going
// through argv parsing.
func dispatch(e *env, cmd string) error {
	stdout, stderr := e.stdout, e.stderr
	switch cmd {
	case "--help", "-h", "help":
		if len(e.args) > 2 {
			return printSubHelp(stderr, e.args[2])
		}
		printUsage(stderr)
		return nil
	case "--version", "-v", "version":
		_, _ = fmt.Fprintf(stdout, "ibkr %s\n", Version)
		return nil
	case "completion":
		return runCompletion(e.arg(2), stdout, stderr)
	case "config":
		return runConfig(e, e.arg(2))
	case "accounts":
		return runAccounts(e, e.arg(2))
	case "positions":
		return runPositions(e, e.arg(2))
	case "orders":
		return runOrders(e, e.arg(2))
	case "stream":
		return runStream(e, e.arg(2))
	case "portfolio":
		return runPortfolio(e, e.arg(2))
	default:
		_, _ = fmt.Fprintf(stderr, "ibkr: unknown command %q\n\n", cmd)
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// newClientFromArgs creates an ibkr.Client from the global flags in args and the
// config file. It takes the subcommand's arguments, not the full argv, so that the
// global flags after the command name are the ones that are read. It takes argv as
// a parameter rather than reading os.Args, so a test can build a client with chosen
// flags.
func newClientFromArgs(args []string) (*ibkr.Client, error) {
	gateway, rest, account, insecure, _ := parseGlobalFlags(args)

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

	// rest is deliberately not forwarded. cfg.RestGatewayURL is parsed, stored,
	// printed by `ibkr config` and settable by `ibkr config set`, but pkg/ibkr
	// exposes no option to receive it - the only URL option is WithGatewayURL,
	// which covers both API surfaces. So the flag is accepted and inert, and the
	// honest fix is either an SDK option or its removal, not a second one here.
	opts := []ibkr.Option{
		ibkr.WithGatewayURL(cfg.GatewayURL),
		ibkr.WithInsecureSkipVerify(insecure),
	}

	return ibkr.NewClient(opts...)
}

// parseGlobalFlags extracts the global flags from args and returns the index of
// the first non-flag argument. It takes the arguments as a parameter, like run,
// so the parsing is reachable from a test without touching os.Args.
func parseGlobalFlags(args []string) (gateway, rest, account string, insecure bool, cmdIdx int) {
	for i := 0; i < len(args); i++ {
		switch argAt(args, i) {
		case "-gateway", "--gateway":
			if v, ok := argValue(args, i); ok {
				gateway = v
				i++
			}
		case "-rest", "--rest":
			if v, ok := argValue(args, i); ok {
				rest = v
				i++
			}
		case "-account", "--account":
			if v, ok := argValue(args, i); ok {
				account = v
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

// argAt returns args[i], or "" when i is out of range.
//
// Every command used to spell its bounds out inline as `if i+1 < len(args)` and
// then index `args[i]` directly in the switch - sixteen times across five files.
// gosec does not follow a bound held in a loop condition or a switch case, so it
// reported the guarded access as a possible out-of-range read. A `//nolint`
// silenced one lint profile and then tripped nolintlint in the other, where gosec
// reported nothing at all. Keeping the check immediately above the index, inside
// the helper, is the form gosec does verify - and it puts the bound in one place.
func argAt(args []string, i int) string {
	if i < 0 || i >= len(args) {
		return ""
	}
	return args[i]
}

// argValue returns the argument following the flag at index i, and reports
// whether one was actually there. The caller advances its own index so the value
// is not read again as a flag.
func argValue(args []string, i int) (string, bool) {
	if i < 0 || i+1 >= len(args) {
		return "", false
	}
	return args[i+1], true
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
func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, `ibkr — Interactive Brokers CLI tool

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

func printSubHelp(w io.Writer, sub string) error {
	switch sub {
	case "accounts":
		_, _ = fmt.Fprintf(w, "Usage: ibkr accounts\n\nList all brokerage accounts accessible in the session.\n")
	case "positions":
		_, _ = fmt.Fprintf(w, "Usage: ibkr positions [-account ACCOUNT]\n\nList positions for an account.\n")
	case "orders":
		_, _ = fmt.Fprintf(w, `Usage: ibkr orders <subcommand> [flags]

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
		_, _ = fmt.Fprintf(w, `Usage: ibkr stream -conid CONID [-fields FIELDS]

Subscribe to a market data snapshot for a contract.

Flags:
  -conid    INT     Contract ID (required)
  -fields   LIST    Comma-separated field IDs (default: 31,84,86,87)
            Available: 31(last), 84(bid), 86(ask), 88(bid size), 85(ask size), 87(volume),
                       7295(open), 70(high), 71(low), 7296(close), 82(change), 83(change%%), 55(symbol)
`)
	case "portfolio":
		_, _ = fmt.Fprintf(w, `Usage: ibkr portfolio <subcommand> [-account ACCOUNT]

Subcommands:
  summary     Portfolio summary
  ledger      Account ledger by currency
  allocation  Asset allocation breakdown
`)
	case "config":
		_, _ = fmt.Fprintf(w, `Usage: ibkr config <subcommand>

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
