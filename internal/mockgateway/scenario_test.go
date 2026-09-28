// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newBufReader(c net.Conn) *bufio.Reader { return bufio.NewReader(c) }

// Fault injection is what makes the SDK's retry, error-classification and
// order-reply tests meaningful. A defect here does not fail loudly: it stops
// injecting, the SDK sees a clean fixture response instead, and the test it was
// written for passes without ever exercising the path. So the injection is
// asserted to actually reach the client, not just to be settable.

// TestScenario_FaultForPrecedence pins the documented resolution order:
// policy, then per-op, then global, then nothing. Every pkg/ibkr retry test
// installs a policy, so a change to this order would silently change what those
// tests inject.
func TestScenario_FaultForPrecedence(t *testing.T) {
	global := &Fault{Status: http.StatusServiceUnavailable, Body: "global"}
	perOp := &Fault{Status: http.StatusBadGateway, Body: "per-op"}
	fromPolicy := &Fault{Status: http.StatusTeapot, Body: "policy"}

	newScenario := func() *Scenario { return NewScenario() }

	t.Run("nothing configured yields no fault", func(t *testing.T) {
		s := newScenario()
		if f := s.FaultFor(OpGetOpenOrders, nil); f != nil {
			t.Errorf("FaultFor on an empty scenario = %+v; want nil", f)
		}
	})

	t.Run("global applies to any op", func(t *testing.T) {
		s := newScenario()
		s.SetGlobal(global)
		if f := s.FaultFor(OpGetOpenOrders, nil); f != global {
			t.Errorf("FaultFor = %+v; want the global fault", f)
		}
		s.ClearGlobal()
		if f := s.FaultFor(OpGetOpenOrders, nil); f != nil {
			t.Errorf("after ClearGlobal, FaultFor = %+v; want nil", f)
		}
	})

	t.Run("per-op beats global, and only for its own op", func(t *testing.T) {
		s := newScenario()
		s.SetGlobal(global)
		s.Set(OpGetOpenOrders, perOp)

		if f := s.FaultFor(OpGetOpenOrders, nil); f != perOp {
			t.Errorf("FaultFor(%s) = %+v; want the per-op fault", OpGetOpenOrders, f)
		}
		if f := s.FaultFor(OpSubmitNewOrder, nil); f != global {
			t.Errorf("FaultFor(%s) = %+v; want the global fault", OpSubmitNewOrder, f)
		}

		s.Clear(OpGetOpenOrders)
		if f := s.FaultFor(OpGetOpenOrders, nil); f != global {
			t.Errorf("after Clear, FaultFor(%s) = %+v; want the global fault",
				OpGetOpenOrders, f)
		}
	})

	t.Run("policy beats everything, and returning nil defers", func(t *testing.T) {
		s := newScenario()
		s.SetGlobal(global)
		s.Set(OpGetOpenOrders, perOp)

		s.SetPolicy(func(op string, _ *Request) *Fault { return fromPolicy })
		if f := s.FaultFor(OpGetOpenOrders, nil); f != fromPolicy {
			t.Errorf("FaultFor = %+v; want the policy fault", f)
		}

		// A policy that declines must fall through, not suppress the fault.
		s.SetPolicy(func(string, *Request) *Fault { return nil })
		if f := s.FaultFor(OpGetOpenOrders, nil); f != perOp {
			t.Errorf("after the policy declined, FaultFor = %+v; want the per-op fault", f)
		}
	})

	t.Run("the policy receives the op and the request", func(t *testing.T) {
		s := newScenario()
		var gotOp string
		var gotReq *Request
		req := reqWith(nil)
		s.SetPolicy(func(op string, r *Request) *Fault {
			gotOp, gotReq = op, r
			return nil
		})
		s.FaultFor(OpCancelOpenOrder, req)

		if gotOp != OpCancelOpenOrder {
			t.Errorf("policy saw op %q; want %q", gotOp, OpCancelOpenOrder)
		}
		if gotReq != req {
			t.Error("policy did not receive the request it was given")
		}
	})
}

func TestScenario_FlagsRoundTrip(t *testing.T) {
	s := NewScenario()

	if s.SessionExpired() {
		t.Error("SessionExpired() is true on a new scenario")
	}
	s.SetSessionExpired(true)
	if !s.SessionExpired() {
		t.Error("SetSessionExpired(true) did not take")
	}
	s.SetSessionExpired(false)
	if s.SessionExpired() {
		t.Error("SetSessionExpired(false) did not take")
	}

	if s.OrderReplyConfirm() {
		t.Error("OrderReplyConfirm() is true on a new scenario")
	}
	s.SetOrderReplyConfirm(true)
	if !s.OrderReplyConfirm() {
		t.Error("SetOrderReplyConfirm(true) did not take")
	}

	// An unset reply body must read as the empty string, which is the signal the
	// server uses to fall back to its built-in default.
	if got := s.OrderReplyBody(); got != "" {
		t.Errorf("OrderReplyBody() on a new scenario = %q; want \"\"", got)
	}
	s.SetOrderReplyBody(`[{"id":"r1"}]`)
	if got := s.OrderReplyBody(); got != `[{"id":"r1"}]` {
		t.Errorf("OrderReplyBody() = %q; want the configured body", got)
	}
}

func TestScenario_LatencyRoundTrip(t *testing.T) {
	s := NewScenario()
	if got := s.Latency(); got != 0 {
		t.Errorf("Latency() = %v; want 0", got)
	}
	s.SetLatency(25 * time.Millisecond)
	if got := s.Latency(); got != 25*time.Millisecond {
		t.Errorf("Latency() = %v; want 25ms", got)
	}
	s.SetLatency(0)
	if got := s.Latency(); got != 0 {
		t.Errorf("Latency() after reset = %v; want 0", got)
	}
}

// TestScenario_ZeroValueScenarioIsUsable guards NewScenario's own initialization.
// Set lazily creates s.ops, so a Scenario built without NewScenario must still
// work - and FaultFor must not panic on a nil map.
func TestScenario_ZeroValueScenarioIsUsable(t *testing.T) {
	var s Scenario

	if f := s.FaultFor(OpGetOpenOrders, nil); f != nil {
		t.Errorf("FaultFor on a zero-value scenario = %+v; want nil", f)
	}
	s.Set(OpGetOpenOrders, &Fault{Status: http.StatusTeapot})
	if f := s.FaultFor(OpGetOpenOrders, nil); f == nil {
		t.Error("Set on a zero-value scenario did not take; ops map was not created")
	}
}

// TestServer_InjectedStatusReplacesTheFixture is the end-to-end check that matters
// most: a fault must actually reach the client. A scenario that silently failed to
// inject would serve the fixture, and an SDK retry test written against it would
// pass without exercising anything.
func TestServer_InjectedStatusReplacesTheFixture(t *testing.T) {
	const path = "/v1/api/iserver/currency/pairs" // OpGetCurrencyPairs

	s := New()
	scn := s.Scenario()
	scn.Set(OpGetCurrencyPairs, &Fault{
		Status: http.StatusTooManyRequests,
		Body:   `{"error":"slow down"}`,
	})

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d; want %d - the fault did not replace the fixture",
			rec.Code, http.StatusTooManyRequests)
	}
	if got := rec.Body.String(); got != `{"error":"slow down"}` {
		t.Errorf("body = %s; want the injected body", got)
	}

	// Clearing the fault must restore the fixture.
	scn.Clear(OpGetCurrencyPairs)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("after Clear, status = %d; want %d", rec.Code, http.StatusOK)
	}
}

// TestServer_InjectedFaultWithNoBodyUsesADefault covers the fallback body, so a
// fault configured with only a status still produces a parseable JSON error rather
// than an empty response.
func TestServer_InjectedFaultWithNoBodyUsesADefault(t *testing.T) {
	s := New()
	s.Scenario().Set(OpGetCurrencyPairs, &Fault{Status: http.StatusInternalServerError})

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/api/iserver/currency/pairs", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Body.String(); got != `{"error":"injected fault"}` {
		t.Errorf("body = %q; want the default injected-fault body", got)
	}
}

// TestServer_InjectedLatencyIsApplied measures the delay rather than trusting the
// setter, and checks that a per-fault delay wins over the global one - the max()
// behaviour SDK timeout tests rely on.
func TestServer_InjectedLatencyIsApplied(t *testing.T) {
	const path = "/v1/api/iserver/currency/pairs"

	t.Run("global latency delays the response", func(t *testing.T) {
		s := New()
		s.Scenario().SetLatency(60 * time.Millisecond)

		start := time.Now()
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		elapsed := time.Since(start)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if elapsed < 40*time.Millisecond {
			t.Errorf("elapsed = %v; want at least ~60ms - the latency was not applied", elapsed)
		}
	})

	t.Run("a larger per-fault delay wins over the global", func(t *testing.T) {
		s := New()
		s.Scenario().SetLatency(0)
		s.Scenario().Set(OpGetCurrencyPairs, &Fault{
			Status: http.StatusTeapot,
			Delay:  70 * time.Millisecond,
		})

		start := time.Now()
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		elapsed := time.Since(start)

		if rec.Code != http.StatusTeapot {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusTeapot)
		}
		if elapsed < 40*time.Millisecond {
			t.Errorf("elapsed = %v; want at least ~70ms - the fault delay was not applied", elapsed)
		}
	})
}

// TestServer_DropConnectionFaultClosesWithoutResponding covers the ambiguous
// transport failure the SDK must not retry blindly. It is asserted through a real
// TCP connection, because a recorder would not notice a connection being hijacked.
func TestServer_DropConnectionFaultClosesWithoutResponding(t *testing.T) {
	s := New()
	s.Scenario().Set(OpGetCurrencyPairs, &Fault{DropConnection: true})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	conn, err := net.DialTimeout("tcp", ts.Listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/api/iserver/currency/pairs", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := req.Write(conn); err != nil {
		t.Fatalf("write request: %v", err)
	}

	// The server closes without a status line, so reading must fail rather than
	// yield a response.
	if _, err := http.ReadResponse(newBufReader(conn), req); err == nil {
		t.Fatal("got an HTTP response; want the connection dropped without one")
	}
}

// TestServer_TimeoutFaultBlocksUntilContextCancel covers the transport-timeout
// simulation. It must honour the client's context: a timeout that outlived its
// context would make every SDK timeout test hang rather than pass.
func TestServer_TimeoutFaultBlocksUntilContextCancel(t *testing.T) {
	s := New()
	s.Scenario().Set(OpGetCurrencyPairs, &Fault{Timeout: true})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ts.URL+"/v1/api/iserver/currency/pairs", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err == nil {
		resp.Body.Close()
		t.Fatalf("got status %d; want the request to fail on context cancellation", resp.StatusCode)
	}
	if elapsed > 3*time.Second {
		t.Errorf("elapsed = %v; the timeout fault outlived the client context", elapsed)
	}
	if ctx.Err() == nil {
		t.Error("the client context was never cancelled")
	}
}

// TestServer_SessionExpiredFaultFiresOnAnAuthenticatedRoute covers the
// session-expired switch. It is checked inside the protected-route block but is
// independent of authRequired, so it fires on its own - and it must stop firing
// once cleared, or a scenario left switched on would silently wedge every later
// test sharing that gateway.
func TestServer_SessionExpiredFaultFiresOnAnAuthenticatedRoute(t *testing.T) {
	const path = "/v1/api/iserver/currency/pairs"

	s := New()
	s.Scenario().SetSessionExpired(true)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Body.String(); got != `{"error":"session expired"}` {
		t.Errorf("body = %s; want the session-expired body", got)
	}

	s.Scenario().SetSessionExpired(false)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("after clearing, status = %d; want %d - the fixture was never restored",
			rec.Code, http.StatusOK)
	}
}
