// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// fieldAlias maps friendly names to IBKR field IDs.
var fieldAlias = map[string]ibkr.Field{
	"last":     ibkr.FieldLastPrice,
	"bid":      ibkr.FieldBidPrice,
	"ask":      ibkr.FieldAskPrice,
	"bid_size": ibkr.FieldBidSize,
	"ask_size": ibkr.FieldAskSize,
	"volume":   ibkr.FieldVolume,
	"open":     ibkr.FieldOpen,
	"high":     ibkr.FieldHigh,
	"low":      ibkr.FieldLow,
	"close":    ibkr.FieldClose,
	"change":   ibkr.FieldChange,
	"pct":      ibkr.FieldChangePercent,
	"symbol":   ibkr.FieldSymbol,
}

func runStream() error {
	args := os.Args[2:]

	var conid int
	var fieldsStr string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-conid", "--conid":
			if i+1 < len(args) {
				conid, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-fields", "--fields":
			if i+1 < len(args) {
				fieldsStr = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, `Usage: ibkr stream -conid CONID [-fields FIELDS]

Subscribe to a market data snapshot for a contract.

Flags:
  -conid    INT     Contract ID (required)
  -fields   LIST    Comma-separated field names or IDs (default: last,bid,ask,volume)
            Aliases: last, bid, ask, bid_size, ask_size, volume, open, high, low, close, change, pct, symbol
            Or numeric IDs: 31, 84, 86, 85, 88, 87, 7295, 70, 71, 7296, 82, 83, 55
`)
			return nil
		}
	}

	if conid == 0 {
		return fmt.Errorf("--conid is required")
	}

	fields := parseFields(fieldsStr)

	cli, err := newClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx()); err != nil {
		fmt.Fprintf(os.Stderr, "error: session initialize: %v\n", err)
		return err
	}

	snapshots, err := cli.MarketData().Snapshot(ctx(), []ibkr.ConID{ibkr.ConID(conid)}, fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: market data snapshot: %v\n", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(snapshots)
}

// parseFields converts a comma-separated field string into []ibkr.Field,
// resolving aliases.
func parseFields(s string) []ibkr.Field {
	if s == "" {
		return []ibkr.Field{
			ibkr.FieldLastPrice,
			ibkr.FieldBidPrice,
			ibkr.FieldAskPrice,
			ibkr.FieldVolume,
		}
	}
	parts := strings.Split(s, ",")
	fields := make([]ibkr.Field, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if f, ok := fieldAlias[p]; ok {
			fields = append(fields, f)
		} else {
			fields = append(fields, ibkr.Field(p))
		}
	}
	return fields
}
