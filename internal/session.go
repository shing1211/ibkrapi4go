// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SessionState int

const (
	StateDisconnected SessionState = iota
	StateInitializing
	StateAuthenticated
	StateExpired
	StateClosed
)

func (s SessionState) String() string {
	switch s {
	case StateDisconnected:
		return "DISCONNECTED"
	case StateInitializing:
		return "INITIALIZING"
	case StateAuthenticated:
		return "AUTHENTICATED"
	case StateExpired:
		return "EXPIRED"
	case StateClosed:
		return "CLOSED"
	default:
		return fmt.Sprintf("SessionState(%d)", int(s))
	}
}

type BrokerageStatus struct {
	Authenticated bool
	Established   bool
	Connected     bool
	Fail          string
	Message       string
}

type api interface {
	initSession(ctx context.Context) (*http.Response, error)
	authStatus(ctx context.Context) (*http.Response, error)
	tickle(ctx context.Context) (*http.Response, error)
	logout(ctx context.Context) error
}

type httpAPI struct {
	client    *http.Client
	serverURL string
}

func newHTTPAPI(client *http.Client, serverURL string) *httpAPI {
	return &httpAPI{client: client, serverURL: serverURL}
}

func (h *httpAPI) initSession(ctx context.Context) (*http.Response, error) {
	body := `{"compete":false,"publish":true}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(h.serverURL, "/")+"/v1/api/iserver/auth/ssodh/init",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.client.Do(req)
}

func (h *httpAPI) authStatus(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		strings.TrimSuffix(h.serverURL, "/")+"/v1/api/iserver/auth/status",
		nil)
	if err != nil {
		return nil, err
	}
	return h.client.Do(req)
}

func (h *httpAPI) tickle(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(h.serverURL, "/")+"/v1/api/tickle",
		nil)
	if err != nil {
		return nil, err
	}
	return h.client.Do(req)
}

func (h *httpAPI) logout(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(h.serverURL, "/")+"/v1/api/logout",
		nil)
	if err != nil {
		return err
	}
	_, err = h.client.Do(req)
	return err
}

type Session struct {
	api             api
	token           string
	state           int32
	mu              sync.RWMutex
	tickleInterval  time.Duration
	requestTimeout  time.Duration
	consecutiveFail atomic.Int64
	tickleStop      atomic.Bool
	tickleStopCh    chan struct{}
	tickleDoneCh    chan struct{}
	logger          *slog.Logger
}

type SessionConfig struct {
	HTTPClient     *http.Client
	ServerURL      string
	TickleInterval time.Duration
	RequestTimeout time.Duration
	Logger         *slog.Logger
}

func NewSession(cfg SessionConfig) *Session {
	if cfg.TickleInterval == 0 {
		cfg.TickleInterval = 60 * time.Second
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &Session{
		api:            newHTTPAPI(cfg.HTTPClient, cfg.ServerURL),
		tickleInterval: cfg.TickleInterval,
		requestTimeout: cfg.RequestTimeout,
		logger:         cfg.Logger,
		state:          int32(StateDisconnected),
	}
}

func (s *Session) State() SessionState {
	return SessionState(atomic.LoadInt32(&s.state))
}

func (s *Session) setState(state SessionState) {
	atomic.StoreInt32(&s.state, int32(state))
}

func (s *Session) Token() (token string, ok bool) {
	if SessionState(atomic.LoadInt32(&s.state)) != StateAuthenticated {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.token == "" {
		return "", false
	}
	return s.token, true
}

func (s *Session) HTTPClient() *http.Client {
	return s.api.(*httpAPI).client
}

func (s *Session) Initialize(ctx context.Context) error {
	current := SessionState(atomic.LoadInt32(&s.state))
	if current == StateAuthenticated {
		return nil
	}
	if current == StateClosed {
		return errors.New("ibkr: session is closed")
	}

	s.setState(StateInitializing)

	if s.logger != nil {
		s.logger.Info("session: initializing")
	}

	{
		ctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
		defer cancel()
		resp, err := s.api.initSession(ctx)
		if err != nil {
			s.setState(StateDisconnected)
			return fmt.Errorf("ibkr: session init request: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			s.setState(StateDisconnected)
			return fmt.Errorf("ibkr: session init: server returned %d", resp.StatusCode)
		}
	}

	{
		pollCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
		defer cancel()
		time.Sleep(1 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pollCtx.Done():
				s.setState(StateDisconnected)
				return fmt.Errorf("ibkr: session init: timeout waiting for established: %w", pollCtx.Err())
			case <-ticker.C:
				bs, err := s.fetchAuthStatus(pollCtx)
				if err != nil {
					if s.logger != nil {
						s.logger.Warn("session: auth_status error", "err", err)
					}
					continue
				}
				if bs.Established && bs.Authenticated {
					s.setState(StateAuthenticated)
					if s.logger != nil {
						s.logger.Info("session: authenticated", "connected", bs.Connected)
					}
					s.startTickle()
					return nil
				}
				if bs.Fail != "" {
					s.setState(StateDisconnected)
					return fmt.Errorf("ibkr: session init: server rejected: %s", bs.Fail)
				}
			}
		}
	}
}

func (s *Session) fetchAuthStatus(ctx context.Context) (*BrokerageStatus, error) {
	resp, err := s.api.authStatus(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth/status: %d", resp.StatusCode)
	}
	var data struct {
		Authenticated *bool   `json:"authenticated"`
		Established   *bool   `json:"established"`
		Connected     *bool   `json:"connected"`
		Fail          *string `json:"fail"`
		Message       *string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &BrokerageStatus{
		Authenticated: derefBool(data.Authenticated, false),
		Established:   derefBool(data.Established, false),
		Connected:     derefBool(data.Connected, false),
		Fail:          derefString(data.Fail, ""),
		Message:       derefString(data.Message, ""),
	}, nil
}

func (s *Session) startTickle() {
	s.mu.Lock()
	if s.tickleStop.Load() || SessionState(atomic.LoadInt32(&s.state)) != StateAuthenticated {
		s.mu.Unlock()
		return
	}
	if s.tickleDoneCh != nil {
		s.mu.Unlock()
		return
	}
	interval := s.tickleInterval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	s.tickleStopCh = stopCh
	s.tickleDoneCh = doneCh
	s.consecutiveFail.Store(0)
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.tickleRound()
			case <-stopCh:
				return
			}
		}
	}()
}

func (s *Session) tickleRound() {
	if s.tickleStop.Load() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	resp, err := s.api.tickle(ctx)
	if err != nil {
		s.onTickleFailure()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		s.onTickleFailure()
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.onTickleFailure()
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		s.onTickleFailure()
		return
	}

	var data struct {
		Session    *string `json:"session"`
		SsoExpires *int64  `json:"ssoExpires,omitempty"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		s.onTickleFailure()
		return
	}

	token := derefString(data.Session, "")
	s.mu.Lock()
	if !s.tickleStop.Load() && token != "" {
		s.token = token
		s.consecutiveFail.Store(0)
	}
	s.mu.Unlock()
}

func (s *Session) onTickleFailure() {
	fails := s.consecutiveFail.Add(1)
	if s.logger != nil {
		s.logger.Warn("session: tickle failure", "consecutive", fails)
	}

	if fails >= 2 {
		s.setState(StateExpired)
		if s.logger != nil {
			s.logger.Warn("session: expired after tickle failures")
		}
	}
}

func (s *Session) stopTickle() {
	if !s.tickleStop.CompareAndSwap(false, true) {
		return
	}
	s.mu.Lock()
	stopCh := s.tickleStopCh
	doneCh := s.tickleDoneCh
	s.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}
}

func (s *Session) Close(ctx context.Context) error {
	current := SessionState(atomic.LoadInt32(&s.state))
	if current == StateClosed || current == StateDisconnected {
		s.setState(StateClosed)
		return nil
	}
	s.setState(StateClosed)

	s.stopTickle()

	if s.logger != nil {
		s.logger.Info("session: closing")
	}
	_ = s.api.logout(ctx)
	return nil
}

func derefBool(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func derefString(v *string, def string) string {
	if v == nil {
		return def
	}
	return *v
}
