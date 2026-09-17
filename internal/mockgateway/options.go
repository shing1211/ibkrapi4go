// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import "time"

// options collects the resolved Server configuration.
type options struct {
	fixtures     *Fixtures
	recorder     *Recorder
	scenario     *Scenario
	latency      time.Duration
	seed         int64
	authRequired bool
	faults       FaultPolicy
	streamScript *StreamScript
}

// Option customizes a Server during construction.
type Option func(*options)

// WithFixtures selects the fixture registry. When unset, DefaultFixtures is
// used.
func WithFixtures(f *Fixtures) Option {
	return func(o *options) { o.fixtures = f }
}

// WithRecorder selects the request recorder. When unset, a new Recorder is
// created and exposed via Server.Recorder.
func WithRecorder(r *Recorder) Option {
	return func(o *options) { o.recorder = r }
}

// WithScenario selects the fault-injection scenario. When unset, a new Scenario
// is created.
func WithScenario(s *Scenario) Option {
	return func(o *options) { o.scenario = s }
}

// WithLatency applies a fixed delay to every response. When a fault also sets a
// delay, the effective delay is the larger of the two.
func WithLatency(d time.Duration) Option {
	return func(o *options) { o.latency = d }
}

// WithSeed sets the deterministic seed used for generated values.
func WithSeed(seed int64) Option {
	return func(o *options) { o.seed = seed }
}

// WithAuthRequired enables session enforcement: protected routes return 401
// until ssodh/init issues a session cookie or Authorization token.
func WithAuthRequired(v bool) Option {
	return func(o *options) { o.authRequired = v }
}

// WithFaultPolicy installs a FaultPolicy consulted before per-op and global
// faults.
func WithFaultPolicy(p FaultPolicy) Option {
	return func(o *options) { o.faults = p }
}

// WithStreamScript installs the scripted WebSocket streaming behavior served at
// GET /v1/api/ws. The script is value-copied at construction.
func WithStreamScript(s *StreamScript) Option {
	return func(o *options) { o.streamScript = s }
}
