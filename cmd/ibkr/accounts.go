// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
)

func runAccounts(e *env, args []string) error {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			_, _ = fmt.Fprintf(e.stderr, "Usage: ibkr accounts\n\nList all brokerage accounts accessible in the session.\n")
			return nil
		}
	}

	cli, err := e.newClient(args)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	cliCtx, cancel := ctx()
	defer cancel()

	if err := cli.Session().Initialize(cliCtx); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: session initialize: %v\n", err)
		return err
	}

	accounts, err := cli.Account().List(cliCtx)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: list accounts: %v\n", err)
		return err
	}

	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(accounts)
}
