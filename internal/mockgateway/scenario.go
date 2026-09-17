// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"sync"
	"sync/atomic"
	"time"
)

// Fault describes an injected deviation from the default fixture response.
// A zero-value Fault is a no-op; callers set only the fields they need.
type Fault struct {
	// Status forces an HTTP status code. Zero keeps the fixture's status.
	Status int
	// Body overrides the response body. Empty keeps the fixture's body.
	Body string
	// Delay adds latency before the response. When global latency is also set,
	// the effective delay is the maximum of this and the global latency.
	Delay time.Duration
	// DropConnection closes the connection without writing a response,
	// simulating an ambiguous transport failure.
	DropConnection bool
	// Timeout blocks until the client's context is cancelled (or a safety
	// cap elapses), simulating a transport timeout.
	Timeout bool
}

// FaultPolicy resolves an injected Fault for a request. Returning nil defers
// to the per-op and global faults (and finally the fixture).
type FaultPolicy func(op string, req *Request) *Fault

// Scenario holds scriptable fault-injection state for a Server. All methods are
// safe for concurrent use.
type Scenario struct {
	mu        sync.RWMutex
	global    *Fault
	ops       map[string]*Fault
	latency   time.Duration
	onRequest FaultPolicy

	sessionExpired atomic.Bool
	orderReply     atomic.Bool
	orderReplyBody atomic.Value // string
}

// NewScenario returns an empty Scenario.
func NewScenario() *Scenario {
	return &Scenario{ops: make(map[string]*Fault)}
}

// SetGlobal installs a fault applied to every operation that has no per-op
// override.
func (s *Scenario) SetGlobal(f *Fault) {
	s.mu.Lock()
	s.global = f
	s.mu.Unlock()
}

// ClearGlobal removes the global fault.
func (s *Scenario) ClearGlobal() {
	s.mu.Lock()
	s.global = nil
	s.mu.Unlock()
}

// Set installs a fault for a single operation ID.
func (s *Scenario) Set(op string, f *Fault) {
	s.mu.Lock()
	if s.ops == nil {
		s.ops = make(map[string]*Fault)
	}
	s.ops[op] = f
	s.mu.Unlock()
}

// Clear removes the per-op fault for op.
func (s *Scenario) Clear(op string) {
	s.mu.Lock()
	delete(s.ops, op)
	s.mu.Unlock()
}

// SetLatency sets a global delay applied to every response.
func (s *Scenario) SetLatency(d time.Duration) {
	s.mu.Lock()
	s.latency = d
	s.mu.Unlock()
}

// Latency returns the global response delay.
func (s *Scenario) Latency() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latency
}

// SetPolicy installs a FaultPolicy. The policy is consulted before per-op and
// global faults.
func (s *Scenario) SetPolicy(p FaultPolicy) {
	s.mu.Lock()
	s.onRequest = p
	s.mu.Unlock()
}

// SetSessionExpired toggles 401 responses on protected routes.
func (s *Scenario) SetSessionExpired(v bool) { s.sessionExpired.Store(v) }

// SessionExpired reports whether protected routes should reject as expired.
func (s *Scenario) SessionExpired() bool { return s.sessionExpired.Load() }

// SetOrderReplyConfirm toggles the submit-order reply-confirmation flow.
func (s *Scenario) SetOrderReplyConfirm(v bool) { s.orderReply.Store(v) }

// OrderReplyConfirm reports whether submit-order should return a reply.
func (s *Scenario) OrderReplyConfirm() bool { return s.orderReply.Load() }

// SetOrderReplyBody overrides the body returned when order replies are enabled.
func (s *Scenario) SetOrderReplyBody(body string) { s.orderReplyBody.Store(body) }

// OrderReplyBody returns the configured order-reply body, or "" for the
// built-in default.
func (s *Scenario) OrderReplyBody() string {
	if v, ok := s.orderReplyBody.Load().(string); ok {
		return v
	}
	return ""
}

// FaultFor resolves the fault to apply for op, preferring a policy result,
// then the per-op fault, then the global fault.
func (s *Scenario) FaultFor(op string, req *Request) *Fault {
	s.mu.RLock()
	policy := s.onRequest
	opFault := s.ops[op]
	global := s.global
	s.mu.RUnlock()

	if policy != nil {
		if f := policy(op, req); f != nil {
			return f
		}
	}
	if opFault != nil {
		return opFault
	}
	return global
}
