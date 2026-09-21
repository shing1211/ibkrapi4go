// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func runAccounts() error {
	for _, arg := range os.Args[2:] {
		if arg == "-h" || arg == "--help" {
			fmt.Fprintf(os.Stderr, "Usage: ibkr accounts\n\nList all brokerage accounts accessible in the session.\n")
			return nil
		}
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

	accounts, err := cli.Account().List(ctx())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list accounts: %v\n", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(accounts)
}
