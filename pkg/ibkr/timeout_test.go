// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"errors"
	"testing"
	"time"
)

func TestWithEndpointTimeout(t *testing.T) {
	var cfg config
	opt := WithEndpointTimeout(5 * time.Second)
	if err := opt(&cfg); err != nil {
		t.Fatalf("WithEndpointTimeout: %v", err)
	}
	if cfg.endpointTimeout != 5*time.Second {
		t.Errorf("endpointTimeout = %v; want 5s", cfg.endpointTimeout)
	}
}

func TestWithEndpointTimeout_Negative(t *testing.T) {
	var cfg config
	opt := WithEndpointTimeout(-1 * time.Second)
	err := opt(&cfg)
	if err == nil {
		t.Fatal("WithEndpointTimeout(-1s): want error")
	}
	var ce *ConfigError
	if !errors.As(err, &ce) {
		t.Errorf("error type = %T; want *ConfigError", err)
	}
	if ce.Field != "EndpointTimeout" {
		t.Errorf("Field = %q; want %q", ce.Field, "EndpointTimeout")
	}
}

func TestWithEndpointTimeout_Zero(t *testing.T) {
	var cfg config
	opt := WithEndpointTimeout(0)
	if err := opt(&cfg); err != nil {
		t.Fatalf("WithEndpointTimeout(0): %v", err)
	}
	if cfg.endpointTimeout != 0 {
		t.Errorf("endpointTimeout = %v; want 0", cfg.endpointTimeout)
	}
}
