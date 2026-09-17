// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// dialStream starts a Server, dials its WebSocket endpoint, and returns the
// connection. A nil script uses the default behavior.
func dialStream(t *testing.T, script *StreamScript) (*Server, *websocket.Conn) {
	t.Helper()
	srv := New(WithStreamScript(script))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/api/ws"
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	t.Cleanup(func() { _ = c.Close(websocket.StatusNormalClosure, "done") })
	return srv, c
}

// writeStreamFrame writes a text frame and fails on error.
func writeStreamFrame(t *testing.T, c *websocket.Conn, msg string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

// readStreamFrame reads one JSON frame.
func readStreamFrame(t *testing.T, c *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode frame %q: %v", data, err)
	}
	return m
}

// waitUnsubscribed pushes ticks until the mock reports no subscriber, draining
// any in-flight frames so later reads see only subsequent scripted frames.
func waitUnsubscribed(t *testing.T, srv *Server, c *websocket.Conn, conid int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if srv.PushTick(Tick{ConID: conid, Field: "31", Value: "1"}) == 0 {
			return
		}
		_ = readStreamFrame(t, c)
		select {
		case <-deadline:
			t.Fatal("unsubscribe not applied")
		default:
		}
	}
}

func TestStream_PushAndUnsubscribe(t *testing.T) {
	srv, c := dialStream(t, nil)

	writeStreamFrame(t, c, `{"id":1,"method":"subscribe","params":{"conids":[265598],"fields":["31"]}}`)
	frame := readStreamFrame(t, c)
	if frame["conid"] != float64(265598) || frame["31"] != "150.25" {
		t.Fatalf("default tick = %v; want conid 265598, 31=150.25", frame)
	}

	if n := srv.PushTick(Tick{ConID: 265598, Field: "84", Value: "150.20"}); n != 1 {
		t.Fatalf("PushTick delivered to %d connections; want 1", n)
	}
	frame = readStreamFrame(t, c)
	if frame["84"] != "150.20" {
		t.Fatalf("pushed tick = %v; want 84=150.20", frame)
	}

	writeStreamFrame(t, c, `{"id":2,"method":"unsubscribe","params":{"conids":[265598]}}`)
	waitUnsubscribed(t, srv, c, 265598)
}

func TestStream_TextFramesAndLimits(t *testing.T) {
	srv, c := dialStream(t, &StreamScript{
		Limits: StreamLimits{MaxConIDsPerRequest: 1},
	})

	// Text subscribe frame.
	writeStreamFrame(t, c, `smd+265598+{"fields":["31"]}`)
	if frame := readStreamFrame(t, c); frame["conid"] != float64(265598) {
		t.Fatalf("text subscribe tick = %v; want conid 265598", frame)
	}

	// Text unsubscribe frame drops the subscription.
	writeStreamFrame(t, c, `umd+265598`)
	waitUnsubscribed(t, srv, c, 265598)

	// Server-side limit emits an error frame.
	writeStreamFrame(t, c, `{"id":3,"method":"subscribe","params":{"conids":[1,2]}}`)
	frame := readStreamFrame(t, c)
	if msg, _ := frame["error"].(string); !strings.Contains(msg, "streaming limit") {
		t.Fatalf("limit frame = %v; want error mentioning streaming limit", frame)
	}
}

func TestStream_LimitsCountDistinctConids(t *testing.T) {
	_, c := dialStream(t, &StreamScript{
		Limits: StreamLimits{MaxSubscriptions: 1},
	})

	// (a) An already-held conid does not count as a new subscription.
	writeStreamFrame(t, c, `{"id":1,"method":"subscribe","params":{"conids":[265598]}}`)
	if frame := readStreamFrame(t, c); frame["error"] != nil {
		t.Fatalf("initial subscribe rejected: %v", frame)
	}
	writeStreamFrame(t, c, `{"id":2,"method":"subscribe","params":{"conids":[265598]}}`)
	if frame := readStreamFrame(t, c); frame["error"] != nil {
		t.Fatalf("re-subscribe to held conid rejected: %v", frame)
	}

	// (b) Duplicate conids in one frame count once.
	writeStreamFrame(t, c, `{"id":3,"method":"subscribe","params":{"conids":[265598,265598]}}`)
	if frame := readStreamFrame(t, c); frame["error"] != nil {
		t.Fatalf("duplicate conids rejected: %v", frame)
	}
	// defaultTicks emits one tick per raw conid; drain the duplicate's tick.
	_ = readStreamFrame(t, c)

	// (c) A genuinely new conid exceeds the distinct limit and errors.
	writeStreamFrame(t, c, `{"id":4,"method":"subscribe","params":{"conids":[2]}}`)
	frame := readStreamFrame(t, c)
	if msg, _ := frame["error"].(string); !strings.Contains(msg, "streaming limit") {
		t.Fatalf("new conid frame = %v; want streaming limit error", frame)
	}
}

func TestStream_ErrorOnSubscribe(t *testing.T) {
	_, c := dialStream(t, &StreamScript{
		ErrorOnSubscribe: func(_ []int, _ []string) string { return "no entitlement" },
	})
	writeStreamFrame(t, c, `{"id":1,"method":"subscribe","params":{"conids":[265598]}}`)
	frame := readStreamFrame(t, c)
	if frame["error"] != "no entitlement" {
		t.Fatalf("error frame = %v; want no entitlement", frame)
	}
}

func TestStream_StatusFrame(t *testing.T) {
	_, c := dialStream(t, &StreamScript{Hello: true})
	frame := readStreamFrame(t, c)
	if frame["sts"] != "connected" {
		t.Fatalf("hello frame = %v; want sts=connected", frame)
	}
}
