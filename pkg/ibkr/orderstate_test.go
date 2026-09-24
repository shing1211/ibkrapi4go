// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"testing"
)

func TestOrderState_String(t *testing.T) {
	tests := []struct {
		s    OrderState
		want string
	}{
		{OrderStateUnknown, "UNKNOWN"},
		{OrderStateSubmitted, "SUBMITTED"},
		{OrderStateAccepted, "ACCEPTED"},
		{OrderStatePartiallyFilled, "PARTIALLY_FILLED"},
		{OrderStateFilled, "FILLED"},
		{OrderStatePendingCancel, "PENDING_CANCEL"},
		{OrderStateCancelled, "CANCELLED"},
		{OrderStatePendingModify, "PENDING_MODIFY"},
		{OrderStateRejected, "REJECTED"},
		{OrderStateExpired, "EXPIRED"},
		{OrderStateApiCanceled, "API_CANCELED"},
		{OrderState(100), "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("OrderState(%d).String() = %q; want %q", tt.s, got, tt.want)
		}
	}
}

func TestOrderState_IsTerminal(t *testing.T) {
	terminal := []OrderState{OrderStateFilled, OrderStateCancelled, OrderStateRejected, OrderStateExpired, OrderStateApiCanceled}
	nonTerminal := []OrderState{OrderStateUnknown, OrderStateSubmitted, OrderStateAccepted, OrderStatePartiallyFilled, OrderStatePendingCancel, OrderStatePendingModify}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%v.IsTerminal() = false; want true", s)
		}
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%v.IsTerminal() = true; want false", s)
		}
	}
}

func TestOrderState_IsActive(t *testing.T) {
	active := []OrderState{OrderStateAccepted, OrderStatePartiallyFilled, OrderStatePendingCancel, OrderStatePendingModify}
	inactive := []OrderState{OrderStateUnknown, OrderStateSubmitted, OrderStateFilled, OrderStateCancelled, OrderStateRejected, OrderStateExpired, OrderStateApiCanceled}
	for _, s := range active {
		if !s.IsActive() {
			t.Errorf("%v.IsActive() = false; want true", s)
		}
	}
	for _, s := range inactive {
		if s.IsActive() {
			t.Errorf("%v.IsActive() = true; want false", s)
		}
	}
}

func TestCOIDRegistry_SetGet(t *testing.T) {
	reg := newCOIDRegistry()

	rec := &orderRecord{State: OrderStateSubmitted, ClientOrderID: "coid1", OrderID: ""}
	reg.Set("coid1", rec)

	got := reg.Get("coid1")
	if got == nil {
		t.Fatal("expected record, got nil")
	}
	if got.State != OrderStateSubmitted {
		t.Errorf("State = %v; want %v", got.State, OrderStateSubmitted)
	}
	if got.ClientOrderID != "coid1" {
		t.Errorf("ClientOrderID = %q; want coid1", got.ClientOrderID)
	}
}

func TestCOIDRegistry_GetMissing(t *testing.T) {
	reg := newCOIDRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestCOIDRegistry_Delete(t *testing.T) {
	reg := newCOIDRegistry()
	reg.Set("coid1", &orderRecord{State: OrderStateSubmitted, ClientOrderID: "coid1"})
	reg.Delete("coid1")
	if reg.Get("coid1") != nil {
		t.Error("expected nil after delete")
	}
}

func TestCOIDRegistry_List(t *testing.T) {
	reg := newCOIDRegistry()
	reg.Set("coid1", &orderRecord{State: OrderStateSubmitted, ClientOrderID: "coid1"})
	reg.Set("coid2", &orderRecord{State: OrderStateAccepted, ClientOrderID: "coid2"})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("len(List()) = %d; want 2", len(list))
	}
}

func TestAdvanceTo_LegalTransitions(t *testing.T) {
	tests := []struct {
		from, to OrderState
		ok       bool
	}{
		{OrderStateSubmitted, OrderStateAccepted, true},
		{OrderStateSubmitted, OrderStateRejected, true},
		{OrderStateSubmitted, OrderStateExpired, true},
		{OrderStateAccepted, OrderStatePartiallyFilled, true},
		{OrderStateAccepted, OrderStateFilled, true},
		{OrderStateAccepted, OrderStatePendingCancel, true},
		{OrderStateAccepted, OrderStatePendingModify, true},
		{OrderStateAccepted, OrderStateCancelled, true},
		{OrderStateAccepted, OrderStateRejected, true},
		{OrderStateAccepted, OrderStateExpired, true},
		{OrderStatePartiallyFilled, OrderStatePartiallyFilled, true},
		{OrderStatePartiallyFilled, OrderStateFilled, true},
		{OrderStatePartiallyFilled, OrderStatePendingCancel, true},
		{OrderStatePartiallyFilled, OrderStatePendingModify, true},
		{OrderStatePartiallyFilled, OrderStateCancelled, true},
		{OrderStatePartiallyFilled, OrderStateRejected, true},
		{OrderStatePartiallyFilled, OrderStateExpired, true},
		{OrderStatePendingCancel, OrderStateCancelled, true},
		{OrderStatePendingCancel, OrderStateAccepted, true},
		{OrderStatePendingCancel, OrderStatePartiallyFilled, true},
		{OrderStatePendingModify, OrderStateAccepted, true},
		{OrderStatePendingModify, OrderStatePartiallyFilled, true},
		{OrderStatePendingModify, OrderStateFilled, true},
		{OrderStatePendingModify, OrderStateCancelled, true},
		{OrderStatePendingModify, OrderStateRejected, true},
		{OrderStatePendingModify, OrderStateExpired, true},
		{OrderStateFilled, OrderStateCancelled, false},
		{OrderStateFilled, OrderStateAccepted, false},
		{OrderStateCancelled, OrderStateAccepted, false},
		{OrderStateCancelled, OrderStateFilled, false},
		{OrderStateRejected, OrderStateAccepted, false},
		{OrderStateExpired, OrderStateAccepted, false},
		{OrderStateApiCanceled, OrderStateAccepted, false},
		{OrderStateUnknown, OrderStateAccepted, false},
	}
	for _, tt := range tests {
		err := AdvanceTo(tt.from, tt.to)
		if tt.ok && err != nil {
			t.Errorf("AdvanceTo(%v, %v) = %v; want nil", tt.from, tt.to, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("AdvanceTo(%v, %v) = nil; want error", tt.from, tt.to)
		}
	}
}

func TestAdvanceTo_UnknownState(t *testing.T) {
	err := AdvanceTo(OrderStateUnknown, OrderStateAccepted)
	if err == nil {
		t.Error("AdvanceTo(OrderStateUnknown, OrderStateAccepted) = nil; want error")
	}
}
