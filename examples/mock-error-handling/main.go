// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command error-handling demonstrates idiomatic error handling with the ibkrapi4go
// SDK against the mock gateway. It covers sentinel error matching via errors.Is,
// session error recovery, rate-limit backoff, graceful degradation across accounts,
// context deadline handling, and composite error collection.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/mock/error-handling.go
//
// No real IBKR account is contacted.
package main

import (
	"context"
	"errors"
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

	demonstrateErrorTypeSwitch(ctx, cli)
	demonstrateSessionErrors(ctx, cli)
	demonstrateRateLimiting(ctx, cli)
	demonstrateGracefulDegradation(ctx, cli)
	demonstrateContextDeadline(ctx, cli)
	demonstrateCompositeErrors(ctx, cli)
}

// demonstrateErrorTypeSwitch shows how to switch on ibkr.Error.Code and how to
// use errors.Is on sentinel errors.
func demonstrateErrorTypeSwitch(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Error type switch ===")

	_, err := cli.Account().Summary(ctx, "DOES_NOT_EXIST")
	if err == nil {
		fmt.Println("Account.Summary: unexpected success")
		return
	}

	var ibkrErr *ibkr.Error
	if errors.As(err, &ibkrErr) {
		fmt.Printf("ibkr.Error: op=%q code=%q status=%d msg=%q\n",
			ibkrErr.Op, ibkrErr.Code, ibkrErr.HTTPStatus, ibkrErr.Message)
	}

	if errors.Is(err, ibkr.ErrNotFound) {
		fmt.Println("errors.Is(err, ibkr.ErrNotFound): true")
	} else {
		fmt.Println("errors.Is(err, ibkr.ErrNotFound): false")
	}
}

// demonstrateSessionErrors shows ErrNotAuthenticated and ErrSessionExpired
// detection and re-authentication.
func demonstrateSessionErrors(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Session errors ===")

	status, err := cli.Session().Status(ctx)
	if err != nil {
		var ibkrErr *ibkr.Error
		if errors.As(err, &ibkrErr) {
			switch {
			case errors.Is(err, ibkr.ErrSessionExpired):
				fmt.Println("Session expired, re-initializing...")
				if reinitErr := cli.Session().Initialize(ctx); reinitErr != nil {
					fmt.Printf("Re-init failed: %v\n", reinitErr)
					return
				}
				fmt.Println("Re-init succeeded")
			case errors.Is(err, ibkr.ErrNotAuthenticated):
				fmt.Println("Not authenticated, re-initializing...")
				if reinitErr := cli.Session().Initialize(ctx); reinitErr != nil {
					fmt.Printf("Re-init failed: %v\n", reinitErr)
					return
				}
				fmt.Println("Re-init succeeded")
			default:
				fmt.Printf("Session.Status failed: %v\n", err)
			}
		}
		return
	}
	fmt.Printf("Session.Established=%v authenticated=%v connected=%v\n",
		status.Established, status.Authenticated, status.Connected)
}

// demonstrateRateLimiting shows ErrRateLimited detection with backoff retry.
func demonstrateRateLimiting(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Rate limiting with backoff ===")

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := cli.Account().List(ctx)
		if err == nil {
			if attempt > 0 {
				fmt.Printf("Account.List succeeded after %d retries\n", attempt)
			} else {
				fmt.Println("Account.List: no rate-limit error")
			}
			return
		}

		if !errors.Is(err, ibkr.ErrRateLimited) {
			fmt.Printf("Account.List: %v (not rate-limited)\n", err)
			return
		}

		fmt.Printf("ErrRateLimited detected on attempt %d, backing off...\n", attempt+1)
		backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
		if backoff > 500*time.Millisecond {
			backoff = 500 * time.Millisecond
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			fmt.Println("Context cancelled during backoff")
			return
		}
	}
	fmt.Println("Rate limit persisted after max retries")
}

// demonstrateGracefulDegradation shows how to continue processing other accounts
// when one fails.
func demonstrateGracefulDegradation(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Graceful degradation ===")

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		fmt.Printf("Account.List: %v\n", err)
		return
	}

	var failed []error
	for _, acct := range accounts {
		summary, err := cli.Account().Summary(ctx, acct.ID)
		if err != nil {
			var ibkrErr *ibkr.Error
			if errors.As(err, &ibkrErr) {
				fmt.Printf("Account %s summary failed: op=%q code=%q\n",
					acct.ID, ibkrErr.Op, ibkrErr.Code)
			} else {
				fmt.Printf("Account %s summary failed: %v\n", acct.ID, err)
			}
			failed = append(failed, fmt.Errorf("account %s: %w", acct.ID, err))
			continue
		}
		fmt.Printf("Account %s: NLV=%s buyingPower=%s\n",
			acct.ID, summary.NetLiquidationValue, summary.BuyingPower)
	}

	if len(failed) > 0 {
		fmt.Printf("Graceful degradation: %d account(s) failed, continuing\n", len(failed))
	} else {
		fmt.Println("All accounts processed successfully")
	}
}

// demonstrateContextDeadline shows context.DeadlineExceeded handling.
func demonstrateContextDeadline(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Context deadline ===")

	deadlineCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	select {
	case <-deadlineCtx.Done():
	default:
	}

	_, err := cli.Account().List(deadlineCtx)
	if err == nil {
		fmt.Println("Account.List: unexpected success with expired context")
		return
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("errors.Is(err, context.DeadlineExceeded): true")
	} else {
		fmt.Printf("Context error: %v (type=%T)\n", err, err)
	}
}

// demonstrateCompositeErrors shows collecting multiple errors with errors.Join.
// MultiClient (not shown here) returns joined errors when multiple sub-requests
// fail; this pattern demonstrates how to unpack them.
func demonstrateCompositeErrors(ctx context.Context, cli *ibkr.Client) {
	fmt.Println("\n=== Composite errors ===")

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		fmt.Printf("Account.List: %v\n", err)
		return
	}

	var multiErr error
	for _, acct := range accounts {
		_, err := cli.Account().Summary(ctx, acct.ID)
		if err != nil {
			multiErr = errors.Join(multiErr,
				fmt.Errorf("account %s: %w", acct.ID, err))
		}
	}

	if multiErr != nil {
		fmt.Printf("Collected errors (%d):\n", len(accounts)-1)
		for _, acct := range accounts {
			if errors.Is(multiErr, ibkr.ErrNotFound) {
				fmt.Printf("  account %s: not found\n", acct.ID)
			}
		}
		fmt.Printf("Joined error: %v\n", multiErr)
	} else {
		fmt.Println("No errors to join")
	}
}
