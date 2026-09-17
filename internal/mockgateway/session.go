// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"net/http"
	"sync"
)

// sessionCookie is the cookie set issued by ssodh/init and accepted on
// protected routes.
const sessionCookie = "ibkr_mock_session"

// defaultSessionToken is issued when no seed-specific token is configured.
const defaultSessionToken = "tok-123"

// sessionStore tracks the mock brokerage session.
type sessionStore struct {
	mu            sync.Mutex
	token         string
	authenticated bool
}

func (s *sessionStore) authenticate() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		s.token = defaultSessionToken
	}
	s.authenticated = true
	return s.token
}

func (s *sessionStore) currentToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		return defaultSessionToken
	}
	return s.token
}

func (s *sessionStore) clear() {
	s.mu.Lock()
	s.authenticated = false
	s.token = ""
	s.mu.Unlock()
}

// isAuthenticated reports whether req carries the session cookie or a matching
// Authorization header.
func (s *sessionStore) isAuthenticated(req *Request) bool {
	s.mu.Lock()
	authed := s.authenticated
	token := s.token
	s.mu.Unlock()
	if !authed {
		return false
	}
	if c, err := (&http.Request{Header: req.Headers}).Cookie(sessionCookie); err == nil && c.Value == token {
		return true
	}
	return req.Headers.Get("Authorization") == token
}

// isSessionOp reports whether op is one of the unauthenticated session routes.
func isSessionOp(op string) bool {
	switch op {
	case OpInitializeSession, OpGetBrokerageStatus, OpGetSessionToken, OpLogout, OpGetSessionValidation:
		return true
	default:
		return false
	}
}

// serveSession handles the session/auth endpoints, which mutate auth state and
// therefore cannot be served from a static fixture.
func (s *Server) serveSession(w http.ResponseWriter, req *Request, op string) {
	switch op {
	case OpInitializeSession:
		token := s.sessions.authenticate()
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true})
		writeJSON(w, http.StatusOK, `{"authenticated":true,"established":true}`)
	case OpGetBrokerageStatus:
		if s.authRequired && !s.sessions.isAuthenticated(req) {
			writeJSON(w, http.StatusUnauthorized, `{"error":"not authenticated"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"authenticated":true,"established":true,"connected":true}`)
	case OpGetSessionToken:
		if s.authRequired && !s.sessions.isAuthenticated(req) {
			writeJSON(w, http.StatusUnauthorized, `{"error":"not authenticated"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"session":"`+s.sessions.currentToken()+`"}`)
	case OpLogout:
		s.sessions.clear()
		writeJSON(w, http.StatusOK, `{}`)
	case OpGetSessionValidation:
		if s.authRequired && !s.sessions.isAuthenticated(req) {
			writeJSON(w, http.StatusUnauthorized, `{"error":"not authenticated"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"valid":true,"message":"ok"}`)
	default:
		writeJSON(w, http.StatusNotFound, `{"error":"unknown path"}`)
	}
}
