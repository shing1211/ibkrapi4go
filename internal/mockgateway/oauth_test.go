// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The OAuth2 token endpoint is the second gatekeeper: it decides which tokens the
// IB REST (bearer) surface will accept. If the mock is more permissive than the
// real gateway, a test can pass against a request the gateway would reject, and if
// it is stricter, valid flows break for reasons that do not exist upstream.

func form(pairs ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v.Set(pairs[i], pairs[i+1])
	}
	return v
}

// b64 encodes s the way a JWT segment is encoded.
func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func jwt(header, payload string) string { return b64(header) + "." + b64(payload) + "." + b64("sig") }

const (
	okHeader  = `{"alg":"RS256"}`
	okPayload = `{"iss":"client","sub":"client"}`
)

func TestValidateTokenRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		form     url.Values
		wantCode string
	}{
		// client_credentials
		{"client_credentials", form("grant_type", "client_credentials",
			"client_id", "id", "client_secret", "s"), ""},
		{"client_credentials missing secret", form("grant_type", "client_credentials",
			"client_id", "id"), "invalid_client"},
		{"client_credentials missing id", form("grant_type", "client_credentials",
			"client_secret", "s"), "invalid_client"},

		// refresh_token
		{"refresh_token", form("grant_type", "refresh_token", "refresh_token", "r"), ""},
		{"refresh_token without token", form("grant_type", "refresh_token"), "invalid_request"},

		// grant type itself
		{"missing grant_type", form(), "invalid_request"},
		{"unknown grant_type", form("grant_type", "password"), "unsupported_grant_type"},

		// private_key_jwt / client assertions
		{"assertion with client_credentials", form("grant_type", "client_credentials",
			"client_assertion", jwt(okHeader, okPayload),
			"client_assertion_type", jwtBearerAssertionType), ""},
		{"explicit private_key_jwt with assertion", form("grant_type", "private_key_jwt",
			"client_assertion", jwt(okHeader, okPayload),
			"client_assertion_type", jwtBearerAssertionType), ""},
		{"private_key_jwt without assertion", form("grant_type", "private_key_jwt"),
			"invalid_request"},
		{"wrong assertion type", form("grant_type", "client_credentials",
			"client_assertion", jwt(okHeader, okPayload),
			"client_assertion_type", "urn:other"), "invalid_request"},
		{"assertion with refresh_token grant", form("grant_type", "refresh_token",
			"refresh_token", "r",
			"client_assertion", jwt(okHeader, okPayload),
			"client_assertion_type", jwtBearerAssertionType), "invalid_request"},
		{"assertion with unsupported grant", form("grant_type", "password",
			"client_assertion", jwt(okHeader, okPayload),
			"client_assertion_type", jwtBearerAssertionType), "unsupported_grant_type"},
		{"malformed assertion", form("grant_type", "client_credentials",
			"client_assertion", "not-a-jwt",
			"client_assertion_type", jwtBearerAssertionType), "invalid_grant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, desc := validateTokenRequest(tc.form)
			if code != tc.wantCode {
				t.Errorf("code = %q (desc %q); want %q", code, desc, tc.wantCode)
			}
			// An accepted request must carry no error description, and a rejected
			// one must explain itself: the description is part of the API surface.
			if tc.wantCode == "" && desc != "" {
				t.Errorf("accepted request returned desc %q; want empty", desc)
			}
			if tc.wantCode != "" && desc == "" {
				t.Error("rejected request returned an empty description")
			}
		})
	}
}

func TestValidJWTAssertionShape(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
		want  bool
	}{
		{"well-formed", jwt(okHeader, okPayload), true},
		{"only iss", jwt(okHeader, `{"iss":"client"}`), true},
		{"only sub", jwt(okHeader, `{"sub":"client"}`), true},
		{"extra claims are fine", jwt(okHeader, `{"iss":"a","aud":"b","exp":1}`), true},

		{"not enough segments", "a.b", false},
		{"too many segments", "a.b.c.d", false},
		{"empty segment", "a..c", false},
		{"empty token", "", false},
		{"header not base64", "!!!." + b64(okPayload) + ".sig", false},
		{"header not json", b64("nope") + "." + b64(okPayload) + ".sig", false},
		{"empty alg", jwt(`{"alg":""}`, okPayload), false},
		{"payload not base64", b64(okHeader) + ".!!!.sig", false},
		{"payload not json", b64(okHeader) + "." + b64("nope") + ".sig", false},
		{"empty claims", jwt(okHeader, `{}`), false},
		{"neither iss nor sub", jwt(okHeader, `{"aud":"b"}`), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validJWTAssertionShape(tc.token); got != tc.want {
				t.Errorf("validJWTAssertionShape() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestOAuthStore_IssueAndValidate(t *testing.T) {
	o := newOAuthStore()

	if o.valid("") {
		t.Error("an empty token was accepted")
	}
	if o.valid("mock-access-1") {
		t.Error("a token was accepted before any was issued")
	}

	access, refresh, expiresIn := o.issue()
	if !strings.HasPrefix(access, "mock-access-") {
		t.Errorf("access token = %q; want a mock-access- prefix", access)
	}
	if !strings.HasPrefix(refresh, "mock-refresh-") {
		t.Errorf("refresh token = %q; want a mock-refresh- prefix", refresh)
	}
	if expiresIn != oauthTokenTTLSeconds {
		t.Errorf("expires_in = %d; want %d", expiresIn, oauthTokenTTLSeconds)
	}
	if !o.valid(access) {
		t.Error("a freshly issued token was rejected")
	}
	// The refresh token is not an access token and must not authenticate.
	if o.valid(refresh) {
		t.Error("the refresh token validated as an access token")
	}

	// Tokens must be distinct, or one could be replayed after the other expires.
	access2, _, _ := o.issue()
	if access2 == access {
		t.Errorf("two issues returned the same access token %q", access)
	}
}

func TestOAuthStore_ExpiredTokenIsRejectedAndDropped(t *testing.T) {
	o := newOAuthStore()
	access, _, _ := o.issue()

	// Backdate the recorded expiry rather than sleeping for the TTL.
	o.mu.Lock()
	o.tokens[access] = time.Now().Add(-time.Minute)
	o.mu.Unlock()

	if o.valid(access) {
		t.Error("an expired token was accepted")
	}
	o.mu.Lock()
	_, still := o.tokens[access]
	o.mu.Unlock()
	if still {
		t.Error("an expired token was left in the store; it should be evicted on rejection")
	}
}

func TestOAuthStore_Authenticate(t *testing.T) {
	o := newOAuthStore()
	access, _, _ := o.issue()

	bearer := func(v string) *Request { return reqWith(map[string]string{"Authorization": v}) }

	for _, tc := range []struct {
		name string
		req  *Request
		want bool
	}{
		{"bearer with valid token", bearer("Bearer " + access), true},
		{"lowercase scheme", bearer("bearer " + access), true},
		{"mixed case scheme", bearer("BeArEr " + access), true},
		{"extra whitespace", bearer("Bearer   " + access), true},
		{"unissued token", bearer("Bearer mock-access-999"), false},
		{"refresh token as bearer", bearer("Bearer mock-refresh-1"), false},
		{"missing scheme", bearer(access), false},
		{"wrong scheme", bearer("Basic " + access), false},
		{"scheme only", bearer("Bearer"), false},
		{"scheme with empty token", bearer("Bearer "), false},
		{"no header", reqWith(nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := o.authenticate(tc.req); got != tc.want {
				t.Errorf("authenticate() = %v; want %v", got, tc.want)
			}
		})
	}
}

// TestServeToken_ClientCredentialsIssuesAUsableToken is the end-to-end property:
// a well-formed request yields a token, and that token authenticates a bearer
// route. Testing issue() and authenticate() separately would not catch a mismatch
// in what the endpoint puts in the JSON body versus what the store expects.
func TestServeToken_ClientCredentialsIssuesAUsableToken(t *testing.T) {
	s := New()
	rec := httptest.NewRecorder()

	body := "grant_type=client_credentials&client_id=id&client_secret=s"
	s.serveToken(rec, reqWithForm(body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var got struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if got.AccessToken == "" {
		t.Fatal("token response carried no access_token")
	}
	if got.TokenType == "" {
		t.Error("token response carried no token_type")
	}
	if got.ExpiresIn != oauthTokenTTLSeconds {
		t.Errorf("expires_in = %d; want %d", got.ExpiresIn, oauthTokenTTLSeconds)
	}
	if !s.oauth.valid(got.AccessToken) {
		t.Error("the issued access token is not valid in the store")
	}
}

func TestServeToken_ErrorsUseRFC6749StatusAndCode(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		// invalid_client maps to 401, everything else to 400.
		{"missing credentials", "grant_type=client_credentials&client_id=id",
			http.StatusUnauthorized, "invalid_client"},
		{"unsupported grant", "grant_type=password",
			http.StatusBadRequest, "unsupported_grant_type"},
		{"missing grant type", "client_id=id",
			http.StatusBadRequest, "invalid_request"},
		{"malformed form", "%zz", http.StatusBadRequest, "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			rec := httptest.NewRecorder()
			s.serveToken(rec, reqWithForm(tc.body))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body %s)", rec.Code, tc.wantStatus, rec.Body)
			}
			var got struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if got.Error != tc.wantCode {
				t.Errorf("error = %q; want %q", got.Error, tc.wantCode)
			}
		})
	}
}

// TestServer_BearerRouteRequiresIssuedToken checks the mock's IB REST surface
// end to end: an unissued bearer token is refused with invalid_token, and one the
// token endpoint actually issued is served.
func TestServer_BearerRouteRequiresIssuedToken(t *testing.T) {
	const bearerPath = "/gw/api/v1/accounts" // OpListAccounts

	t.Run("unissued token is refused", func(t *testing.T) {
		s := New()
		req := httptest.NewRequest(http.MethodGet, bearerPath, nil)
		req.Header.Set("Authorization", "Bearer mock-access-999")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusUnauthorized)
		}
		if !strings.Contains(rec.Body.String(), "invalid_token") {
			t.Errorf("body = %s; want it to report invalid_token", rec.Body)
		}
	})

	t.Run("issued token is served", func(t *testing.T) {
		s := New()
		tokenRec := httptest.NewRecorder()
		s.serveToken(tokenRec, reqWithForm("grant_type=client_credentials&client_id=id&client_secret=s"))

		var got struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(tokenRec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode token response: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, bearerPath, nil)
		req.Header.Set("Authorization", "Bearer "+got.AccessToken)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
		}
	})
}

// reqWithForm builds a POST-shaped Request with a form body, which is what
// serveToken parses.
func reqWithForm(body string) *Request {
	r := reqWith(map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	r.Method = "POST"
	r.Body = []byte(body)
	return r
}
