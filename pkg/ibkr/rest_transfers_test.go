// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRESTExternalAssetTransfers_Transfer(t *testing.T) {
	var transferPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/external-asset-transfers":
			transferPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":12345,"status":202}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	transferID, err := rest.Banking().ExternalTransfers().Transfer(context.Background(), AssetTransferRequest{
		AccountID:             "U1234567",
		ClientInstructionID:   1,
		ContraBrokerAccountID: "12345678A",
		ContraBrokerDtcCode:   "534",
		Direction:             "IN",
		Quantity:              100,
		ConID:                 459200101,
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if transferID != "12345" {
		t.Errorf("transferID = %q; want 12345", transferID)
	}
	if transferPath != "/gw/api/v1/external-asset-transfers" {
		t.Errorf("path = %q; want /gw/api/v1/external-asset-transfers", transferPath)
	}
}

func TestRESTInternalAssetTransfers_Transfer(t *testing.T) {
	var transferPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/internal-asset-transfers":
			transferPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":67890,"status":202}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	transferID, err := rest.Banking().InternalTransfers().Transfer(context.Background(), InternalAssetTransferRequest{
		ClientInstructionID: 1,
		SourceAccountID:     "U1234567",
		TargetAccountID:     "U7654321",
		ConID:               459200101,
		TransferQuantity:    100,
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if transferID != "67890" {
		t.Errorf("transferID = %q; want 67890", transferID)
	}
	if transferPath != "/gw/api/v1/internal-asset-transfers" {
		t.Errorf("path = %q; want /gw/api/v1/internal-asset-transfers", transferPath)
	}
}

func TestRESTExternalCashTransfers_Transfer(t *testing.T) {
	var transferPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/external-cash-transfers":
			transferPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":11111,"status":202}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	transferID, err := rest.Banking().CashTransfers().Transfer(context.Background(), CashTransferRequest{
		AccountID:             "U1234567",
		ClientInstructionID:   1,
		Amount:                1000,
		Currency:              "USD",
		BankInstructionMethod: "WIRE",
	}, true)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if transferID != "11111" {
		t.Errorf("transferID = %q; want 11111", transferID)
	}
	if transferPath != "/gw/api/v1/external-cash-transfers" {
		t.Errorf("path = %q; want /gw/api/v1/external-cash-transfers", transferPath)
	}
}

func TestRESTInternalCashTransfers_Transfer(t *testing.T) {
	var transferPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/internal-cash-transfers":
			transferPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":22222,"status":202}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	transferID, err := rest.Banking().InternalCash().Transfer(context.Background(), InternalCashTransferRequest{
		ClientInstructionID: 1,
		SourceAccountID:     "U1234567",
		TargetAccountID:     "U7654321",
		Amount:              500,
		Currency:            "USD",
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if transferID != "22222" {
		t.Errorf("transferID = %q; want 22222", transferID)
	}
	if transferPath != "/gw/api/v1/internal-cash-transfers" {
		t.Errorf("path = %q; want /gw/api/v1/internal-cash-transfers", transferPath)
	}
}

func TestRESTBankInstructions_Create(t *testing.T) {
	var createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/bank-instructions":
			createPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":33333,"status":202}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	instructionID, err := rest.Banking().BankInstructions().Create(context.Background(), BankInstructionCreateRequest{
		AccountID:           "U1234567",
		ClientInstructionID: 1,
		BankInstructionCode: "USACH",
		BankInstructionName: "TestBank",
		BankAccountNumber:   "123456789",
		BankRoutingNumber:   "021000021",
		BankAccountTypeCode: 1,
		BankName:            "Test Bank",
		Currency:            "USD",
		AchType:             "DEBIT_CREDIT",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if instructionID != "33333" {
		t.Errorf("instructionID = %q; want 33333", instructionID)
	}
	if createPath != "/gw/api/v1/bank-instructions" {
		t.Errorf("path = %q; want /gw/api/v1/bank-instructions", createPath)
	}
}

func TestRESTExternalAssetTransfers_TransferBulk(t *testing.T) {
	var transferPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/external-asset-transfers:bulk":
			transferPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"instructionSetId":44444,"status":202,"instructionResults":[{"instructionResult":{"clientInstructionId":1,"instructionId":100,"instructionStatus":"PENDING"},"instructionSetId":44444,"status":202}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	results, err := rest.Banking().ExternalTransfers().TransferBulk(context.Background(), []AssetTransferRequest{
		{
			AccountID:             "U1234567",
			ClientInstructionID:   1,
			ContraBrokerAccountID: "12345678A",
			ContraBrokerDtcCode:   "534",
			Direction:             "IN",
			Quantity:              100,
			ConID:                 459200101,
		},
	})
	if err != nil {
		t.Fatalf("TransferBulk: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("len(results) = %d; want 1", len(results))
	}
	if transferPath != "/gw/api/v1/external-asset-transfers:bulk" {
		t.Errorf("path = %q; want /gw/api/v1/external-asset-transfers:bulk", transferPath)
	}
}

func TestRESTExternalCashTransfers_QueryBalances(t *testing.T) {
	var queryPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/external-cash-transfers/query":
			queryPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cashBalance":5000.00,"currency":"USD"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	result, err := rest.Banking().CashTransfers().QueryBalances(context.Background(), CashTransferRequest{
		AccountID: "U1234567",
		Currency:  "USD",
	})
	if err != nil {
		t.Fatalf("QueryBalances: %v", err)
	}
	if result.CashBalance != "5000.00" {
		t.Errorf("CashBalance = %q; want 5000.00", result.CashBalance)
	}
	if result.Currency != "USD" {
		t.Errorf("Currency = %q; want USD", result.Currency)
	}
	if queryPath != "/gw/api/v1/external-cash-transfers/query" {
		t.Errorf("path = %q; want /gw/api/v1/external-cash-transfers/query", queryPath)
	}
}
