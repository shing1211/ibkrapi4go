// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"errors"
	"sync"
)

// OrderState is the SDK's normalised order lifecycle state, collapsed from
// the gateway's per-broker status strings.
type OrderState int

// Normalised order states. The zero value is OrderStateUnknown so an
// unrecognised gateway status is visible rather than silently terminal.
const (
	// OrderStateUnknown is the zero value: the gateway sent a status this
	// SDK version does not recognise.
	OrderStateUnknown OrderState = iota
	OrderStateSubmitted
	OrderStateAccepted
	OrderStatePartiallyFilled
	OrderStateFilled
	OrderStatePendingCancel
	OrderStateCancelled
	OrderStatePendingModify
	OrderStateRejected
	OrderStateExpired
	OrderStateApiCanceled
)

func (s OrderState) String() string {
	switch s {
	case OrderStateSubmitted:
		return "SUBMITTED"
	case OrderStateAccepted:
		return "ACCEPTED"
	case OrderStatePartiallyFilled:
		return "PARTIALLY_FILLED"
	case OrderStateFilled:
		return "FILLED"
	case OrderStatePendingCancel:
		return "PENDING_CANCEL"
	case OrderStateCancelled:
		return "CANCELLED"
	case OrderStatePendingModify:
		return "PENDING_MODIFY"
	case OrderStateRejected:
		return "REJECTED"
	case OrderStateExpired:
		return "EXPIRED"
	case OrderStateApiCanceled:
		return "API_CANCELED"
	default:
		return "UNKNOWN"
	}
}

// IsTerminal reports whether no further transition is possible: the order is
// filled, cancelled, or inactive.
func (s OrderState) IsTerminal() bool {
	return s == OrderStateFilled || s == OrderStateCancelled || s == OrderStateRejected || s == OrderStateExpired || s == OrderStateApiCanceled
}

// IsActive reports whether the order is still working at the exchange.
func (s OrderState) IsActive() bool {
	return s == OrderStateAccepted || s == OrderStatePartiallyFilled || s == OrderStatePendingCancel || s == OrderStatePendingModify
}

// TransitionError describes an attempted state change that the order state
// machine does not permit.
//
// The name does not follow the ErrFoo convention staticcheck prefers (ST1012).
// It is exported and has been part of the public API since it was introduced, so
// renaming it to ErrTransition would be a breaking change; the suppression is
// deliberate and this comment is the reason.
//
//nolint:staticcheck // ST1012: renaming would break the published API.
var TransitionError = errors.New("ibkr: order: invalid state transition")

type orderRecord struct {
	State         OrderState
	OrderID       string
	ClientOrderID string
	AccountID     AccountID
	ConID         ConID
}

type cOIDRegistry struct {
	mu   sync.RWMutex
	recs map[string]*orderRecord
}

func newCOIDRegistry() *cOIDRegistry {
	return &cOIDRegistry{recs: make(map[string]*orderRecord)}
}

func (r *cOIDRegistry) Get(cOID string) *orderRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.recs[cOID]
}

func (r *cOIDRegistry) Set(cOID string, rec *orderRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recs[cOID] = rec
}

func (r *cOIDRegistry) Delete(cOID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.recs, cOID)
}

func (r *cOIDRegistry) List() []*orderRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*orderRecord, 0, len(r.recs))
	for _, rec := range r.recs {
		out = append(out, rec)
	}
	return out
}

// AdvanceTo moves an order to the next state, returning an error describing
// the illegal transition if one is not allowed. It is the single entry point
// for state changes, so the transition table cannot be bypassed.
func AdvanceTo(oldState, newState OrderState) error {
	valid := map[OrderState][]OrderState{
		OrderStateSubmitted:       {OrderStateAccepted, OrderStateRejected, OrderStateExpired},
		OrderStateAccepted:        {OrderStatePartiallyFilled, OrderStateFilled, OrderStatePendingCancel, OrderStatePendingModify, OrderStateCancelled, OrderStateRejected, OrderStateExpired},
		OrderStatePartiallyFilled: {OrderStatePartiallyFilled, OrderStateFilled, OrderStatePendingCancel, OrderStatePendingModify, OrderStateCancelled, OrderStateRejected, OrderStateExpired},
		OrderStatePendingCancel:   {OrderStateCancelled, OrderStateAccepted, OrderStatePartiallyFilled},
		OrderStatePendingModify:   {OrderStateAccepted, OrderStatePartiallyFilled, OrderStateFilled, OrderStateCancelled, OrderStateRejected, OrderStateExpired},
		OrderStateFilled:          {},
		OrderStateCancelled:       {},
		OrderStateRejected:        {},
		OrderStateExpired:         {},
		OrderStateApiCanceled:     {},
	}
	allowed, ok := valid[oldState]
	if !ok {
		return TransitionError
	}
	for _, s := range allowed {
		if s == newState {
			return nil
		}
	}
	return TransitionError
}
