// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import "context"

// TokenProvider is a test double for internal.TokenProvider. It returns a
// fixed token or delegates to the TokenFn callback. TokenFn takes precedence
// when non-nil.
type TokenProvider struct {
	// AccessToken is the fixed access token returned by Token when TokenFn is nil.
	AccessToken string
	// Err is the error returned by Token when TokenFn is nil and AccessToken is "".
	Err error
	// TokenFn, when set, is called by Token. It receives the context and
	// returns a token and error.
	TokenFn func(ctx context.Context) (string, error)
	// CallCount records the number of times Token was called.
	CallCount int
}

// Token satisfies internal.TokenProvider.
func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	p.CallCount++
	if p.TokenFn != nil {
		return p.TokenFn(ctx)
	}
	if p.AccessToken != "" {
		return p.AccessToken, nil
	}
	return "", p.Err
}
