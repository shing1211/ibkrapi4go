// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

const DefaultServerURL = "https://localhost:5000"

type Config struct {
	ServerURL      string
	HTTPClient     *http.Client
	TickleInterval time.Duration
	RequestTimeout time.Duration
}

type Client struct {
	session   *internal.Session
	generated *client.ClientWithResponses
	serverURL string
}

type Option func(*Client) error

func New(cfg Config, opts ...Option) (*Client, error) {
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}

	session := internal.NewSession(internal.SessionConfig{
		HTTPClient:     cfg.HTTPClient,
		ServerURL:      cfg.ServerURL,
		TickleInterval: cfg.TickleInterval,
		RequestTimeout: cfg.RequestTimeout,
	})

	generated, err := client.NewClientWithResponses(
		cfg.ServerURL,
		client.WithHTTPClient(cfg.HTTPClient),
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		session:   session,
		generated: generated,
		serverURL: cfg.ServerURL,
	}

	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) HTTPClient() *http.Client {
	return c.session.HTTPClient()
}

func (c *Client) Initialize(ctx context.Context) error {
	return c.session.Initialize(ctx)
}

func (c *Client) Close(ctx context.Context) error {
	return c.session.Close(ctx)
}

func (c *Client) State() internal.SessionState {
	return c.session.State()
}

func (c *Client) ServerURL() string {
	return c.serverURL
}
