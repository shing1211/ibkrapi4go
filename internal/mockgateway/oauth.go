// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuth2 token endpoint behavior. The mock validates the *shape* of token
// requests (grant type, required fields, assertion format) but never a
// signature; it then issues opaque tokens that it remembers for bearer
// validation on the IB REST surface.
const (
	// oauthTokenType is the token_type returned by the token endpoint.
	oauthTokenType = "Bearer"
	// oauthTokenTTLSeconds is the expires_in value returned by the endpoint.
	oauthTokenTTLSeconds = 3600
	// jwtBearerAssertionType is the client_assertion_type for private_key_jwt.
	jwtBearerAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	// bearerPrefix is the Authorization scheme used on the REST surface.
	bearerPrefix = "Bearer "
)

// oauthStore issues and validates the opaque access tokens handed out by the
// token endpoint. It is safe for concurrent use.
type oauthStore struct {
	mu     sync.Mutex
	seq    int64
	tokens map[string]time.Time
}

// newOAuthStore returns an empty token store.
func newOAuthStore() *oauthStore {
	return &oauthStore{tokens: make(map[string]time.Time)}
}

// issue mints a fresh access/refresh token pair and remembers the access token
// until it expires.
func (o *oauthStore) issue() (access, refresh string, expiresIn int64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seq++
	access = fmt.Sprintf("mock-access-%d", o.seq)
	refresh = fmt.Sprintf("mock-refresh-%d", o.seq)
	expiresIn = oauthTokenTTLSeconds
	o.tokens[access] = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return access, refresh, expiresIn
}

// valid reports whether token was issued by this store and has not expired.
func (o *oauthStore) valid(token string) bool {
	if token == "" {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	expiry, ok := o.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(o.tokens, token)
		return false
	}
	return true
}

// authenticate reports whether req carries a valid bearer access token.
func (o *oauthStore) authenticate(req *Request) bool {
	auth := req.Headers.Get("Authorization")
	if len(auth) <= len(bearerPrefix) || !strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
		return false
	}
	return o.valid(strings.TrimSpace(auth[len(bearerPrefix):]))
}

// serveToken handles POST /oauth2/api/v1/token for the client_credentials,
// refresh_token, and private_key_jwt flows.
func (s *Server) serveToken(w http.ResponseWriter, req *Request) {
	form, err := url.ParseQuery(string(req.Body))
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "malformed form body")
		return
	}
	if code, desc := validateTokenRequest(form); code != "" {
		status := http.StatusBadRequest
		if code == "invalid_client" {
			status = http.StatusUnauthorized
		}
		writeOAuthError(w, status, code, desc)
		return
	}

	access, refresh, expiresIn := s.oauth.issue()
	body, err := json.Marshal(struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}{access, oauthTokenType, expiresIn, refresh})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, string(body))
}

// validateTokenRequest checks the shape of a token request. It returns an
// empty code when the request is acceptable; otherwise it returns an RFC 6749
// error code and a human-readable description.
func validateTokenRequest(form url.Values) (code, desc string) {
	grant := form.Get("grant_type")
	assertion := form.Get("client_assertion")
	assertionType := form.Get("client_assertion_type")

	// private_key_jwt: detected by the presence of a client assertion. IBKR
	// sends it with grant_type=client_credentials (the assertion itself is the
	// client authentication), but an explicit private_key_jwt grant is also
	// accepted.
	if grant == "private_key_jwt" && assertion == "" {
		return "invalid_request", "client_assertion is required for the private_key_jwt grant"
	}
	if assertion != "" || assertionType != "" {
		if assertionType != jwtBearerAssertionType {
			return "invalid_request", "client_assertion_type must be " + jwtBearerAssertionType
		}
		if grant == "refresh_token" {
			return "invalid_request", "client_assertion is not valid for the refresh_token grant"
		}
		if grant != "" && grant != "client_credentials" && grant != "private_key_jwt" {
			return "unsupported_grant_type", "only client_credentials and private_key_jwt are supported with client assertions"
		}
		if !validJWTAssertionShape(assertion) {
			return "invalid_grant", "client_assertion is not a well-formed JWT"
		}
		return "", ""
	}

	switch grant {
	case "client_credentials":
		if form.Get("client_id") == "" || form.Get("client_secret") == "" {
			return "invalid_client", "client_id and client_secret are required"
		}
		return "", ""
	case "refresh_token":
		if form.Get("refresh_token") == "" {
			return "invalid_request", "refresh_token is required"
		}
		return "", ""
	case "":
		return "invalid_request", "grant_type is required"
	default:
		return "unsupported_grant_type", "unsupported grant_type: " + grant
	}
}

// validJWTAssertionShape reports whether token is a syntactically plausible
// JWT assertion: three non-empty segments, a non-empty `alg`, and at least one
// of the `iss`/`sub` claims. Signatures and the algorithm are not verified.
func validJWTAssertionShape(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(header, &hdr); err != nil || hdr.Alg == "" {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil || len(claims) == 0 {
		return false
	}
	_, hasSub := claims["sub"]
	_, hasIss := claims["iss"]
	return hasSub || hasIss
}

// writeOAuthError writes an RFC 6749 error response.
func writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	body, err := json.Marshal(struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description,omitempty"`
	}{code, desc})
	if err != nil {
		writeJSON(w, status, `{"error":"server_error"}`)
		return
	}
	writeJSON(w, status, string(body))
}
