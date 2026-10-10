// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"testing"
)

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status HealthStatus
		want   string
	}{
		{HealthUnknown, "unknown"},
		{HealthHealthy, "healthy"},
		{HealthDegraded, "degraded"},
		{HealthUnhealthy, "unhealthy"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("HealthStatus(%d).String() = %q; want %q", tt.status, got, tt.want)
		}
	}
}

func TestWorstOf(t *testing.T) {
	tests := []struct {
		input []HealthStatus
		want  HealthStatus
	}{
		{[]HealthStatus{HealthHealthy, HealthHealthy}, HealthHealthy},
		{[]HealthStatus{HealthHealthy, HealthDegraded}, HealthDegraded},
		{[]HealthStatus{HealthHealthy, HealthUnhealthy}, HealthUnhealthy},
		{[]HealthStatus{HealthDegraded, HealthUnhealthy}, HealthUnhealthy},
		{[]HealthStatus{HealthUnknown, HealthHealthy}, HealthHealthy},
		{[]HealthStatus{HealthUnknown, HealthDegraded}, HealthDegraded},
		{[]HealthStatus{HealthUnknown, HealthUnhealthy}, HealthUnhealthy},
		{[]HealthStatus{HealthUnknown, HealthUnknown}, HealthUnknown},
		{[]HealthStatus{HealthDegraded, HealthDegraded}, HealthDegraded},
		{[]HealthStatus{HealthUnhealthy, HealthUnhealthy}, HealthUnhealthy},
	}
	for _, tt := range tests {
		got := worstOf(tt.input...)
		if got != tt.want {
			t.Errorf("worstOf(%v) = %v; want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		err    error
		isConn bool
	}{
		{nil, false},
		{&Error{Message: "connection reset"}, true},
		{&Error{Message: "connection refused"}, true},
		{&Error{Message: "connection timeout"}, true},
		{&Error{Message: "timeout"}, true},
		{&Error{Message: "unreachable"}, true},
		{&Error{Message: "some other error"}, false},
		{&Error{Message: "invalid request"}, false},
	}
	for _, tt := range tests {
		got := isConnectionError(tt.err)
		if got != tt.isConn {
			t.Errorf("isConnectionError(%v) = %v; want %v", tt.err, got, tt.isConn)
		}
	}
}
