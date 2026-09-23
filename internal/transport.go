// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const defaultMaxResponseBytes = 32 << 20

// TransportConfig assembles the client's HTTP middleware chain. Zero fields are
// skipped. Middlewares are applied outermost-first in the order below.
type TransportConfig struct {
	RequestID        func() string
	UserAgent        string
	AuthHeader       string
	Token            func() (string, bool)
	Logger           *slog.Logger
	Telemetry        Telemetry
	Metrics          Metrics
	Breaker          *Breaker
	Retry            RetryPolicy
	Limiter          RateLimiter
	Timeout          time.Duration
	MaxResponseBytes int64
	UserMiddleware   []Middleware
}

// NewClientTransport builds the RoundTripper chain used by the SDK. Order
// (outer → inner): requestID → userAgent → auth → … → errorDecode → base.
func NewClientTransport(base http.RoundTripper, cfg TransportConfig) http.RoundTripper {
	if base == nil {
		base = newDefaultTransport()
	}
	var ms []func(http.RoundTripper) http.RoundTripper
	if cfg.RequestID != nil {
		ms = append(ms, RequestID(cfg.RequestID))
	}
	if cfg.UserAgent != "" {
		ms = append(ms, UserAgent(cfg.UserAgent))
	}
	if cfg.AuthHeader != "" && cfg.Token != nil {
		ms = append(ms, Auth(cfg.AuthHeader, cfg.Token))
	}
	if cfg.Logger != nil || cfg.Telemetry != nil {
		ms = append(ms, Logging(cfg.Logger, cfg.Telemetry))
	}
	if cfg.Metrics != nil {
		ms = append(ms, Instrument(cfg.Metrics))
	}
	if cfg.Breaker != nil {
		ms = append(ms, CircuitBreaker(cfg.Breaker))
	}
	if cfg.Retry.enabled() {
		ms = append(ms, Retry(cfg.Retry))
	}
	if cfg.Limiter != nil {
		ms = append(ms, RateLimit(cfg.Limiter))
	}
	if cfg.Timeout > 0 {
		ms = append(ms, Timeout(cfg.Timeout))
	}
	if cfg.MaxResponseBytes > 0 {
		ms = append(ms, MaxBytes(cfg.MaxResponseBytes))
	}
	ms = append(ms, ErrorDecode())
	ms = append(ms, cfg.UserMiddleware...)
	return Chain(base, ms...)
}

// RoundTripFunc lets a plain function satisfy http.RoundTripper.
type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Chain builds a middleware stack on top of base. Middlewares are applied
// left-to-right: ms[0] is the outermost (called first on the way in, last on
// the way out).
func Chain(base http.RoundTripper, ms ...func(http.RoundTripper) http.RoundTripper) http.RoundTripper {
	for i := len(ms) - 1; i >= 0; i-- {
		base = ms[i](base)
	}
	return base
}

// newDefaultTransport returns an *http.Transport tuned for IBKR's long-lived
// HTTPS connections with HTTP/2 multiplexing and connection pooling.
func newDefaultTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
}

// RequestID returns a middleware that injects X-request-id from f.
func RequestID(f func() string) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("X-request-id", f())
			return base.RoundTrip(req)
		})
	}
}

// UserAgent returns a middleware that sets User-Agent.
func UserAgent(ua string) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("User-Agent", ua)
			return base.RoundTrip(req)
		})
	}
}

// Auth returns a middleware that adds token as header when token() returns ok.
func Auth(header string, token func() (string, bool)) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			if tok, ok := token(); ok {
				req.Header.Set(header, tok)
			}
			return base.RoundTrip(req)
		})
	}
}

// Timeout returns a middleware that applies d as a per-request deadline when the
// caller's context has none.
func Timeout(d time.Duration) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if d <= 0 {
				return base.RoundTrip(req)
			}
			if _, ok := req.Context().Deadline(); ok {
				return base.RoundTrip(req)
			}
			ctx, cancel := context.WithTimeout(req.Context(), d)
			defer cancel()
			resp, err := base.RoundTrip(req.WithContext(ctx))
			if err != nil {
				return resp, err
			}
			if resp == nil {
				return resp, err
			}
			resp.Body = &cancelOnCloseBody{body: resp.Body, cancel: cancel}
			return resp, nil
		})
	}
}

// cancelOnCloseBody wraps resp.Body so that the timeout cancel fires only after
// the body is fully consumed and closed. This prevents context.Canceled errors
// when a large or slow success body is read after the transport has returned.
type cancelOnCloseBody struct {
	body   io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnCloseBody) Read(b []byte) (int, error) { return c.body.Read(b) }

func (c *cancelOnCloseBody) Close() error {
	err := c.body.Close()
	c.cancel()
	return err
}

// maxBytesReader wraps resp.Body with an io.LimitedReader and enforces a byte
// limit on reads. Close delegates to the original body.
type maxBytesReader struct {
	orig io.ReadCloser
	lim  *io.LimitedReader
}

func (m *maxBytesReader) Read(b []byte) (int, error) { return m.lim.Read(b) }

func (m *maxBytesReader) Close() error { return m.orig.Close() }

// MaxBytes returns a middleware that limits the response body size to n bytes,
// preventing unbounded memory growth on large responses. Truncation is detected
// by the caller via a short read and surfaced as a typed error.
func MaxBytes(n int64) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := base.RoundTrip(req)
			if err != nil || resp == nil {
				return resp, err
			}
			orig := resp.Body
			resp.Body = &maxBytesReader{orig: orig, lim: &io.LimitedReader{R: orig, N: n}}
			return resp, nil
		})
	}
}

// ErrorDecode is the innermost middleware: it decodes 4xx/5xx bodies into the
// X-ibkr-* headers consumed by ResponseError. 1xx (WebSocket upgrade) and 3xx
// pass through untouched.
func ErrorDecode() func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := base.RoundTrip(req)
			if err != nil {
				return resp, err
			}
			if resp.StatusCode < 400 {
				return resp, nil
			}
			reqID := resp.Header.Get("X-request-id")
			if reqID == "" {
				reqID = req.Header.Get("X-request-id")
			}
			return carryError(resp, reqID), nil
		})
	}
}

// isErrorStatus reports whether a status code should be decoded as an error.
func isErrorStatus(code int) bool { return code >= 400 }

var headerRedact = regexp.MustCompile(`(?i)(Authorization|Cookie|Set-Cookie)\s*:\s*[^\r\n,;]*`)

var tokenPatterns = []string{
	`(?i)bearer\s+[A-Za-z0-9_\-\.~+/]+=*`,
	`(?i)access_token\s*=\s*[A-Za-z0-9_\-\.~+/]+=*`,
	`(?i)refresh_token\s*=\s*[A-Za-z0-9_\-\.~+/]+=*`,
	`(?i)client_secret\s*=\s*[A-Za-z0-9_\-\.~+/]+=*`,
	`(?i)id_token\s*=\s*[A-Za-z0-9_\-\.~+/]+=*`,
	`sess=[A-Za-z0-9_\-\.~+/]+=*`,
	`(?i)x-csrf-token\s*[:=]\s*[A-Za-z0-9_\-\.~+/]+=*`,
}

var secretRedact = regexp.MustCompile(func() string {
	var all []string
	for _, p := range tokenPatterns {
		all = append(all, p)
	}
	return `(?i)(` + strings.Join(all, `|`) + `)`
}())

func redact(s string) string {
	s = headerRedact.ReplaceAllString(s, "$1: <redacted>")
	s = secretRedact.ReplaceAllString(s, "<redacted>")
	return s
}

// carryError reads the response body, parses the IBKR error envelope, and returns
// a response with the error details stored in headers. The body is replaced with
// a fresh reader so the caller can still read it.
func carryError(resp *http.Response, reqID string) *http.Response {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return errorResponse(resp, reqID, http.StatusText(resp.StatusCode), "", statusSentinel[resp.StatusCode])
	}
	resp.Body.Close()

	var env ibkrErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return errorResponse(resp, reqID, http.StatusText(resp.StatusCode), "", statusSentinel[resp.StatusCode])
	}

	msg := strings.TrimSpace(env.Message)
	if msg == "" {
		msg = strings.TrimSpace(env.Error)
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	code := strings.TrimSpace(env.Code)
	if code == "" {
		code = strings.TrimSpace(env.Error)
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))

	return errorResponse(resp, reqID, msg, code, statusSentinel[resp.StatusCode])
}

func errorResponse(resp *http.Response, reqID, msg, code string, serr error) *http.Response {
	resp.Header.Set("X-ibkr-error", msg)
	resp.Header.Set("X-ibkr-error-code", code)
	resp.Header.Set("X-ibkr-request-id", reqID)
	if serr != nil {
		resp.Header.Set("X-ibkr-sentinel", serr.Error())
	}
	return resp
}

// ResponseError extracts an *Error from a response that was flagged as an error.
// It returns nil if the response is not an error response (<400).
// The caller must not have read the body yet.
func ResponseError(resp *http.Response) *Error {
	if !isErrorStatus(resp.StatusCode) {
		return nil
	}
	errMsg := resp.Header.Get("X-ibkr-error")
	code := resp.Header.Get("X-ibkr-error-code")
	reqID := resp.Header.Get("X-ibkr-request-id")
	sentinelName := resp.Header.Get("X-ibkr-sentinel")

	var serr error
	if sentinelName != "" {
		serr = sentinelByName(sentinelName)
	}
	if serr == nil {
		serr = statusSentinel[resp.StatusCode]
	}
	if errMsg == "" {
		errMsg = http.StatusText(resp.StatusCode)
	}
	return &Error{
		Op:         "unknown",
		Code:       code,
		Message:    redact(errMsg),
		HTTPStatus: resp.StatusCode,
		RequestID:  reqID,
		Err:        serr,
	}
}

func sentinelByName(name string) error {
	switch name {
	case ErrNotAuthenticated.Error():
		return ErrNotAuthenticated
	case ErrSessionExpired.Error():
		return ErrSessionExpired
	case ErrRateLimited.Error():
		return ErrRateLimited
	case ErrNotFound.Error():
		return ErrNotFound
	case ErrInvalidRequest.Error():
		return ErrInvalidRequest
	case ErrOrderRejected.Error():
		return ErrOrderRejected
	case ErrClosed.Error():
		return ErrClosed
	case ErrStreamingLimit.Error():
		return ErrStreamingLimit
	default:
		return nil
	}
}
