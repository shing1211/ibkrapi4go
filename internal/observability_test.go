// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type recordingTelemetry struct {
	starts atomic.Int64
	ends   atomic.Int64
	req    RequestInfo
	resp   ResponseInfo
}

func (t *recordingTelemetry) OnRequestStart(ctx context.Context, info RequestInfo) context.Context {
	t.starts.Add(1)
	t.req = info
	return ctx
}

func (t *recordingTelemetry) OnRequestEnd(ctx context.Context, info RequestInfo, resp ResponseInfo) {
	t.ends.Add(1)
	t.resp = resp
}

func (t *recordingTelemetry) OnWSConnect(context.Context, WSConnInfo)                         {}
func (t *recordingTelemetry) OnWSDisconnect(context.Context, WSConnInfo)                     {}
func (t *recordingTelemetry) OnWSSubscribe(context.Context, WSSubInfo)                      {}
func (t *recordingTelemetry) OnWSUnsubscribe(context.Context, WSSubInfo)                   {}
func (t *recordingTelemetry) OnOrderSubmit(context.Context, OrderEventInfo)                  {}
func (t *recordingTelemetry) OnOrderUpdate(context.Context, OrderEventInfo)                  {}

func newBufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func okBase(status int) http.RoundTripper {
	return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
}

func TestLogging_LogsAndInvokesHooks(t *testing.T) {
	var buf bytes.Buffer
	tel := &recordingTelemetry{}
	rt := NewClientTransport(okBase(200), TransportConfig{
		RequestID: func() string { return "req-abc" },
		Logger:    newBufferLogger(&buf),
		Telemetry: tel,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/v1/api/portfolio/U1234567/summary", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if tel.starts.Load() != 1 || tel.ends.Load() != 1 {
		t.Errorf("hooks starts=%d ends=%d; want 1/1", tel.starts.Load(), tel.ends.Load())
	}
	if tel.req.Path != "/v1/api/portfolio/{}/summary" {
		t.Errorf("telemetry path = %q; want normalized", tel.req.Path)
	}
	if tel.resp.StatusCode != 200 {
		t.Errorf("telemetry status = %d; want 200", tel.resp.StatusCode)
	}
	out := buf.String()
	if !strings.Contains(out, "http request") || !strings.Contains(out, "/v1/api/portfolio/{}/summary") {
		t.Errorf("log output = %q; want normalized request log", out)
	}
}

func TestLogging_RedactsSecrets(t *testing.T) {
	var buf bytes.Buffer
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("Authorization: Bearer supersecret-token")
	})
	rt := NewClientTransport(base, TransportConfig{Logger: newBufferLogger(&buf)})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)
	_, _ = rt.RoundTrip(req)

	out := buf.String()
	if strings.Contains(out, "supersecret-token") {
		t.Errorf("secret leaked in logs: %q", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Errorf("log output = %q; want redaction marker", out)
	}
}

func TestLogError_Redacts(t *testing.T) {
	var buf bytes.Buffer
	LogError(newBufferLogger(&buf), &Error{
		Op:         "Account.List",
		Code:       "500",
		HTTPStatus: 500,
		RequestID:  "req-1",
		Message:    "Cookie: session=abc123",
	})
	out := buf.String()
	if strings.Contains(out, "abc123") {
		t.Errorf("secret leaked in error log: %q", out)
	}
	if !strings.Contains(out, "Account.List") || !strings.Contains(out, "<redacted>") {
		t.Errorf("log output = %q; want op and redaction", out)
	}
}
