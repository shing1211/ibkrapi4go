// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// NopLogger returns a logger that discards all output. It is used as the
// default by every component so callers never have to nil-check *slog.Logger.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// SpanContext carries trace context across the API boundary.
type SpanContext struct {
	TraceID string
	SpanID  string
}

// RequestInfo describes an outbound request for telemetry.
type RequestInfo struct {
	// Method is the HTTP method.
	Method string
	// Path is the normalized request path (dynamic id segments become {}).
	Path string
	// RequestID is the correlation id, if set.
	RequestID string
	// SpanContext carries the active trace context from OnRequestStart.
	SpanContext SpanContext
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

// WSConnInfo describes a WebSocket connection event.
type WSConnInfo struct {
	// Event is "connect", "disconnect", "reconnect"
	Event string
	// URL is the gateway WS URL (host:port only, no tokens)
	URL string
	// Subscriptions is the number of active subscriptions at event time.
	Subscriptions int
	// Err is the error on disconnect, nil otherwise.
	Err error
}

// WSSubInfo describes a subscription change.
type WSSubInfo struct {
	// Event is "subscribe" or "unsubscribe"
	Event string
	// ConIDs are the contract IDs affected.
	ConIDs []int
	// Fields are the field codes subscribed (subscribe only).
	Fields []string
}

// OrderEventInfo describes an order lifecycle event.
type OrderEventInfo struct {
	// Event is "submit", "modify", "cancel", "fill", "partial_fill", "reject", "timeout"
	Event string
	// AccountID is the account.
	AccountID string
	// OrderID is the server-assigned order ID (filled on submit confirmation).
	OrderID string
	// ClientOrderID is the caller-supplied order ID.
	ClientOrderID string
	// ConID is the contract ID.
	ConID int
	// Err is non-nil for reject/timeout events.
	Err error
}

// Telemetry receives lifecycle callbacks. The interface is compatible with
// OpenTelemetry tracers; implementations must be safe for concurrent use
// and must not block.
type Telemetry interface {
	// OnRequestStart is called before the request is sent. The returned context
	// carries the active span and is passed to OnRequestEnd.
	OnRequestStart(ctx context.Context, info RequestInfo) context.Context
	// OnRequestEnd is called after the response is received or the request fails.
	OnRequestEnd(ctx context.Context, info RequestInfo, resp ResponseInfo)

	// OnWSConnect is called when a WebSocket connection is established.
	OnWSConnect(ctx context.Context, info WSConnInfo)
	// OnWSDisconnect is called when a WebSocket connection closes.
	OnWSDisconnect(ctx context.Context, info WSConnInfo)
	// OnWSSubscribe is called when a subscription changes.
	OnWSSubscribe(ctx context.Context, info WSSubInfo)
	// OnWSUnsubscribe is called when an unsubscription occurs.
	OnWSUnsubscribe(ctx context.Context, info WSSubInfo)

	// OnOrderSubmit is called when an order is submitted.
	OnOrderSubmit(ctx context.Context, info OrderEventInfo)
	// OnOrderUpdate is called when an order status changes (fill, partial, reject, cancel).
	OnOrderUpdate(ctx context.Context, info OrderEventInfo)
}

type nopTelemetry struct{}

func (nopTelemetry) OnRequestStart(ctx context.Context, info RequestInfo) context.Context { return ctx }
func (nopTelemetry) OnRequestEnd(context.Context, RequestInfo, ResponseInfo)              {}
func (nopTelemetry) OnWSConnect(context.Context, WSConnInfo)                              {}
func (nopTelemetry) OnWSDisconnect(context.Context, WSConnInfo)                           {}
func (nopTelemetry) OnWSSubscribe(context.Context, WSSubInfo)                             {}
func (nopTelemetry) OnWSUnsubscribe(context.Context, WSSubInfo)                           {}
func (nopTelemetry) OnOrderSubmit(context.Context, OrderEventInfo)                        {}
func (nopTelemetry) OnOrderUpdate(context.Context, OrderEventInfo)                        {}

func NopTelemetry() Telemetry { return nopTelemetry{} }

// Logging returns a middleware that logs each request (method, normalized path,
// status, duration, request id) and invokes telemetry hooks. Bodies and headers
// are never logged; error text is redacted.
func Logging(logger *slog.Logger, hooks Telemetry) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return func(base http.RoundTripper) http.RoundTripper {
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
			logRequest(logger, info, status, duration, err)
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
		logger.Warn("ibkr.http failed", append(attrs, "err", redact(err.Error()))...)
	case status >= 400:
		logger.Warn("ibkr.http error", attrs...)
	default:
		logger.Debug("ibkr.http request", attrs...)
	}
}

// LogError logs a decoded *Error with operation context. The message is redacted.
func LogError(logger *slog.Logger, e *Error) {
	if e == nil {
		return
	}
	logger.Warn("ibkr.error",
		"op", e.Op,
		"code", e.Code,
		"status", e.HTTPStatus,
		"request_id", e.RequestID,
		"message", redact(e.Message),
	)
}
