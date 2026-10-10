// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"testing"
)

func utilitiesSurface(t *testing.T) *RESTUtilities {
	t.Helper()
	surface, err := newRESTClient(t).REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	return surface.Utilities()
}

func TestUtilities_Enumerations(t *testing.T) {
	values, err := utilitiesSurface(t).Enumerations(context.Background(), "assetClass")
	if err != nil {
		t.Fatalf("Enumerations: %v", err)
	}
	// The fixture carries jsonData: null, so there is nothing to enumerate. The
	// point is that the wrapper decodes the envelope without erroring.
	if len(values) != 0 {
		t.Errorf("values = %v; want empty for a null jsonData fixture", values)
	}
}

func TestUtilities_ComplexAssetTransferBrokers(t *testing.T) {
	brokers, err := utilitiesSurface(t).ComplexAssetTransferBrokers(context.Background())
	if err != nil {
		t.Fatalf("ComplexAssetTransferBrokers: %v", err)
	}
	if len(brokers) != 2 {
		t.Fatalf("brokers = %v; want two", brokers)
	}
	if brokers[0] != "BROKER_A" || brokers[1] != "BROKER_B" {
		t.Errorf("brokers = %v; want [BROKER_A BROKER_B]", brokers)
	}
}

func TestUtilities_Forms(t *testing.T) {
	forms, err := utilitiesSurface(t).Forms(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("Forms: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("forms = %+v; want one", forms)
	}
	if forms[0].FormNo != 1 {
		t.Errorf("formNo = %d; want 1", forms[0].FormNo)
	}
	if forms[0].Name != "Form 1" {
		t.Errorf("name = %q; want Form 1", forms[0].Name)
	}
	if forms[0].Content != "synthetic" {
		t.Errorf("content = %q; want synthetic", forms[0].Content)
	}
}

func TestUtilities_RequiredForms(t *testing.T) {
	forms, err := utilitiesSurface(t).RequiredForms(context.Background())
	if err != nil {
		t.Fatalf("RequiredForms: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("forms = %+v; want one", forms)
	}
	if forms[0].Name != "Form 1" {
		t.Errorf("name = %q; want Form 1", forms[0].Name)
	}
}

func TestUtilities_ParticipatingBanks(t *testing.T) {
	banks, err := utilitiesSurface(t).ParticipatingBanks(context.Background())
	if err != nil {
		t.Fatalf("ParticipatingBanks: %v", err)
	}
	if len(banks) != 1 {
		t.Fatalf("banks = %+v; want one", banks)
	}
	if banks[0].ID != "bank-1" {
		t.Errorf("id = %q; want bank-1", banks[0].ID)
	}
	if banks[0].Name != "Example Bank" {
		t.Errorf("name = %q; want Example Bank", banks[0].Name)
	}
}

func TestUtilities_ValidateUsername(t *testing.T) {
	available, err := utilitiesSurface(t).ValidateUsername(context.Background(), "jdoe")
	if err != nil {
		t.Fatalf("ValidateUsername: %v", err)
	}
	if !available {
		t.Error("available = false; the fixture encodes true")
	}
}
