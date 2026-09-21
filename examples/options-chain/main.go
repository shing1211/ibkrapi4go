// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command options-chain searches for a stock by symbol, then fetches its options
// chain for a given expiry month showing available call and put strikes.
//
// This example requires a running IBKR Client Portal Gateway and paper trading
// credentials. Set the environment variables before running:
//
//	IBKR_GATEWAY=https://localhost:5000 \
//	IBKR_USERNAME=yourpaperusername \
//	IBKR_PASSWORD=yourpaperpassword \
//	  go run ./examples/options-chain [SYMBOL [MONTH]]
//
// SYMBOL defaults to AAPL. MONTH defaults to the next monthly expiry after today
// (e.g. 202506 for June 2025). Month must be in YYYYMM format.
//
// All operations are READ-ONLY: no orders are submitted and no positions are modified.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	gateway := os.Getenv("IBKR_GATEWAY")
	username := os.Getenv("IBKR_USERNAME")
	password := os.Getenv("IBKR_PASSWORD")

	if gateway == "" || username == "" || password == "" {
		log.Fatal("IBKR_GATEWAY, IBKR_USERNAME, and IBKR_PASSWORD are required")
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
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

	httpCli := cli.HTTPClient()
	baseURL := strings.TrimSuffix(gateway, "/")

	fmt.Printf("=== Symbol search: %s ===\n", symbol)

	searchURL := fmt.Sprintf("%s/v1/api/iserver/secdef/search?symbol=%s&secType=STK",
		baseURL, url.QueryEscape(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		log.Fatalf("NewRequest: %v", err)
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		log.Fatalf("/iserver/secdef/search: %v", err)
	}
	defer resp.Body.Close()

	var rawSearch []map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawSearch); err != nil {
		log.Fatalf("decode search response: %v", err)
	}
	if len(rawSearch) == 0 {
		log.Fatalf("no contracts found for %q", symbol)
	}

	var stockConID string
	for _, r := range rawSearch {
		if raw, ok := r["secType"]; ok {
			var st string
			if err := json.Unmarshal(raw, &st); err == nil && st == "STK" {
				if raw, ok := r["conid"]; ok {
					stockConID = strings.Trim(string(raw), `"`)
					break
				}
			}
		}
	}
	if stockConID == "" {
		if raw, ok := rawSearch[0]["conid"]; ok {
			stockConID = strings.Trim(string(raw), `"`)
		}
	}
	fmt.Printf("  conId=%s\n\n", stockConID)

	fmt.Printf("=== Options chain: conId=%s, month=%s ===\n", stockConID, month)

	strikesURL := fmt.Sprintf("%s/v1/api/iserver/secdef/strikes?conid=%s&sectype=OPT&month=%s",
		baseURL, url.QueryEscape(stockConID), url.QueryEscape(month))
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, strikesURL, nil)
	if err != nil {
		log.Fatalf("NewRequest: %v", err)
	}
	resp2, err := httpCli.Do(req2)
	if err != nil {
		log.Fatalf("/iserver/secdef/strikes: %v", err)
	}
	defer resp2.Body.Close()

	var strikeData struct {
		Call []float32 `json:"call,omitempty"`
		Put  []float32 `json:"put,omitempty"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&strikeData); err != nil {
		log.Fatalf("decode strikes response: %v", err)
	}

	if len(strikeData.Call) > 0 {
		fmt.Println("  Calls:")
		for i, s := range strikeData.Call {
			if i >= 10 {
				fmt.Printf("    ... and %d more\n", len(strikeData.Call)-10)
				break
			}
			fmt.Printf("    %.2f\n", s)
		}
	}
	if len(strikeData.Put) > 0 {
		fmt.Println("  Puts:")
		for i, s := range strikeData.Put {
			if i >= 10 {
				fmt.Printf("    ... and %d more\n", len(strikeData.Put)-10)
				break
			}
			fmt.Printf("    %.2f\n", s)
		}
	}
	if len(strikeData.Call) == 0 && len(strikeData.Put) == 0 {
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
