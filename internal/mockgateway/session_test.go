// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The mock gateway is the substrate every pkg/ibkr test runs against, so its auth
// decisions are load-bearing: if the mock accepts a request the real gateway would
// reject, a test passes that should not. These tests pin that behaviour directly
// rather than trusting the suite that depends on it.

func reqWith(headers map[string]string) *Request {
	h := make(http.Header, len(headers))
	for k, v := range headers {
		h.Set(k, v)
	}
	return &Request{Method: "GET", Path: "/", Headers: h}
}

// TestSessionStore_TokenAloneIsNotEnough pins the subtle rule in isAuthenticated:
// a request carrying the correct token is still rejected until a session has been
// established. Reading the check as "token matches" would let any caller who
// guessed the token in, which for a default-token mock is not a real secret.
func TestSessionStore_TokenAloneIsNotEnough(t *testing.T) {
	var s sessionStore

	token := s.authenticate()
	if token != defaultSessionToken {
		t.Fatalf("authenticate() issued %q; want %q", token, defaultSessionToken)
	}
	if !s.isAuthenticated(reqWith(map[string]string{"Authorization": token})) {
		t.Error("a matching Authorization header was rejected after authenticate()")
	}

	// A fresh store has the same default token available, but no session.
	var fresh sessionStore
	if got := fresh.currentToken(); got != defaultSessionToken {
		t.Errorf("currentToken() on a fresh store = %q; want %q", got, defaultSessionToken)
	}
	if fresh.isAuthenticated(reqWith(map[string]string{"Authorization": defaultSessionToken})) {
		t.Error("a fresh store accepted the default token; the session must be established first")
	}
	if fresh.isAuthenticated(reqWith(nil)) {
		t.Error("a fresh store authenticated a request with no credentials at all")
	}
}

func TestSessionStore_AcceptsCookieOrHeaderOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"matching Authorization header", map[string]string{"Authorization": defaultSessionToken}, true},
		{"matching session cookie", map[string]string{"Cookie": sessionCookie + "=" + defaultSessionToken}, true},
		{"wrong Authorization header", map[string]string{"Authorization": "wrong"}, false},
		{"wrong cookie value", map[string]string{"Cookie": sessionCookie + "=wrong"}, false},
		{"right cookie, wrong name", map[string]string{"Cookie": "other=" + defaultSessionToken}, false},
		{"no credentials", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s sessionStore
			s.authenticate()
			if got := s.isAuthenticated(reqWith(tc.headers)); got != tc.want {
				t.Errorf("isAuthenticated() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestSessionStore_ClearRevokesTheToken(t *testing.T) {
	var s sessionStore
	token := s.authenticate()
	if !s.isAuthenticated(reqWith(map[string]string{"Authorization": token})) {
		t.Fatal("not authenticated after authenticate()")
	}

	s.clear()
	if s.isAuthenticated(reqWith(map[string]string{"Authorization": token})) {
		t.Error("the old token still authenticated after clear()")
	}

	// clear must also drop the stored token, not just the flag, so a later
	// authenticate() cannot resurrect a revoked value.
	if got := s.currentToken(); got != defaultSessionToken {
		t.Errorf("currentToken() after clear() = %q; want the default %q", got, defaultSessionToken)
	}
}

// TestSessionStore_ClearDropsANonDefaultToken covers the part of clear() that the
// default-token case cannot observe. authenticate() only ever hands out
// defaultSessionToken, so with the default in play, forgetting to clear the stored
// token is indistinguishable from clearing it. The store's own comment anticipates a
// seed-specific token, and that is the case where the difference is load-bearing: a
// revoked non-default token must stop being the current token.
func TestSessionStore_ClearDropsANonDefaultToken(t *testing.T) {
	const seeded = "seeded-session-token"

	var s sessionStore
	s.token = seeded
	s.authenticated = true

	if got := s.currentToken(); got != seeded {
		t.Fatalf("currentToken() = %q; want %q", got, seeded)
	}
	if !s.isAuthenticated(reqWith(map[string]string{"Authorization": seeded})) {
		t.Fatal("the seeded token did not authenticate")
	}

	s.clear()

	if got := s.currentToken(); got != defaultSessionToken {
		t.Errorf("currentToken() after clear() = %q; want the default %q - the revoked token was kept",
			got, defaultSessionToken)
	}
	if s.isAuthenticated(reqWith(map[string]string{"Authorization": seeded})) {
		t.Error("the revoked seeded token still authenticated")
	}
}

// TestIsSessionOp_PinnedAllowlist pins the exact set of operations that bypass
// authentication. This is an allowlist, so every entry is an unauthenticated
// endpoint: an operation added here becomes reachable with no session at all. A
// new operation must default to protected, which is what the zero value of the
// switch gives - this test exists so that stays deliberate.
func TestIsSessionOp_PinnedAllowlist(t *testing.T) {
	sessionOps := []string{
		OpInitializeSession,
		OpGetBrokerageStatus,
		OpGetSessionToken,
		OpLogout,
		OpGetSessionValidation,
	}

	for _, op := range sessionOps {
		if !isSessionOp(op) {
			t.Errorf("isSessionOp(%q) = false; the session endpoint would demand a session", op)
		}
	}

	// A representative set of operations that must stay authenticated.
	for _, op := range []string{
		OpGetAllAccounts, OpGetAccountSummary, OpGetPortfolioSummary,
		OpGetOpenOrders, OpSubmitNewOrder, OpCancelOpenOrder,
	} {
		if isSessionOp(op) {
			t.Errorf("isSessionOp(%q) = true; a protected operation would bypass authentication", op)
		}
	}
}

func TestServeSession_InitIssuesCookie(t *testing.T) {
	s := New()
	rec := httptest.NewRecorder()

	s.serveSession(rec, reqWith(nil), OpInitializeSession)

	if rec.Code != http.StatusOK {
		t.Fatalf("init status = %d; want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Body.String(), `{"authenticated":true,"established":true}`; got != want {
		t.Errorf("init body = %s; want %s", got, want)
	}

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			found = true
			if c.Value != defaultSessionToken {
				t.Errorf("cookie value = %q; want %q", c.Value, defaultSessionToken)
			}
			if !c.HttpOnly {
				t.Error("session cookie is not HttpOnly")
			}
		}
	}
	if !found {
		t.Fatalf("init did not set the %q cookie (headers: %v)", sessionCookie, rec.Header())
	}
}

// TestServeSession_ProtectedOpsRejectUnauthenticated covers the three session
// operations that are themselves guarded when authRequired is set. They are in
// isSessionOp so they are exempt from the blanket check in ServeHTTP, and carry
// their own - if that inner check were dropped, these would answer anonymous
// callers.
func TestServeSession_ProtectedOpsRejectUnauthenticated(t *testing.T) {
	for _, op := range []string{OpGetBrokerageStatus, OpGetSessionToken, OpGetSessionValidation} {
		t.Run(op, func(t *testing.T) {
			s := New(WithAuthRequired(true))
			rec := httptest.NewRecorder()

			s.serveSession(rec, reqWith(nil), op)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
			}
			if got, want := rec.Body.String(), `{"error":"not authenticated"}`; got != want {
				t.Errorf("body = %s; want %s", got, want)
			}
		})
	}
}

func TestServeSession_ProtectedOpsAcceptSession(t *testing.T) {
	for op, want := range map[string]string{
		OpGetBrokerageStatus:   `{"authenticated":true,"established":true,"connected":true}`,
		OpGetSessionToken:      `{"session":"` + defaultSessionToken + `"}`,
		OpGetSessionValidation: `{"valid":true,"message":"ok"}`,
	} {
		t.Run(op, func(t *testing.T) {
			s := New(WithAuthRequired(true))
			s.sessions.authenticate()
			rec := httptest.NewRecorder()

			s.serveSession(rec, reqWith(map[string]string{
				"Authorization": defaultSessionToken,
			}), op)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Body.String(); got != want {
				t.Errorf("body = %s; want %s", got, want)
			}
		})
	}
}

// TestServeSession_LogoutRevokesAccess is the end-to-end shape of a logout: the
// call succeeds, and the session it held stops working on the next call. Asserting
// only the 200 would pass even if clear() were never reached.
func TestServeSession_LogoutRevokesAccess(t *testing.T) {
	s := New(WithAuthRequired(true))
	rec := httptest.NewRecorder()

	s.serveSession(rec, reqWith(map[string]string{"Authorization": defaultSessionToken}),
		OpInitializeSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("init status = %d; want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	s.serveSession(rec, reqWith(map[string]string{"Authorization": defaultSessionToken}), OpLogout)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status = %d; want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	s.serveSession(rec, reqWith(map[string]string{"Authorization": defaultSessionToken}),
		OpGetSessionToken)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("after logout, status = %d; want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestServeSession_UnknownOpIs404(t *testing.T) {
	s := New()
	rec := httptest.NewRecorder()

	s.serveSession(rec, reqWith(nil), "notASessionOp")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusNotFound)
	}
	if got, want := rec.Body.String(), `{"error":"unknown path"}`; got != want {
		t.Errorf("body = %s; want %s", got, want)
	}
}

// TestServer_AuthRequiredEndToEnd exercises the property through the real handler:
// a protected route is refused with no session and served once ssodh/init has
// issued one. This is the guarantee the pkg/ibkr auth tests lean on, so it is
// asserted here at the layer that implements it.
func TestServer_AuthRequiredEndToEnd(t *testing.T) {
	const protectedPath = "/v1/api/iserver/currency/pairs" // OpGetCurrencyPairs

	t.Run("without auth enforcement a bare request is served", func(t *testing.T) {
		s := New()
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, protectedPath, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d (auth enforcement is off by default)",
				rec.Code, http.StatusOK)
		}
	})

	t.Run("enforced, an anonymous request is refused", func(t *testing.T) {
		s := New(WithAuthRequired(true))
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, protectedPath, nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
		}
		if got, want := rec.Body.String(), `{"error":"not authenticated"}`; got != want {
			t.Errorf("body = %s; want %s", got, want)
		}
	})

	t.Run("enforced, a forged token is refused", func(t *testing.T) {
		s := New(WithAuthRequired(true))
		req := httptest.NewRequest(http.MethodGet, protectedPath, nil)
		req.Header.Set("Authorization", "not-the-token")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("enforced, an established session is served", func(t *testing.T) {
		s := New(WithAuthRequired(true))

		// Establish the session the way a client would.
		initRec := httptest.NewRecorder()
		s.Handler().ServeHTTP(initRec,
			httptest.NewRequest(http.MethodPost, "/v1/api/iserver/auth/ssodh/init", nil))
		if initRec.Code != http.StatusOK {
			t.Fatalf("init status = %d; want %d", initRec.Code, http.StatusOK)
		}

		req := httptest.NewRequest(http.MethodGet, protectedPath, nil)
		for _, c := range initRec.Result().Cookies() {
			if c.Name == sessionCookie {
				req.AddCookie(c)
			}
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d with a session cookie", rec.Code, http.StatusOK)
		}
	})
}
