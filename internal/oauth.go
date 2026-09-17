// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultTokenURL is the IBKR OAuth2 token endpoint.
const DefaultTokenURL = "https://api.ibkr.com/oauth2/api/v1/token"

// OAuthConfig configures an OAuth2 TokenSource.
type OAuthConfig struct {
	// TokenURL is the token endpoint. Defaults to DefaultTokenURL.
	TokenURL string
	// ClientID is the OAuth2 client id.
	ClientID string
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string
	// RefreshToken, when set, selects the refresh_token grant.
	RefreshToken string
	// Scope is the requested scope, if any.
	Scope string
	// HTTPClient performs the token requests. It must NOT inject the bearer
	// token (to avoid recursion). Defaults to a client with a 15s timeout.
	HTTPClient *http.Client
	// EarlyRefresh refreshes this long before expiry. Defaults to 30s.
	EarlyRefresh time.Duration
	// Logger receives token refresh diagnostics. Nil disables logging.
	Logger *slog.Logger

	// JWTKey is the RSA private key for JWT-bearer token exchange.
	// When set, the token source uses client_assertion grant (private_key_jwt)
	// instead of client_credentials or refresh_token.
	JWTKey *rsa.PrivateKey
	// JWTKeyPEM is raw PEM-encoded RSA private key bytes.
	// Mutually exclusive with JWTKey and JWTKeyFile.
	JWTKeyPEM []byte
	// JWTExpiry sets the JWT assertion expiry. Defaults to 60s.
	JWTExpiry time.Duration
}

// TokenSource acquires and refreshes OAuth2 access tokens. It is safe for
// concurrent use and serializes refreshes (single-flight).
type TokenSource struct {
	cfg    OAuthConfig
	client *http.Client
	jwtKey *rsa.PrivateKey
	logger *slog.Logger

	mu           sync.Mutex
	token        string
	expiry       time.Time
	refreshToken string
	lastErr      error
	inflight     chan struct{}
}

// NewTokenSource builds a TokenSource from cfg.
func NewTokenSource(cfg OAuthConfig) *TokenSource {
	if cfg.TokenURL == "" {
		cfg.TokenURL = DefaultTokenURL
	}
	if cfg.EarlyRefresh <= 0 {
		cfg.EarlyRefresh = 30 * time.Second
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.Logger == nil {
		cfg.Logger = NopLogger()
	}
	var jwtKey *rsa.PrivateKey
	switch {
	case cfg.JWTKey != nil:
		jwtKey = cfg.JWTKey
	case len(cfg.JWTKeyPEM) > 0:
		var err error
		jwtKey, err = ParsePrivateKeyFromPEM(cfg.JWTKeyPEM)
		if err != nil {
			jwtKey = nil
		}
	}
	return &TokenSource{cfg: cfg, client: hc, jwtKey: jwtKey, logger: cfg.Logger, refreshToken: cfg.RefreshToken}
}

// RefreshToken returns the current refresh token (which may have rotated).
func (ts *TokenSource) RefreshToken() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.refreshToken
}

// Token returns a valid access token, refreshing it when missing or near expiry.
func (ts *TokenSource) Token(ctx context.Context) (string, error) {
	ts.mu.Lock()
	if ts.token != "" && time.Now().Before(ts.expiry.Add(-ts.cfg.EarlyRefresh)) {
		tok := ts.token
		ts.mu.Unlock()
		return tok, nil
	}
	if ch := ts.inflight; ch != nil {
		ts.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		ts.mu.Lock()
		tok, err := ts.token, ts.lastErr
		ts.mu.Unlock()
		return tok, err
	}
	ch := make(chan struct{})
	ts.inflight = ch
	ts.mu.Unlock()

	tok, expiry, newRefresh, err := ts.fetch(ctx)

	ts.mu.Lock()
	ts.inflight = nil
	ts.lastErr = err
	if err == nil {
		ts.token = tok
		ts.expiry = expiry
		if newRefresh != "" {
			ts.refreshToken = newRefresh
		}
	}
	close(ch)
	ts.mu.Unlock()
	if err != nil {
		ts.logger.Warn("ibkr.oauth token refresh failed", "err", redact(err.Error()))
	} else {
		ts.logger.Debug("ibkr.oauth token refreshed", "expires_at", expiry.Format(time.RFC3339))
	}
	return tok, err
}

// ForceRefresh discards the cached token and fetches a new one.
func (ts *TokenSource) ForceRefresh(ctx context.Context) (string, error) {
	ts.mu.Lock()
	ts.token = ""
	ts.expiry = time.Time{}
	ts.mu.Unlock()
	return ts.Token(ctx)
}

func (ts *TokenSource) fetch(ctx context.Context) (string, time.Time, string, error) {
	if ts.jwtKey != nil {
		return ts.fetchWithJWTAssertion(ctx)
	}
	return ts.fetchWithSecret(ctx)
}

func (ts *TokenSource) fetchWithSecret(ctx context.Context) (string, time.Time, string, error) {
	form := url.Values{}
	form.Set("client_id", ts.cfg.ClientID)
	form.Set("client_secret", ts.cfg.ClientSecret)
	if ts.cfg.Scope != "" {
		form.Set("scope", ts.cfg.Scope)
	}
	ts.mu.Lock()
	refresh := ts.refreshToken
	ts.mu.Unlock()
	if refresh != "" {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", refresh)
	} else {
		form.Set("grant_type", "client_credentials")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.cfg.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: err.Error(), Err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := ts.client.Do(req)
	if err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: err.Error(), Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		return "", time.Time{}, "", &Error{
			Op:         "OAuth.Token",
			Message:    fmt.Sprintf("token endpoint returned %d: %s", resp.StatusCode, redact(strings.TrimSpace(string(body)))),
			HTTPStatus: resp.StatusCode,
			Err:        statusSentinel[resp.StatusCode],
		}
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: "decode token response: " + err.Error(), Err: err}
	}
	if tr.AccessToken == "" {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: "token response missing access_token", Err: ErrNotAuthenticated}
	}
	expiry := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn <= 0 {
		expiry = time.Now().Add(time.Minute)
	}
	return tr.AccessToken, expiry, tr.RefreshToken, nil
}

func (ts *TokenSource) fetchWithJWTAssertion(ctx context.Context) (string, time.Time, string, error) {
	assertion, err := BuildJWTAssertion(JWTConfig{
		ClientID:   ts.cfg.ClientID,
		TokenURL:   ts.cfg.TokenURL,
		PrivateKey: ts.jwtKey,
		Expiry:     ts.cfg.JWTExpiry,
	})
	if err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: err.Error(), Err: err}
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", assertion)
	form.Set("client_authentication_method", "private_key_jwt")
	if ts.cfg.Scope != "" {
		form.Set("scope", ts.cfg.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.cfg.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: err.Error(), Err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := ts.client.Do(req)
	if err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: err.Error(), Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		return "", time.Time{}, "", &Error{
			Op:         "OAuth.Token",
			Message:    fmt.Sprintf("token endpoint returned %d: %s", resp.StatusCode, redact(strings.TrimSpace(string(body)))),
			HTTPStatus: resp.StatusCode,
			Err:        statusSentinel[resp.StatusCode],
		}
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: "decode token response: " + err.Error(), Err: err}
	}
	if tr.AccessToken == "" {
		return "", time.Time{}, "", &Error{Op: "OAuth.Token", Message: "token response missing access_token", Err: ErrNotAuthenticated}
	}
	expiry := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn <= 0 {
		expiry = time.Now().Add(time.Minute)
	}
	return tr.AccessToken, expiry, tr.RefreshToken, nil
}
