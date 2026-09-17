// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command marketdata-streaming opens a real-time market-data subscription for
// one or more conids and prints the first N field updates before closing.
// Uses the mock gateway so no real market data feed is required.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/marketdata-streaming
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

// Mock conid that the mock gateway returns data for.
const mockConid = 8314

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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	fields := []ibkr.Field{
		ibkr.FieldLastPrice,
		ibkr.FieldBidPrice,
		ibkr.FieldAskPrice,
		ibkr.FieldBidSize,
		ibkr.FieldAskSize,
		ibkr.FieldVolume,
	}

	sub, err := cli.MarketData().Subscribe(ctx, []ibkr.ConID{mockConid}, fields)
	if err != nil {
		log.Fatalf("MarketData.Subscribe: %v", err)
	}

	fmt.Println("=== Streaming market data (conid:", mockConid, ") ===")
	fmt.Println("Fields: lastPrice(31) | bidPrice(84) | askPrice(86) | bidSize(88) | askSize(85) | volume(87)")
	fmt.Print("Press Ctrl+C or wait 10s to stop.\n\n")

	deadline, _ := ctx.Deadline()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	updateCount := 0
	const maxUpdates = 40

	for updateCount < maxUpdates {
		select {
		case <-ctx.Done():
			fmt.Println("\ncontext cancelled")
			return
		case update, ok := <-sub.Updates():
			if !ok {
				fmt.Println("\nupdate channel closed")
				return
			}
			updateCount++
			ts := update.Received.Format("15:04:05.000")
			fmt.Printf("  [%s]  conid=%d  field=%s  value=%s\n",
				ts, update.ConID, update.Field, update.Value)
		case err := <-sub.Errors():
			fmt.Printf("\nerror: %v\n", err)
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				fmt.Println("\ndeadline reached")
				return
			}
		}
	}

	if err := sub.Close(); err != nil {
		log.Fatalf("Subscription.Close: %v", err)
	}
	fmt.Printf("\nreceived %d field updates\n", updateCount)
}
