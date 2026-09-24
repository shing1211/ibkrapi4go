// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"strings"
	"time"
)

type HealthStatus int

const (
	HealthUnknown   HealthStatus = iota
	HealthHealthy
	HealthDegraded
	HealthUnhealthy
)

func (s HealthStatus) String() string {
	switch s {
	case HealthHealthy:
		return "healthy"
	case HealthDegraded:
		return "degraded"
	case HealthUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

type Dimension struct {
	Status  HealthStatus
	Message string
	Since   time.Time
}

type HealthReport struct {
	Transport  Dimension
	Session    Dimension
	MarketData Dimension
	Overall    HealthStatus
	ReportedAt time.Time
}

func (m *SessionManager) Health(ctx context.Context) *HealthReport {
	report := &HealthReport{ReportedAt: time.Now()}

	state := m.State()
	status, err := m.Status(ctx)

	report.Session = m.healthSession(state)
	report.Transport = m.healthTransport(status, err)
	report.MarketData = m.healthMarketData(status, err)

	report.Overall = worstOf(report.Transport.Status, report.Session.Status, report.MarketData.Status)
	return report
}

func (m *SessionManager) healthSession(state SessionState) Dimension {
	switch state {
	case StateAuthenticated:
		return Dimension{Status: HealthHealthy, Message: "active", Since: time.Now()}
	case StateExpired:
		return Dimension{Status: HealthUnhealthy, Message: "expired", Since: time.Now()}
	case StateDisconnected:
		return Dimension{Status: HealthUnhealthy, Message: "disconnected", Since: time.Now()}
	case StateClosed:
		return Dimension{Status: HealthUnhealthy, Message: "closed", Since: time.Now()}
	default:
		return Dimension{Status: HealthDegraded, Message: "initializing", Since: time.Now()}
	}
}

func (m *SessionManager) healthTransport(status *AuthStatus, err error) Dimension {
	if err != nil {
		msg := err.Error()
		if isConnectionError(err) {
			return Dimension{Status: HealthUnhealthy, Message: "connection: " + msg, Since: time.Now()}
		}
		return Dimension{Status: HealthDegraded, Message: "transport: " + msg, Since: time.Now()}
	}
	if status != nil && !status.Connected {
		return Dimension{Status: HealthUnhealthy, Message: "gateway disconnected", Since: time.Now()}
	}
	return Dimension{Status: HealthHealthy, Message: "available", Since: time.Now()}
}

func (m *SessionManager) healthMarketData(status *AuthStatus, err error) Dimension {
	if err != nil {
		return Dimension{Status: HealthUnhealthy, Message: err.Error(), Since: time.Now()}
	}
	if !status.Authenticated {
		return Dimension{Status: HealthUnhealthy, Message: "not authenticated", Since: time.Now()}
	}
	if !status.Established {
		return Dimension{Status: HealthDegraded, Message: "session not fully established", Since: time.Now()}
	}
	return Dimension{Status: HealthHealthy, Message: "authorized", Since: time.Now()}
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "refused") ||
		strings.Contains(msg, "reset") ||
		strings.Contains(msg, "unreachable")
}

func worstOf(statuses ...HealthStatus) HealthStatus {
	worst := HealthUnknown
	for _, s := range statuses {
		if s > worst {
			worst = s
		}
	}
	return worst
}
