// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"errors"
	"sync"
)

type OrderState int

const (
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

func (s OrderState) IsTerminal() bool {
	return s == OrderStateFilled || s == OrderStateCancelled || s == OrderStateRejected || s == OrderStateExpired || s == OrderStateApiCanceled
}

func (s OrderState) IsActive() bool {
	return s == OrderStateAccepted || s == OrderStatePartiallyFilled || s == OrderStatePendingCancel || s == OrderStatePendingModify
}

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
