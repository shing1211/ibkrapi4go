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
	"sync/atomic"
	"testing"
	"time"
)

// testLogger discards output; used by tests that construct Session directly.
var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakeAPI struct {
	initSessionFn func(ctx context.Context) (*http.Response, error)
	authStatusFn  func(ctx context.Context) (*http.Response, error)
	tickleFn      func(ctx context.Context) (*http.Response, error)
	logoutFn      func(ctx context.Context) error
	logoutCalls   atomic.Int64
	// tickleCalls counts served tickle rounds, so a test can wait for the loop
	// to have run rather than sleeping and hoping.
	tickleCalls atomic.Int64
}

func (f *fakeAPI) initSession(ctx context.Context) (*http.Response, error) {
	return f.initSessionFn(ctx)
}
func (f *fakeAPI) authStatus(ctx context.Context) (*http.Response, error) {
	return f.authStatusFn(ctx)
}
func (f *fakeAPI) tickle(ctx context.Context) (*http.Response, error) {
	resp, err := f.tickleFn(ctx)
	f.tickleCalls.Add(1)
	return resp, err
}
func (f *fakeAPI) logout(ctx context.Context) error {
	f.logoutCalls.Add(1)
	return f.logoutFn(ctx)
}

func makeResp(status int, v any) *http.Response {
	body, _ := json.Marshal(v)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{},
	}
}

func fakeAuthStatus(authenticated, established, connected bool, fail string) *http.Response {
	return makeResp(http.StatusOK, map[string]any{
		"authenticated": authenticated,
		"established":   established,
		"connected":     connected,
		"fail":          fail,
	})
}

func TestSession_HappyPath(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "test-token-abc"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{
		api:            api,
		tickleInterval: 1 * time.Millisecond,
		requestTimeout: 10 * time.Second,
		state:          int32(StateDisconnected),
		logger:         testLogger,
	}

	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if s.State() != StateAuthenticated {
		t.Errorf("State = %v; want AUTHENTICATED", s.State())
	}

	// The session token is written only by a tickle round - Initialize does not
	// set it - so the caller-visible token exists once a round has run. This used
	// to sleep 20ms against a 1ms interval and hope one had landed. Driving the
	// round directly makes it exact and instant.
	s.tickleRound()

	token, ok := s.Token()
	if !ok || token != "test-token-abc" {
		t.Errorf("Token = %q, ok=%v; want %q, true", token, ok, "test-token-abc")
	}
}

func TestSession_Initialize_Idempotent(t *testing.T) {
	callCount := atomic.Int64{}
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			callCount.Add(1)
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{api: api, state: int32(StateAuthenticated), logger: testLogger}
	ctx := context.Background()
	defer settleGoroutines(t)
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if callCount.Load() != 0 {
		t.Errorf("initSession called %d times; want 0 for already-AUTHENTICATED", callCount.Load())
	}
}

func TestSession_TickleFailure_Expires(t *testing.T) {
	tickleCalls := atomic.Int64{}
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			tickleCalls.Add(1)
			return makeResp(http.StatusInternalServerError, nil), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{
		api:            api,
		tickleInterval: 50 * time.Millisecond,
		requestTimeout: 10 * time.Second,
		state:          int32(StateDisconnected),
		logger:         testLogger,
	}

	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Drive exactly the number of failing rounds the expiry threshold needs.
	//
	// This used to sleep 250ms - five times the 50ms tickle interval - and then
	// assert on whatever state the loop happened to reach. A loaded machine
	// could see fewer than two rounds and fail for a reason unrelated to the
	// code. Calling tickleRound is what the loop itself does once per tick, so
	// the threshold logic is exercised exactly, with no timing involved.
	//
	// The loop's own wiring is covered separately, by the goroutine tests below.
	for i := 0; i < 2; i++ {
		s.tickleRound()
	}

	if s.State() != StateExpired {
		t.Errorf("State = %v; want EXPIRED after tickle failures", s.State())
	}
	if tickleCalls.Load() < 2 {
		t.Errorf("tickle called %d times; want at least 2", tickleCalls.Load())
	}
}

func TestSession_ReinitializeFromExpired(t *testing.T) {
	initCalls := atomic.Int64{}
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			initCalls.Add(1)
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "new-token"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{api: api, requestTimeout: 10 * time.Second, state: int32(StateExpired), logger: testLogger}
	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Reinitialize from EXPIRED: %v", err)
	}
	if s.State() != StateAuthenticated {
		t.Errorf("State = %v; want AUTHENTICATED", s.State())
	}
	if initCalls.Load() != 1 {
		t.Errorf("initSession called %d times; want 1", initCalls.Load())
	}
}

func TestSession_Close_Idempotent(t *testing.T) {
	logoutCalls := atomic.Int64{}
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error {
			logoutCalls.Add(1)
			return nil
		},
	}
	s := &Session{api: api, state: int32(StateAuthenticated), logger: testLogger}
	ctx := context.Background()

	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if s.State() != StateClosed {
		t.Errorf("State = %v; want CLOSED", s.State())
	}
	if logoutCalls.Load() != 1 {
		t.Errorf("logout called %d times; want 1", logoutCalls.Load())
	}

	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close (idempotent): %v", err)
	}
	if logoutCalls.Load() != 1 {
		t.Errorf("logout called %d times on second close; want 1 (idempotent)", logoutCalls.Load())
	}
}

func TestSession_CloseFromDisconnected(t *testing.T) {
	api := &fakeAPI{logoutFn: func(ctx context.Context) error { return nil }}
	s := &Session{api: api, state: int32(StateDisconnected), logger: testLogger}
	ctx := context.Background()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close from DISCONNECTED: %v", err)
	}
	if s.State() != StateClosed {
		t.Errorf("State = %v; want CLOSED", s.State())
	}
}

func TestSession_CloseFromClosed_IsIdempotent(t *testing.T) {
	api := &fakeAPI{logoutFn: func(ctx context.Context) error { return nil }}
	s := &Session{api: api, state: int32(StateClosed), logger: testLogger}
	ctx := context.Background()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close from CLOSED: %v", err)
	}
	if s.State() != StateClosed {
		t.Errorf("State = %v; want CLOSED", s.State())
	}
}

func TestSession_InitError_AuthRejected(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": false, "established": false}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(false, false, false, "auth failed"), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{api: api, requestTimeout: 1 * time.Second, state: int32(StateDisconnected), logger: testLogger}
	ctx := context.Background()
	defer s.Close(ctx)

	err := s.Initialize(ctx)
	if err == nil {
		t.Error("Initialize: expected error for rejected auth; got nil")
	}
	if s.State() != StateDisconnected {
		t.Errorf("State = %v; want DISCONNECTED", s.State())
	}
}

func TestSession_StateTransitions(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{api: api, requestTimeout: 10 * time.Second, state: int32(StateDisconnected), logger: testLogger}
	ctx := context.Background()
	defer s.Close(ctx)

	if s.State() != StateDisconnected {
		t.Errorf("initial state = %v; want DISCONNECTED", s.State())
	}

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if s.State() != StateAuthenticated {
		t.Errorf("after Initialize: state = %v; want AUTHENTICATED", s.State())
	}

	s.Close(ctx)
	if s.State() != StateClosed {
		t.Errorf("after Close: state = %v; want CLOSED", s.State())
	}
}

func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state SessionState
		want  string
	}{
		{StateDisconnected, "DISCONNECTED"},
		{StateInitializing, "INITIALIZING"},
		{StateAuthenticated, "AUTHENTICATED"},
		{StateExpired, "EXPIRED"},
		{StateClosed, "CLOSED"},
		{SessionState(99), "SessionState(99)"},
	}
	for _, tc := range tests {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("SessionState(%d).String() = %q; want %q", tc.state, got, tc.want)
		}
	}
}

func TestSession_Token_NotAuthenticated(t *testing.T) {
	s := &Session{state: int32(StateDisconnected), logger: testLogger}
	if tok, ok := s.Token(); ok || tok != "" {
		t.Errorf("Token from DISCONNECTED: got %q, ok=%v; want %q, false", tok, ok, "")
	}
}

func TestSession_ReauthorizeAfterTickleFailure(t *testing.T) {
	tickleCalls := atomic.Int64{}
	failFirst := atomic.Bool{}
	failFirst.Store(true)
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			tickleCalls.Add(1)
			if failFirst.Load() {
				failFirst.Store(false)
				return makeResp(http.StatusInternalServerError, nil), nil
			}
			return makeResp(http.StatusOK, map[string]string{"session": "recovered-token"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{
		api:            api,
		tickleInterval: 50 * time.Millisecond,
		requestTimeout: 10 * time.Second,
		state:          int32(StateDisconnected),
		logger:         testLogger,
	}
	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// One failing round must not expire the session: the threshold is two.
	// Driven directly rather than by sleeping through the 50ms interval, which
	// is what this test used to do.
	s.tickleRound()

	if s.State() == StateExpired {
		t.Errorf("State = EXPIRED after 1 failure; want AUTHENTICATED (threshold=2)")
	}
	if s.State() != StateAuthenticated {
		t.Errorf("State = %v; want AUTHENTICATED", s.State())
	}
}

func TestSession_TokenCopy(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok1"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{api: api, tickleInterval: 1 * time.Millisecond, requestTimeout: 10 * time.Second, state: int32(StateDisconnected), logger: testLogger}
	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// The token the caller reads is the one the tickle loop most recently
	// fetched, so drive one round and then read it. This used to sleep 20ms
	// against a 1ms interval and hope a round had landed.
	s.tickleRound()

	tok1, ok := s.Token()
	if !ok || tok1 != "tok1" {
		t.Errorf("Token = %q, ok=%v; want tok1, true", tok1, ok)
	}
}

func TestSession_StartTickle_Idempotent(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	clock, mt := newManualClock()
	s := &Session{api: api, clock: clock, state: int32(StateAuthenticated), logger: testLogger}
	ctx := context.Background()
	defer settleGoroutines(t)
	defer s.Close(ctx)

	s.startTickle()
	s.startTickle() // idempotent

	// The second startTickle must not have replaced the running loop, so exactly
	// one loop exists and one tick produces one round.
	//
	// This previously slept 20ms and compared two token reads, which asserted
	// nothing: tickleFn always returned the same token, so the reads were equal
	// whether or not a second loop had been started. The manual clock gives each
	// loop its own channel, so a second loop would consume the same tick and the
	// count would be 2.
	mt.waitForTicker(t)
	mt.tick()
	waitForTickles(t, api, 1)

	if got := mt.count(); got != 1 {
		t.Errorf("tickers = %d; want 1 - a second startTickle must not start a second loop", got)
	}
	if got := api.tickleCalls.Load(); got != 1 {
		t.Errorf("tickle rounds = %d; want 1", got)
	}
}

func TestSession_TickleGoroutine_NoLeak(t *testing.T) {
	api := &fakeAPI{
		initSessionFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]any{"authenticated": true, "established": true}), nil
		},
		authStatusFn: func(ctx context.Context) (*http.Response, error) {
			return fakeAuthStatus(true, true, true, ""), nil
		},
		tickleFn: func(ctx context.Context) (*http.Response, error) {
			return makeResp(http.StatusOK, map[string]string{"session": "tok"}), nil
		},
		logoutFn: func(ctx context.Context) error { return nil },
	}
	s := &Session{
		api:            api,
		tickleInterval: 1 * time.Millisecond,
		requestTimeout: 10 * time.Second,
		state:          int32(StateDisconnected),
		logger:         testLogger,
	}

	ctx := context.Background()
	defer s.Close(ctx)

	if err := s.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Wait for the loop to have actually served a round, so the leak check that
	// Close triggers has a live goroutine to account for. The previous version
	// slept 20ms against a 1ms tickle interval, which is the right idea and the
	// wrong mechanism: it could wake before any round had run, and it could not
	// tell a slow loop from a broken one. This fails with a specific message
	// instead of passing vacuously.
	waitForTickles(t, api, 1)
}
