// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command oauth2-flow demonstrates the full OAuth2 token lifecycle for the IB REST
// surface: acquisition, automatic and explicit refresh, invalidation, and error handling.
//
// This example requires a live IBKR account with OAuth2 credentials.
// Set the environment variables before running:
//
//	IBKR_CLIENT_ID=your_client_id \
//	IBKR_CLIENT_SECRET=your_client_secret \
//	IBKR_CLIENT_REFRESH_TOKEN=your_refresh_token \
//	  go run ./examples/live/oauth2-flow
//
// Alternatively, for client_credentials grant, omit IBKR_CLIENT_REFRESH_TOKEN.
// For JWT-bearer grant, set IBKR_JWT_KEY_FILE to a PEM-encoded RSA private key.
//
// To obtain OAuth2 credentials:
//   - Log into https://interactivebrokers.com
//   - Navigate to Settings > API > Manage API Applications
//   - Create a new application to receive client_id and client_secret
//
// WARNING: This example makes live API calls against your account. All operations
// are read-only, but you should exercise caution with credentials.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	clientID := os.Getenv("IBKR_CLIENT_ID")
	clientSecret := os.Getenv("IBKR_CLIENT_SECRET")
	refreshToken := os.Getenv("IBKR_CLIENT_REFRESH_TOKEN")
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	jwtKeyFile := os.Getenv("IBKR_JWT_KEY_FILE")

	if clientID == "" || clientSecret == "" {
		log.Fatal("IBKR_CLIENT_ID and IBKR_CLIENT_SECRET are required")
	}
	if gateway == "" {
		gateway = "https://api.ibkr.com"
	}

	opts := []ibkr.Option{
		ibkr.WithRESTGateway(gateway),
		ibkr.WithOAuth2ClientCredentials(clientID, clientSecret),
	}

	if refreshToken != "" {
		opts = append(opts, ibkr.WithOAuth2RefreshToken(refreshToken))
	}
	if jwtKeyFile != "" {
		opts = append(opts, ibkr.WithOAuth2JWTKeyFile(jwtKeyFile))
	}

	cli, err := ibkr.NewClient(opts...)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Acquire an OAuth2 token by calling REST().Token().
	// This triggers the initial token acquisition (client_credentials or refresh_token).
	fmt.Println("=== Step 1: OAuth2 Token Acquisition ===")
	rest, err := cli.REST()
	if err != nil {
		log.Fatalf("Client.REST: %v", err)
	}

	if _, err := rest.Token(ctx); err != nil {
		log.Fatalf("REST.Token (acquire): %v", err)
	}
	fmt.Println("  Access token acquired (value not displayed)")
	fmt.Printf("  Gateway URL: %s\n", rest.GatewayURL())

	// Step 2: Verify token validity by making a read call to the REST surface.
	fmt.Println("\n=== Step 2: Verify Token — Call REST Account Details ===")
	accounts, err := rest.Accounts().Details(ctx, "")
	if err != nil {
		log.Fatalf("REST.Accounts.Details: %v", err)
	}
	fmt.Printf("  Account ID: %s\n", accounts.ID)
	fmt.Printf("  Account Alias: %s\n", accounts.Alias)
	fmt.Printf("  Base Currency: %s\n", accounts.BaseCurrency)

	// Step 3: The token source auto-refreshes before expiry (30s early by
	// default). Callers can also request a refresh explicitly.
	fmt.Println("\n=== Step 3: Explicit Token Refresh ===")
	if err := rest.ForceRefresh(ctx); err != nil {
		log.Fatalf("REST.ForceRefresh: %v", err)
	}
	fmt.Println("  Access token refreshed (value not displayed)")
	rest.Invalidate()
	fmt.Println("  Cached access token invalidated; the next REST call will reacquire it")

	// Step 4: Demonstrate error handling for expired/invalid credentials.
	fmt.Println("\n=== Step 4: Error Handling — Invalid Credentials ===")
	invalidClient := os.Getenv("IBKR_CLIENT_ID") + "_invalid"
	invalidOpts := []ibkr.Option{
		ibkr.WithRESTGateway(gateway),
		ibkr.WithOAuth2ClientCredentials(invalidClient, clientSecret),
	}
	if refreshToken != "" {
		invalidOpts = append(invalidOpts, ibkr.WithOAuth2RefreshToken(refreshToken))
	}

	invalidCLI, err := ibkr.NewClient(invalidOpts...)
	if err != nil {
		log.Fatalf("NewClient (invalid): %v", err)
	}
	defer invalidCLI.Close()

	invalidREST, err := invalidCLI.REST()
	if err != nil {
		log.Fatalf("Client.REST (invalid): %v", err)
	}
	_, err = invalidREST.Token(ctx)
	if err != nil {
		fmt.Printf("  Expected error received: %v\n", err)
	} else {
		fmt.Println("  WARNING: Invalid credentials did not fail — check token caching")
	}

	// Step 5: Demonstrate error handling for network errors.
	fmt.Println("\n=== Step 5: Error Handling — Network Error ===")
	badGateway := "https://localhost:9999"
	badOpts := []ibkr.Option{
		ibkr.WithRESTGateway(badGateway),
		ibkr.WithOAuth2ClientCredentials(clientID, clientSecret),
		ibkr.WithOAuth2TokenURL("http://127.0.0.1:1/oauth2/api/v1/token"),
	}
	if refreshToken != "" {
		badOpts = append(badOpts, ibkr.WithOAuth2RefreshToken(refreshToken))
	}

	badCLI, err := ibkr.NewClient(badOpts...)
	if err != nil {
		log.Fatalf("NewClient (bad gateway): %v", err)
	}
	defer badCLI.Close()

	badCtx, badCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer badCancel()

	badREST, err := badCLI.REST()
	if err != nil {
		log.Fatalf("Client.REST (bad gateway): %v", err)
	}
	_, err = badREST.Token(badCtx)
	if err != nil {
		fmt.Printf("  Expected network error received: %v\n", err)
	} else {
		fmt.Println("  WARNING: Network error was not raised")
	}

	// Step 6: Demonstrate refresh-token rotation without exposing token material.
	fmt.Println("\n=== Step 6: Refresh Token Rotation ===")
	if err := rest.ForceRefresh(ctx); err != nil {
		log.Fatalf("REST.ForceRefresh (rotation check): %v", err)
	}
	fmt.Println("  Rotated refresh token retained internally (value not displayed)")

	fmt.Println("\n=== OAuth2 Lifecycle Complete ===")
	fmt.Println("Summary:")
	fmt.Println("  1. Token acquired via client_credentials or refresh_token grant")
	fmt.Println("  2. Token validated by calling REST.Accounts().Details()")
	fmt.Println("  3. The token source refreshes automatically before expiry")
	fmt.Println("  4. Invalid credentials produce an error on token fetch")
	fmt.Println("  5. Network errors are surfaced with proper error wrapping")
	fmt.Println("  6. Refresh token rotation is handled automatically by the token source")
}

// isAuthError returns true if err indicates an authentication failure.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "credentials") ||
		strings.Contains(errStr, "unauthorized")
}

// handleAuthError takes appropriate action based on auth error type.
func handleAuthError(ctx context.Context, err error, rest *ibkr.RESTSurface) error {
	if !isAuthError(err) {
		return err
	}
	fmt.Println("  Authentication error detected, token may need re-acquisition")
	refreshErr := rest.ForceRefresh(ctx)
	if refreshErr != nil {
		return fmt.Errorf("token re-acquisition failed: %w", refreshErr)
	}
	return nil
}

// httpStatusCheck is a helper that returns true for 4xx and 5xx responses.
func httpStatusCheck(resp *http.Response) bool {
	return resp != nil && resp.StatusCode >= 400
}
