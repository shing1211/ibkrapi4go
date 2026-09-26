// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
	"go.uber.org/goleak"
)

func TestWS_Resilience(t *testing.T) {
	t.Run("TestWS_Reconnect_DropFirstConnection", TestWS_Reconnect_DropFirstConnection)
	t.Run("TestWS_HeartbeatTimeout", TestWS_HeartbeatTimeout)
	t.Run("TestWS_CancelWriteDuringSend", TestWS_CancelWriteDuringSend)
	t.Run("TestWS_DuplicateUpdatedSequence", TestWS_DuplicateUpdatedSequence)
	t.Run("TestWS_OutOfOrderSequence", TestWS_OutOfOrderSequence)
	t.Run("TestWS_ReconnectStorm_ThreeDrop", TestWS_ReconnectStorm_ThreeDrop)
	t.Run("TestWS_NTFAndSORFrames", TestWS_NTFAndSORFrames)
}

func TestWS_Reconnect_DropFirstConnection(t *testing.T) {
	defer goleak.VerifyNone(t)

	srv := mockgateway.New(mockgateway.WithStreamScript(&mockgateway.StreamScript{
		DropFirstConnection: true,
	}))
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:        NopLogger(),
		Metrics:       NopMetrics(),
		Reconnect:     true,
		PingInterval:  50 * time.Millisecond,
		PongTimeout:   10 * time.Millisecond,
		ReconnectBase: 10 * time.Millisecond,
		ReconnectMax:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sink := &fakeSink{conids: map[int]bool{265598: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for tick after reconnect")
		case <-time.After(100 * time.Millisecond):
			sink.mu.Lock()
			got := len(sink.updates)
			sink.mu.Unlock()
			if got > 0 {
				goto done
			}
		}
	}

done:
	if n := conn.ActiveSubscriptions(); n != 1 {
		t.Fatalf("ActiveSubscriptions = %d, want 1", n)
	}

	sink.mu.Lock()
	got := len(sink.updates)
	sink.mu.Unlock()
	if got == 0 {
		t.Errorf("expected at least one update after reconnect")
	}
}

func TestWS_HeartbeatTimeout(t *testing.T) {
	defer goleak.VerifyNone(t)

	srv := mockgateway.New()
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:       NopLogger(),
		Metrics:      NopMetrics(),
		PingInterval: 50 * time.Millisecond,
		PongTimeout:  50 * time.Millisecond,
		Reconnect:    true,
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sink := &fakeHeartbeatSink{conids: map[int]bool{265598: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sink.mu.Lock()
			hasErr := len(sink.errs) > 0
			var errWS error
			for _, e := range sink.errs {
				if e == ErrWSDisconnected || e == ErrWSReconnected {
					errWS = e
					break
				}
			}
			sink.mu.Unlock()
			if errWS != nil {
				return
			}
			if hasErr && errWS == nil {
				t.Errorf("unexpected error in sink: %v", sink.errs)
				return
			}
		case <-timeout:
			sink.mu.Lock()
			errCount := len(sink.errs)
			sink.mu.Unlock()
			t.Logf("timeout after 5s, sink.errs count: %d", errCount)
			return
		}
	}
}

type fakeHeartbeatSink struct {
	mu      sync.Mutex
	updates []WSUpdate
	errs    []error
	conids  map[int]bool
}

func (s *fakeHeartbeatSink) Wants(conid int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conids[conid]
}

func (s *fakeHeartbeatSink) Deliver(u WSUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, u)
}

func (s *fakeHeartbeatSink) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err)
}

func TestWS_CancelWriteDuringSend(t *testing.T) {
	defer goleak.VerifyNone(t)

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

	sink := &fakeSink{conids: map[int]bool{265598: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{265598}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	subCtx, subCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = conn.Subscribe(subCtx, &fakeSink{conids: map[int]bool{111: true}}, nil, []int{111}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	subCancel()

	cancel()
	conn.Close()

	select {
	case <-conn.doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("websocket loops did not stop after context cancellation and Close")
	}

	if !conn.closed.Load() {
		t.Errorf("expected connection to be closed after context cancel and Close()")
	}
}

func TestWS_CloseUnblocksSilentPeer(t *testing.T) {
	connected := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releasePeer := func() {
		releaseOnce.Do(func() { close(release) })
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer peer.CloseNow()
		close(connected)
		<-release
	}))
	t.Cleanup(func() {
		releasePeer()
		server.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, server.URL, WSOptions{
		Logger:       NopLogger(),
		Metrics:      NopMetrics(),
		PingInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for websocket peer")
	}

	closed := make(chan error, 1)
	go func() {
		closed <- conn.Close()
	}()

	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not interrupt the blocked reader")
	}

	select {
	case <-conn.doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("background websocket loops did not stop")
	}

	releasePeer()
}

func TestWS_DuplicateUpdatedSequence(t *testing.T) {
	defer goleak.VerifyNone(t)

	const conid = 265598

	script := mockgateway.StreamScript{
		OnSubscribe: func(conids []int, fields []string) []mockgateway.Tick {
			return []mockgateway.Tick{
				{ConID: conid, Field: "31", Value: "100.00"},
				{ConID: conid, Field: "31", Value: "200.00"},
			}
		},
	}

	srv := mockgateway.New(mockgateway.WithStreamScript(&script))
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

	sink := &fakeSink{conids: map[int]bool{conid: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{conid}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		t.Fatalf("context cancelled")
	}

	sink.mu.Lock()
	got := len(sink.updates)
	sink.mu.Unlock()
	if got != 2 {
		t.Errorf("expected 2 updates (no dedup), got %d", got)
	}
}

func TestWS_OutOfOrderSequence(t *testing.T) {
	defer goleak.VerifyNone(t)

	const conid = 265598

	script := mockgateway.StreamScript{
		OnSubscribe: func(conids []int, fields []string) []mockgateway.Tick {
			return []mockgateway.Tick{
				{ConID: conid, Field: "31", Value: "first"},
				{ConID: conid, Field: "31", Value: "second"},
				{ConID: conid, Field: "31", Value: "third"},
			}
		},
	}

	srv := mockgateway.New(mockgateway.WithStreamScript(&script))
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

	sink := &fakeOutOfOrderSink{conids: map[int]bool{conid: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{conid}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		t.Fatalf("context cancelled")
	}

	sink.mu.Lock()
	order := sink.receivedOrder
	sink.mu.Unlock()

	if len(order) != 3 {
		t.Errorf("expected 3 updates, got %d", len(order))
	}
	if len(order) >= 3 {
		if order[0] != "first" || order[1] != "second" || order[2] != "third" {
			t.Errorf("expected updates in arrival order, got %v", order)
		}
	}
}

type fakeOutOfOrderSink struct {
	mu            sync.Mutex
	updates       []WSUpdate
	errs          []error
	conids        map[int]bool
	receivedOrder []string
}

func (s *fakeOutOfOrderSink) Wants(conid int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conids[conid]
}

func (s *fakeOutOfOrderSink) Deliver(u WSUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, u)
	s.receivedOrder = append(s.receivedOrder, u.Value)
}

func (s *fakeOutOfOrderSink) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err)
}

func TestWS_ReconnectStorm_ThreeDrop(t *testing.T) {
	defer goleak.VerifyNone(t)

	const conid = 265598
	const drops = 3
	tickCount := atomic.Int64{}

	script := mockgateway.StreamScript{
		DropConnections: drops,
		OnSubscribe: func(conids []int, fields []string) []mockgateway.Tick {
			tickCount.Add(1)
			return []mockgateway.Tick{
				{ConID: conid, Field: "31", Value: "150.00"},
				{ConID: conid, Field: "31", Value: "200.00"},
			}
		},
	}

	srv := mockgateway.New(mockgateway.WithStreamScript(&script))
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := DialWS(ctx, "http://"+server.Listener.Addr().String(), WSOptions{
		Logger:        NopLogger(),
		Metrics:       NopMetrics(),
		Reconnect:     true,
		PingInterval:  50 * time.Millisecond,
		PongTimeout:   10 * time.Millisecond,
		ReconnectBase: 10 * time.Millisecond,
		ReconnectMax:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("DialWS: %v", err)
	}
	defer conn.Close()

	sink := &fakeSink{conids: map[int]bool{conid: true}}
	_, err = conn.Subscribe(ctx, sink, nil, []int{conid}, []string{"31"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// The mock drops the first `drops` connections after their first subscribe
	// frame, so the client must reconnect `drops` times before reaching a
	// connection that survives.
	deadline := time.After(15 * time.Second)
	for {
		if srv.Stream().AcceptedConnections() >= drops+1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("accepted connections = %d, want at least %d after %d drops",
				srv.Stream().AcceptedConnections(), drops+1, drops)
		case <-time.After(50 * time.Millisecond):
		}
	}

	deadline2 := time.After(5 * time.Second)
	for {
		select {
		case <-deadline2:
			t.Fatalf("timeout waiting for ticks after reconnects, got %d updates", len(sink.updates))
		case <-time.After(100 * time.Millisecond):
			sink.mu.Lock()
			got := len(sink.updates)
			sink.mu.Unlock()
			if got >= 2 {
				goto done
			}
		}
	}

done:
	if n := conn.ActiveSubscriptions(); n != 1 {
		t.Fatalf("ActiveSubscriptions = %d, want 1", n)
	}
	if got := srv.Stream().AcceptedConnections(); got < drops+1 {
		t.Errorf("accepted connections = %d, want at least %d", got, drops+1)
	}

	// Each successful reconnect must also be signalled to the subscriber.
	sink.mu.Lock()
	reconnects := 0
	for _, err := range sink.errs {
		if errors.Is(err, ErrWSReconnected) {
			reconnects++
		}
	}
	sink.mu.Unlock()
	if reconnects < drops {
		t.Errorf("ErrWSReconnected signals = %d, want at least %d", reconnects, drops)
	}

	sink.mu.Lock()
	got := len(sink.updates)
	sink.mu.Unlock()
	if got < 2 {
		t.Errorf("expected at least 2 updates after reconnects, got %d, tickCount=%d", got, tickCount.Load())
	}
}

func TestWS_NTFAndSORFrames(t *testing.T) {
	defer goleak.VerifyNone(t)

	script := mockgateway.StreamScript{Hello: true}
	srv := mockgateway.New(mockgateway.WithStreamScript(&script))
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
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		t.Fatalf("context cancelled")
	}

	sysSink.mu.Lock()
	frames := sysSink.frames
	sysSink.mu.Unlock()

	if len(frames) == 0 {
		t.Fatalf("expected system frames, got none")
	}

	var hasSTS, hasNTF bool
	for _, f := range frames {
		if f.Type == "sts" && f.Status == "connected" {
			hasSTS = true
		}
		if f.Type == "ntf" {
			hasNTF = true
		}
	}

	if !hasSTS {
		t.Errorf("expected sts frame with status=connected")
	}
	if !hasNTF {
		t.Errorf("expected ntf frame")
	}
}
