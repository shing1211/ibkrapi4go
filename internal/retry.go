// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryPolicy controls retries for safe (idempotent) requests. Unsafe methods
// and requests with a body are never retried (ADR 0009).
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts (including the first).
	MaxAttempts int
	// BaseDelay is the initial backoff.
	BaseDelay time.Duration
	// MaxDelay caps the backoff.
	MaxDelay time.Duration
	// Jitter applies full jitter to the backoff.
	Jitter bool
	// RetryOnStatus lists the HTTP statuses that trigger a retry.
	RetryOnStatus []int
	// Metrics receives retry observations. Nil is a valid default (no-op).
	Metrics Metrics
}

// DefaultRetryPolicy is the SDK default: 3 attempts for safe methods.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     200 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		Jitter:        true,
		RetryOnStatus: []int{429, 500, 502, 503, 504},
	}
}

func (p RetryPolicy) enabled() bool { return p.MaxAttempts > 1 }

func (p RetryPolicy) retryableStatus(code int) bool {
	for _, s := range p.RetryOnStatus {
		if s == code {
			return true
		}
	}
	return false
}

// delayFor returns the backoff before the next attempt, honoring Retry-After on
// 429 responses.
func (p RetryPolicy) delayFor(attempt int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return d
		}
	}
	base := p.BaseDelay
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	d := base << attempt
	if p.MaxDelay > 0 && d > p.MaxDelay {
		d = p.MaxDelay
	}
	if p.Jitter && d > 0 {
		d = time.Duration(rand.Int63n(int64(d) + 1))
	}
	return d
}

// Retry returns a middleware that retries safe requests per p.
func Retry(p RetryPolicy) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !p.enabled() || !isSafeMethod(req.Method) || req.Body != nil {
				return base.RoundTrip(req)
			}
			var lastResp *http.Response
			var lastErr error
			for attempt := 0; attempt < p.MaxAttempts; attempt++ {
				if attempt > 0 {
					d := p.delayFor(attempt-1, lastResp)
					observeHistogram(req.Context(), p.Metrics, MetricHTTPRetryBackoffMS, float64(d.Nanoseconds())/1e6,
						Attr{Key: "attempt", Value: strconv.Itoa(attempt)})
					if err := sleepCtx(req.Context(), d); err != nil {
						return lastResp, err
					}
					incrCounter(req.Context(), p.Metrics, MetricHTTPRetries, 1,
						Attr{Key: "attempt", Value: strconv.Itoa(attempt)})
				}
				resp, err := base.RoundTrip(req)
				lastResp, lastErr = resp, err
				if err != nil {
					if !retryableErr(req.Context(), err) {
						return resp, err
					}
					continue
				}
				if !p.retryableStatus(resp.StatusCode) {
					return resp, nil
				}
				if attempt < p.MaxAttempts-1 {
					resp.Body.Close()
				}
			}
			return lastResp, lastErr
		})
	}
}

// isSafeMethod reports whether an HTTP method is idempotent and safe to retry.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// retryableErr reports whether a transport error should be retried. Context
// cancellation/deadline and non-timeout protocol errors are not retried.
func retryableErr(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// sleepCtx waits for d or returns ctx.Err().
func sleepCtx(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// parseRetryAfter parses a Retry-After header value (seconds or HTTP date).
func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}
