// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command otel demonstrates using the contrib/otel bridge to send SDK metrics
// to OpenTelemetry.
//
// Start the mock gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./contrib/otel/examples/otel
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
	otelbridge "github.com/shing1211/ibkrapi4go/contrib/otel"
)

const defaultGatewayURL = "http://localhost:5001"

func main() {
	// Set up OTel meter with stdout exporter for demonstration.
	exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		log.Fatalf("stdoutmetric.New: %v", err)
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(5*time.Second))),
	)
	defer provider.Shutdown(context.Background())

	meter := otel.Meter("ibkrapi4go")
	metrics := otelbridge.New(meter)

	// Create SDK client with OTel metrics wired in.
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	if gateway == "" {
		gateway = defaultGatewayURL
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithMetrics(metrics),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	// Make some requests — metrics are automatically recorded.
	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatalf("Account.List: %v", err)
	}

	fmt.Printf("Found %d account(s)\n", len(accounts))
	fmt.Println("Metrics are being exported to stdout via OpenTelemetry.")
	fmt.Println("Check the exporter output for ibkr.* metrics.")
}
