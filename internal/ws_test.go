// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

func TestBackoffDelay(t *testing.T) {
	base := 100 * time.Millisecond
	max := 2 * time.Second

	got := backoffDelay(0, base, max)
	if got < 0 || got > base {
		t.Errorf("attempt 0: got %v, want 0..%v", got, base)
	}

	got = backoffDelay(1, base, max)
	if got < 0 || got > base*2 {
		t.Errorf("attempt 1: got %v, want 0..%v", got, base*2)
	}

	got = backoffDelay(5, base, max)
	if got < 0 || got > max {
		t.Errorf("attempt 5: got %v, want 0..%v", got, max)
	}

	got = backoffDelay(-1, base, max)
	if got < 0 || got > base {
		t.Errorf("attempt -1: got %v, want 0..%v (treated as 0)", got, base)
	}

	got = backoffDelay(10, base, max)
	if got < 0 || got > max {
		t.Errorf("attempt 10: got %v, want 0..%v (capped at 5)", got, max)
	}
}

func TestWsURLFromGateway(t *testing.T) {
	tests := []struct {
		gateway string
		want    string
		wantErr bool
	}{
		{"https://localhost:5000", "wss://localhost:5000/v1/api/ws", false},
		{"http://localhost:5000", "ws://localhost:5000/v1/api/ws", false},
		{"wss://localhost:5000", "wss://localhost:5000/v1/api/ws", false},
		{"ws://localhost:5000", "ws://localhost:5000/v1/api/ws", false},
		{"https://host/prefix/", "wss://host/prefix/v1/api/ws", false},
		{"http://host/prefix", "ws://host/prefix/v1/api/ws", false},
		{"ftp://localhost", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := wsURLFromGateway(tt.gateway)
		if tt.wantErr {
			if err == nil {
				t.Errorf("wsURLFromGateway(%q): want error, got nil", tt.gateway)
			}
			continue
		}
		if err != nil {
			t.Errorf("wsURLFromGateway(%q): unexpected error: %v", tt.gateway, err)
			continue
		}
		if got != tt.want {
			t.Errorf("wsURLFromGateway(%q) = %q, want %q", tt.gateway, got, tt.want)
		}
	}
}

func TestWsScalarString(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{`"hello"`, "hello", true},
		{`"123"`, "123", true},
		{`123`, "123", true},
		{`123.45`, "123.45", true},
		{`true`, "true", true},
		{`false`, "false", true},
		{`null`, "", false},
		{`""`, "", true},
		{`{}`, "", false},
		{`[]`, "", false},
		{`[1,2]`, "", false},
		{`{"a":1}`, "", false},
		{`  "str"  `, "str", true},
		{`  42  `, "42", true},
	}

	for _, tt := range tests {
		got, ok := wsScalarString(json.RawMessage(tt.input))
		if ok != tt.ok {
			t.Errorf("wsScalarString(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("wsScalarString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWsReservedField(t *testing.T) {
	reserved := []string{"conid", "_updated", "server_id", "6119", "6509", "topic", "method", "id"}
	for _, k := range reserved {
		if !wsReservedField(k) {
			t.Errorf("wsReservedField(%q) = false, want true", k)
		}
	}

	notReserved := []string{"31", "55", "field_name", "something_else", "CONID", "Conid"}
	for _, k := range notReserved {
		if wsReservedField(k) {
			t.Errorf("wsReservedField(%q) = true, want false", k)
		}
	}
}

func TestParseSystemFrame(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]json.RawMessage
		want  *WSSystemFrame
	}{
		{
			name:  "sts frame",
			input: map[string]json.RawMessage{"sts": toRaw(`"connected"`), "topic": toRaw(`"sts"`)},
			want:  &WSSystemFrame{Type: "sts", Status: "connected", Topic: "sts"},
		},
		{
			name:  "ntf frame",
			input: map[string]json.RawMessage{"ntf": toRaw(`"hello"`), "topic": toRaw(`"ntf"`)},
			want:  &WSSystemFrame{Type: "ntf", Topic: "ntf", Payload: toRaw(`"hello"`)},
		},
		{
			name:  "sor frame",
			input: map[string]json.RawMessage{"sor": toRaw(`{"order_id":123}`)},
			want:  &WSSystemFrame{Type: "sor", Payload: toRaw(`{"order_id":123}`)},
		},
		{
			name:  "usr frame",
			input: map[string]json.RawMessage{"usr": toRaw(`{"user":"x"}`)},
			want:  &WSSystemFrame{Type: "usr", Payload: toRaw(`{"user":"x"}`)},
		},
		{
			name:  "empty map",
			input: map[string]json.RawMessage{},
			want:  nil,
		},
		{
			name:  "market data with conid",
			input: map[string]json.RawMessage{"conid": toRaw(`265598`), "31": toRaw(`"150.25"`)},
			want:  nil,
		},
		{
			name:  "random fields no system key",
			input: map[string]json.RawMessage{"foo": toRaw(`"bar"`)},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSystemFrame(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseSystemFrame() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseSystemFrame() = nil, want %+v", tt.want)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.Topic != tt.want.Topic {
				t.Errorf("Topic = %q, want %q", got.Topic, tt.want.Topic)
			}
		})
	}
}

func toRaw(s string) json.RawMessage { return json.RawMessage(s) }

type fakeSink struct {
	mu      sync.Mutex
	updates []WSUpdate
	errs    []error
	conids  map[int]bool
}

func (s *fakeSink) Wants(conid int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conids[conid]
}

func (s *fakeSink) Deliver(u WSUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, u)
}

func (s *fakeSink) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err)
}

type fakeSystemSink struct {
	mu     sync.Mutex
	frames []WSSystemFrame
	errs   []error
}

func (s *fakeSystemSink) WantsSystem() bool { return true }

func (s *fakeSystemSink) DeliverSystem(f WSSystemFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, f)
}

func (s *fakeSystemSink) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err)
}

func TestWS_DialAndSubscribe(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sink := &fakeSink{conids: map[int]bool{265598: true}}
	handle, err := conn.Subscribe(ctx, sink, nil, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer handle.Close()

	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		t.Fatalf("timeout waiting for tick")
	}

	sink.mu.Lock()
	got := len(sink.updates)
	sink.mu.Unlock()
	if got == 0 {
		t.Errorf("expected at least one update after 500ms")
	}
}

func TestWS_SubscribeDeliversToCorrectSink(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()
	hub := srv.Stream()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sinkA := &fakeSink{conids: map[int]bool{111: true}}
	sinkB := &fakeSink{conids: map[int]bool{222: true}}

	_, err = conn.Subscribe(ctx, sinkA, nil, []int{111}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe A: %v", err)
	}
	_, err = conn.Subscribe(ctx, sinkB, nil, []int{222}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe B: %v", err)
	}

	hub.Push(mockgateway.Tick{ConID: 111, Field: "31", Value: "100.00"})
	hub.Push(mockgateway.Tick{ConID: 222, Field: "31", Value: "200.00"})
	hub.Push(mockgateway.Tick{ConID: 333, Field: "31", Value: "300.00"})

	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		t.Fatalf("timeout")
	}

	sinkA.mu.Lock()
	aUpdates := len(sinkA.updates)
	sinkA.mu.Unlock()

	sinkB.mu.Lock()
	bUpdates := len(sinkB.updates)
	sinkB.mu.Unlock()

	if aUpdates == 0 {
		t.Errorf("sinkA got no updates for conid 111")
	}
	if bUpdates == 0 {
		t.Errorf("sinkB got no updates for conid 222")
	}
}

func TestWS_SystemFrameDeliversToSystemSink(t *testing.T) {
	helloScript := mockgateway.StreamScript{Hello: true}
	srv := mockgateway.New(mockgateway.WithStreamScript(&helloScript))
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sysSink := &fakeSystemSink{}
	_, err = conn.Subscribe(ctx, &fakeSink{conids: map[int]bool{265598: true}}, sysSink, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		t.Fatalf("timeout waiting for system frame")
	}

	sysSink.mu.Lock()
	got := len(sysSink.frames)
	sysSink.mu.Unlock()
	if got == 0 {
		t.Errorf("expected at least one system frame")
	}
}

func TestWS_CloseUnsubscribes(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}

	handle, err := conn.Subscribe(ctx, &fakeSink{conids: map[int]bool{265598: true}}, nil, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if n := conn.ActiveSubscriptions(); n != 1 {
		t.Fatalf("ActiveSubscriptions = %d, want 1", n)
	}

	handle.Close()

	if n := conn.ActiveSubscriptions(); n != 0 {
		t.Fatalf("ActiveSubscriptions after Close = %d, want 0", n)
	}

	conn.Close()
}

func TestWS_CloseIdempotent(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}

	conn.Close()
	conn.Close()
	conn.Close()
}

func TestWS_ActiveSubscriptions(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	if n := conn.ActiveSubscriptions(); n != 0 {
		t.Fatalf("initial ActiveSubscriptions = %d, want 0", n)
	}

	h1, err := conn.Subscribe(ctx, &fakeSink{conids: map[int]bool{111: true}}, nil, []int{111}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe 1: %v", err)
	}
	if n := conn.ActiveSubscriptions(); n != 1 {
		t.Fatalf("after sub 1: ActiveSubscriptions = %d, want 1", n)
	}

	h2, err := conn.Subscribe(ctx, &fakeSink{conids: map[int]bool{222: true}}, nil, []int{222}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe 2: %v", err)
	}
	if n := conn.ActiveSubscriptions(); n != 2 {
		t.Fatalf("after sub 2: ActiveSubscriptions = %d, want 2", n)
	}

	h1.Close()
	if n := conn.ActiveSubscriptions(); n != 1 {
		t.Fatalf("after close 1: ActiveSubscriptions = %d, want 1", n)
	}

	h2.Close()
	if n := conn.ActiveSubscriptions(); n != 0 {
		t.Fatalf("after close 2: ActiveSubscriptions = %d, want 0", n)
	}
}

func TestWS_ConcurrentSubscribeClose(t *testing.T) {
	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:  NopLogger(),
		Metrics: NopMetrics(),
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sink := &fakeSink{conids: map[int]bool{i: true}}
			h, err := conn.Subscribe(ctx, sink, nil, []int{i}, []string{"31"})
			if err != nil {
				errCh <- err
				return
			}
			h.Close()
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}
}
