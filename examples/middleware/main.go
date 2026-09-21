// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command middleware demonstrates custom HTTP transport middleware injection
// using WithTransportMiddleware.
//
// This example installs two middleware layers:
//   - A logging middleware that prints every request's method and path.
//   - A custom auth middleware that adds a static API key header.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/middleware
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

const defaultGatewayURL = "http://localhost:5001"

// loggingTransport logs each request's method and URL path.
type loggingTransport struct {
	next http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Printf("[middleware] %s %s\n", req.Method, req.URL.Path)
	return t.next.RoundTrip(req)
}

// apiKeyTransport injects a custom API key header into every request.
type apiKeyTransport struct {
	next http.RoundTripper
	key  string
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-API-Key", t.key)
	return t.next.RoundTrip(req)
}

func main() {
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	if gateway == "" {
		gateway = defaultGatewayURL
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
		// First middleware: logging (outermost user layer).
		ibkr.WithTransportMiddleware(func(next http.RoundTripper) http.RoundTripper {
			return &loggingTransport{next: next}
		}),
		// Second middleware: custom auth (wraps closer to the transport).
		ibkr.WithTransportMiddleware(func(next http.RoundTripper) http.RoundTripper {
			return &apiKeyTransport{next: next, key: "my-secret-api-key"}
		}),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatalf("Account.List: %v", err)
	}
	for _, a := range accounts {
		fmt.Printf("account: %s (%s)\n", a.ID, a.Alias)
	}
}
