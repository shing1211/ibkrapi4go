// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// RequestInfo describes an outbound request for telemetry.
type RequestInfo struct {
	// Method is the HTTP method.
	Method string
	// Path is the normalized request path (dynamic id segments become {}).
	Path string
	// RequestID is the correlation id, if set.
	RequestID string
}

// ResponseInfo describes the outcome of a request for telemetry.
type ResponseInfo struct {
	// StatusCode is the HTTP status (0 on transport error).
	StatusCode int
	// Duration is the round-trip time.
	Duration time.Duration
	// Err is the transport error, if any.
	Err error
}

// Telemetry receives request lifecycle callbacks. Implementations must be safe
// for concurrent use and must not block. It is compatible with OpenTelemetry
// tracers but carries no dependency on them; callers may bridge to OTel spans.
type Telemetry interface {
	// OnRequestStart is called before the request is sent. The returned context
	// is passed to OnRequestEnd.
	OnRequestStart(ctx context.Context, info RequestInfo) context.Context
	// OnRequestEnd is called after the response is received or the request fails.
	OnRequestEnd(ctx context.Context, info RequestInfo, resp ResponseInfo)
}

// Logging returns a middleware that logs each request (method, normalized path,
// status, duration, request id) and invokes telemetry hooks. Bodies and headers
// are never logged; error text is redacted.
func Logging(logger *slog.Logger, hooks Telemetry) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		if logger == nil && hooks == nil {
			return base
		}
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			info := RequestInfo{
				Method:    req.Method,
				Path:      normalizePath(req.URL.Path),
				RequestID: req.Header.Get("X-request-id"),
			}
			ctx := req.Context()
			if hooks != nil {
				ctx = hooks.OnRequestStart(ctx, info)
			}
			start := time.Now()
			resp, err := base.RoundTrip(req)
			duration := time.Since(start)

			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			if hooks != nil {
				hooks.OnRequestEnd(ctx, info, ResponseInfo{StatusCode: status, Duration: duration, Err: err})
			}
			if logger != nil {
				logRequest(logger, info, status, duration, err)
			}
			return resp, err
		})
	}
}

func logRequest(logger *slog.Logger, info RequestInfo, status int, duration time.Duration, err error) {
	attrs := []any{
		"method", info.Method,
		"path", info.Path,
		"status", status,
		"duration", duration.String(),
		"request_id", info.RequestID,
	}
	switch {
	case err != nil:
		logger.Warn("http request failed", append(attrs, "err", redact(err.Error()))...)
	case status >= 400:
		logger.Warn("http request error", attrs...)
	default:
		logger.Debug("http request", attrs...)
	}
}

// LogError logs a decoded *Error with operation context. The message is redacted.
func LogError(logger *slog.Logger, e *Error) {
	if logger == nil || e == nil {
		return
	}
	logger.Warn("ibkr error",
		"op", e.Op,
		"code", e.Code,
		"status", e.HTTPStatus,
		"request_id", e.RequestID,
		"message", redact(e.Message),
	)
}
