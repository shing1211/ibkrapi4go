// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command screener runs a market scanner against the live gateway. It first fetches
// available scanner parameters (instruments, locations, scan types) then runs a
// scan using the first available values.
//
// This example requires a running IBKR Client Portal Gateway and paper trading
// credentials. Set the environment variables before running:
//
//	IBKR_GATEWAY=https://localhost:5000 \
//	IBKR_USERNAME=yourpaperusername \
//	IBKR_PASSWORD=yourpaperpassword \
//	  go run ./examples/screener
//
// All operations are READ-ONLY: no orders are submitted and no positions are modified.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
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

	params, err := cli.Scanner().ScannerParameters(ctx)
	if err != nil {
		log.Fatalf("Scanner.ScannerParameters: %v", err)
	}

	var paramData struct {
		InstrumentList []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"instrument_list"`
		LocationTree []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"location_tree"`
		ScanTypeList []struct {
			Code        string   `json:"code"`
			DisplayName string   `json:"displayName"`
			Instruments []string `json:"instruments"`
		} `json:"scan_type_list"`
	}
	if err := json.Unmarshal(params, &paramData); err != nil {
		log.Fatalf("decode scanner parameters: %v", err)
	}

	var defaultInstrument, defaultLocation, defaultScanType string
	if len(paramData.InstrumentList) > 0 {
		defaultInstrument = paramData.InstrumentList[0].Code
	}
	if len(paramData.LocationTree) > 0 {
		defaultLocation = paramData.LocationTree[0].Code
	}
	if len(paramData.ScanTypeList) > 0 {
		defaultScanType = paramData.ScanTypeList[0].Code
	}

	fmt.Println("=== Scanner parameters ===")
	fmt.Printf("  Instruments (%d): %s", len(paramData.InstrumentList), defaultInstrument)
	for i := 1; i < len(paramData.InstrumentList) && i < 5; i++ {
		fmt.Printf(", %s", paramData.InstrumentList[i].Code)
	}
	if len(paramData.InstrumentList) > 5 {
		fmt.Printf(" ... (+%d more)", len(paramData.InstrumentList)-5)
	}
	fmt.Println()

	fmt.Printf("  Locations (%d): %s", len(paramData.LocationTree), defaultLocation)
	for i := 1; i < len(paramData.LocationTree) && i < 5; i++ {
		fmt.Printf(", %s", paramData.LocationTree[i].Code)
	}
	if len(paramData.LocationTree) > 5 {
		fmt.Printf(" ... (+%d more)", len(paramData.LocationTree)-5)
	}
	fmt.Println()

	fmt.Printf("  Scan types (%d): %s (%s)", len(paramData.ScanTypeList), defaultScanType,
		paramData.ScanTypeList[0].DisplayName)
	for i := 1; i < len(paramData.ScanTypeList) && i < 3; i++ {
		fmt.Printf(", %s", paramData.ScanTypeList[i].Code)
	}
	if len(paramData.ScanTypeList) > 3 {
		fmt.Printf(" ... (+%d more)", len(paramData.ScanTypeList)-3)
	}
	fmt.Println()

	fmt.Printf("\n=== Running scan: instrument=%s location=%s type=%s ===\n",
		defaultInstrument, defaultLocation, defaultScanType)

	scanReq := map[string]interface{}{
		"instrument": defaultInstrument,
		"location":  defaultLocation,
		"type":      defaultScanType,
	}

	results, err := cli.Scanner().ScannerResults(ctx, scanReq)
	if err != nil {
		log.Fatalf("Scanner.ScannerResults: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("  (no results)")
		return
	}

	fmt.Printf("  %d results\n\n", len(results))
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	fmt.Println("  Top results:")
	for i, r := range results {
		if i >= 10 {
			fmt.Printf("  ... and %d more\n", len(results)-10)
			break
		}
		fmt.Printf("  [%2d] %-6s %-8s %-20s  exchange=%-10s  distance=%s\n",
			i+1, r.SecType, r.Symbol, r.CompanyName, r.Exchange, r.Distance)
	}
}
