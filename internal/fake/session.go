// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"context"

	"github.com/shing1211/ibkrapi4go/internal"
)

// SessionMachine is a test double for internal.SessionMachine. State and token
// are controlled via fields. Every method call is recorded in the corresponding
// call-count field.
type SessionMachine struct {
	// StateResult is the value returned by State.
	StateResult internal.SessionState
	// TokenResult is the token returned by Token.
	TokenResult string
	// TokenOK is the boolean returned by Token.
	TokenOK bool
	// InitErr is the error returned by Initialize.
	InitErr error
	// CloseErr is the error returned by Close.
	CloseErr error

	// StateCalls records the number of State calls.
	StateCalls int
	// TokenCalls records the number of Token calls.
	TokenCalls int
	// InitCalls records the number of Initialize calls.
	InitCalls int
	// CloseCalls records the number of Close calls.
	CloseCalls int
}

// State satisfies internal.SessionMachine.
func (s *SessionMachine) State() internal.SessionState {
	s.StateCalls++
	return s.StateResult
}

// Token satisfies internal.SessionMachine.
func (s *SessionMachine) Token() (string, bool) {
	s.TokenCalls++
	return s.TokenResult, s.TokenOK
}

// Initialize satisfies internal.SessionMachine.
func (s *SessionMachine) Initialize(_ context.Context) error {
	s.InitCalls++
	return s.InitErr
}

// Close satisfies internal.SessionMachine.
func (s *SessionMachine) Close(_ context.Context) error {
	s.CloseCalls++
	return s.CloseErr
}
