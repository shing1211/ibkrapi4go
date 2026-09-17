// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type tokenServer struct {
	*httptest.Server
	calls  atomic.Int64
	mu     sync.Mutex
	forms  []map[string]string
	delay  time.Duration
	tokens []string // access tokens returned in order
	refs   []string // refresh tokens returned in order (may be "")
}

func newTokenServer(t *testing.T, tokens, refs []string) *tokenServer {
	t.Helper()
	ts := &tokenServer{tokens: tokens, refs: refs}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := ts.calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		vals, _ := parseForm(string(body))
		ts.mu.Lock()
		ts.forms = append(ts.forms, vals)
		ts.mu.Unlock()
		if ts.delay > 0 {
			time.Sleep(ts.delay)
		}
		i := int(n) - 1
		if i >= len(ts.tokens) {
			i = len(ts.tokens) - 1
		}
		ref := ""
		if i < len(ts.refs) {
			ref = ts.refs[i]
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":%q,"expires_in":%d,"refresh_token":%q,"token_type":"Bearer"}`, ts.tokens[i], 3600, ref)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func parseForm(body string) (map[string]string, error) {
	vals, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range vals {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out, nil
}

func (ts *tokenServer) lastForm(t *testing.T) map[string]string {
	t.Helper()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if len(ts.forms) == 0 {
		t.Fatal("no token requests recorded")
	}
	return ts.forms[len(ts.forms)-1]
}

func TestTokenSource_ClientCredentials(t *testing.T) {
	srv := newTokenServer(t, []string{"tok-1"}, []string{""})
	ts := NewTokenSource(OAuthConfig{TokenURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})

	tok, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok != "tok-1" {
		t.Errorf("token = %q; want tok-1", tok)
	}
	form := srv.lastForm(t)
	if form["grant_type"] != "client_credentials" || form["client_id"] != "cid" || form["client_secret"] != "sec" {
		t.Errorf("form = %v; want client_credentials with cid/sec", form)
	}
	// Second call is served from cache.
	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if srv.calls.Load() != 1 {
		t.Errorf("token requests = %d; want 1 (cached)", srv.calls.Load())
	}
}

func TestTokenSource_RefreshesNearExpiry(t *testing.T) {
	srv := newTokenServer(t, []string{"tok-1", "tok-2"}, []string{"", ""})
	ts := NewTokenSource(OAuthConfig{
		TokenURL:     srv.URL,
		ClientID:     "cid",
		EarlyRefresh: time.Hour, // always considered near expiry
	})

	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token 1: %v", err)
	}
	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token 2: %v", err)
	}
	if srv.calls.Load() != 2 {
		t.Errorf("token requests = %d; want 2 (early refresh)", srv.calls.Load())
	}
}

func TestTokenSource_SingleFlight(t *testing.T) {
	srv := newTokenServer(t, []string{"tok-1"}, []string{""})
	srv.delay = 100 * time.Millisecond
	ts := NewTokenSource(OAuthConfig{TokenURL: srv.URL, ClientID: "cid"})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ts.Token(context.Background()); err != nil {
				t.Errorf("Token: %v", err)
			}
		}()
	}
	wg.Wait()
	if srv.calls.Load() != 1 {
		t.Errorf("token requests = %d; want 1 (single-flight)", srv.calls.Load())
	}
}

func TestTokenSource_Rotation(t *testing.T) {
	srv := newTokenServer(t, []string{"tok-1", "tok-2"}, []string{"refresh-1", "refresh-2"})
	ts := NewTokenSource(OAuthConfig{TokenURL: srv.URL, ClientID: "cid"})

	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token 1: %v", err)
	}
	// First response rotated in refresh-1.
	if got := ts.RefreshToken(); got != "refresh-1" {
		t.Fatalf("RefreshToken = %q; want refresh-1", got)
	}
	if _, err := ts.ForceRefresh(context.Background()); err != nil {
		t.Fatalf("ForceRefresh: %v", err)
	}
	form := srv.lastForm(t)
	if form["grant_type"] != "refresh_token" || form["refresh_token"] != "refresh-1" {
		t.Errorf("refresh form = %v; want refresh_token grant with refresh-1", form)
	}
	if got := ts.RefreshToken(); got != "refresh-2" {
		t.Errorf("RefreshToken = %q; want refresh-2 (rotated)", got)
	}
}

func TestTokenSource_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid_client"}`)
	}))
	defer srv.Close()

	ts := NewTokenSource(OAuthConfig{TokenURL: srv.URL, ClientID: "bad"})
	if _, err := ts.Token(context.Background()); err == nil {
		t.Fatal("Token = nil; want error")
	}
}

func TestTokenSource_JWTAssertion(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	var capturedAssertion, capturedGrantType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		vals, _ := parseForm(string(body))
		capturedGrantType = vals["grant_type"]
		capturedAssertion = vals["client_assertion"]
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":%q,"expires_in":3600,"refresh_token":%q,"token_type":"Bearer"}`, "jwt-tok", "jwt-refresh")
	}))
	defer srv.Close()

	ts := NewTokenSource(OAuthConfig{
		TokenURL:  srv.URL,
		ClientID:  "jwt-cid",
		JWTKey:    privateKey,
		JWTExpiry: 5 * time.Minute,
	})

	tok, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok != "jwt-tok" {
		t.Errorf("token = %q; want jwt-tok", tok)
	}
	if capturedGrantType != "client_credentials" {
		t.Errorf("grant_type = %q; want client_credentials", capturedGrantType)
	}
	if capturedAssertion == "" {
		t.Fatal("client_assertion is empty; want a JWT")
	}
	// Verify JWT structure: header.payload.signature
	parts := strings.Split(capturedAssertion, ".")
	if len(parts) != 3 {
		t.Fatalf("assertion has %d parts; want 3 (header.payload.signature)", len(parts))
	}
	// Decode and verify header contains RS256
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode JWT header: %v", err)
	}
	if !strings.Contains(string(headerBytes), `"alg":"RS256"`) {
		t.Errorf("JWT header = %s; want RS256 alg", string(headerBytes))
	}
}

func TestTokenSource_JWTAssertion_SingleFlight(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"tok-jwt","expires_in":3600}`)
	}))
	defer srv.Close()

	ts := NewTokenSource(OAuthConfig{
		TokenURL: srv.URL,
		ClientID: "cid",
		JWTKey:   privateKey,
	})

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ts.Token(context.Background()); err != nil {
				t.Errorf("Token: %v", err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Errorf("token requests = %d; want 1 (single-flight)", calls)
	}
}

func TestTokenSource_JWTAssertion_PEM(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	var capturedAssertion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		vals, _ := parseForm(string(body))
		capturedAssertion = vals["client_assertion"]
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"tok-jwt","expires_in":3600}`)
	}))
	defer srv.Close()

	// Encode private key as PKCS8 PEM
	pemBytes, err := EncodePrivateKeyToPEM(privateKey)
	if err != nil {
		t.Fatalf("EncodePrivateKeyToPEM: %v", err)
	}

	ts := NewTokenSource(OAuthConfig{
		TokenURL:  srv.URL,
		ClientID:  "cid",
		JWTKeyPEM: pemBytes,
	})

	_, err = ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if capturedAssertion == "" {
		t.Fatal("client_assertion is empty")
	}
}

// EncodePrivateKeyToPEM encodes an RSA private key to PEM format.
func EncodePrivateKeyToPEM(key *rsa.PrivateKey) ([]byte, error) {
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}
	out := &strings.Builder{}
	if err := pem.Encode(out, block); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}
