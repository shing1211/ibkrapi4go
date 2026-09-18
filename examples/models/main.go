// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command models fetches the list of model portfolios and their positions
// for the first account, using the mock gateway so no real account is needed.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/models
//
// The gateway base URL comes from IBKR_GATEWAY_URL and defaults to the mock's
// default address, http://localhost:5001.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

const defaultGatewayURL = "http://localhost:5001"

func main() {
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	if gateway == "" {
		gateway = defaultGatewayURL
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	reqID := time.Now().UnixMilli()

	models, err := cli.Model().AllModels(ctx, reqID)
	if err != nil {
		log.Fatalf("Model.GetAllModels: %v", err)
	}
	if len(models) == 0 {
		fmt.Println("(no model portfolios)")
		return
	}

	fmt.Println("=== Model portfolios ===")
	for _, name := range models {
		fmt.Printf("  %s\n", name)
	}

	first := models[0]
	fmt.Println("\n=== Positions for:", first, "===")
	positions, err := cli.Model().AllModelPositions(ctx, first)
	if err != nil {
		log.Fatalf("Model.GetAllModelPositions: %v", err)
	}
	if len(positions) == 0 {
		fmt.Println("  (no positions)")
		return
	}
	for _, p := range positions {
		fmt.Printf("  conId=%d  symbol=%s  position=%s  avgCost=%s\n",
			p.ConID, p.Symbol, p.Position, p.AvgCost)
	}

	fmt.Println("\n=== Model presets ===")
	presets, err := cli.Model().ModelPresets(ctx, reqID)
	if err != nil {
		log.Fatalf("Model.GetModelPresets: %v", err)
	}
	if len(presets) == 0 {
		fmt.Println("  (no presets)")
		return
	}
	for _, p := range presets {
		fmt.Printf("  name=%s  accounts=%v\n", p.Name, p.Accounts)
	}
}
