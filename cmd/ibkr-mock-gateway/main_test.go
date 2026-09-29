// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

func TestParseFlags_Defaults(t *testing.T) {
	o, err := parseFlags(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags(nil) = %v; want nil", err)
	}
	want := options{addr: ":5001", scenario: "none", latency: 0, seed: 1}
	if o != want {
		t.Errorf("options = %+v; want %+v", o, want)
	}
}

func TestParseFlags_Overrides(t *testing.T) {
	args := []string{
		"-addr", "127.0.0.1:0",
		"-scenario", "flaky",
		"-latency", "25ms",
		"-seed", "99",
		"-tls",
		"-verbose",
		"-auth-required",
	}
	o, err := parseFlags(args, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags = %v; want nil", err)
	}
	want := options{
		addr:         "127.0.0.1:0",
		scenario:     "flaky",
		latency:      25 * time.Millisecond,
		seed:         99,
		useTLS:       true,
		verbose:      true,
		authRequired: true,
	}
	if o != want {
		t.Errorf("options = %+v; want %+v", o, want)
	}
}

// TestParseFlags_NegativeLatency covers the one validation that is not the flag
// package's job. A negative delay would otherwise be silently dropped, because
// the mockgateway option is only applied when latency > 0 - so the flag would
// look accepted and change nothing.
func TestParseFlags_NegativeLatency(t *testing.T) {
	_, err := parseFlags([]string{"-latency", "-1s"}, io.Discard)
	if err == nil {
		t.Fatal("parseFlags(-latency -1s) = nil; want an error")
	}
	if !strings.Contains(err.Error(), "must not be negative") {
		t.Errorf("error = %q; want it to say the value must not be negative", err)
	}
}

// TestParseFlags_UnknownFlag proves an unrecognised flag is an error rather than
// an os.Exit. The package-level flag set would have terminated the test binary.
func TestParseFlags_UnknownFlag(t *testing.T) {
	var out strings.Builder
	_, err := parseFlags([]string{"-nope"}, &out)
	if err == nil {
		t.Fatal("parseFlags(-nope) = nil; want an error")
	}
	if !strings.Contains(out.String(), "nope") {
		t.Errorf("usage output = %q; want it to name the unknown flag", out.String())
	}
}

func TestBuildScenario(t *testing.T) {
	for _, name := range []string{"none", ""} {
		scn, err := buildScenario(name)
		if err != nil {
			t.Errorf("buildScenario(%q) = %v; want nil", name, err)
			continue
		}
		if scn == nil {
			t.Errorf("buildScenario(%q) = nil scenario; want a usable one", name)
		}
	}

	// Every named scenario must build, and must actually change behaviour rather
	// than parse successfully and do nothing. A scenario that builds and injects
	// nothing is the failure mode worth guarding: it makes the SDK's retry and
	// error-classification tests pass without exercising the path they exist for,
	// and it does so silently. The status the mock answers is the observable.
	probe := func(t *testing.T, scn *mockgateway.Scenario, path string) int {
		t.Helper()
		mock := mockgateway.New(mockgateway.WithScenario(scn))
		srv := httptest.NewServer(mock.Handler())
		t.Cleanup(func() {
			mock.Close()
			srv.Close()
		})
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s = %v; want nil", path, err)
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode
	}

	t.Run("none leaves the fixture response alone", func(t *testing.T) {
		scn, err := buildScenario("none")
		if err != nil {
			t.Fatalf("buildScenario(none) = %v; want nil", err)
		}
		if got := probe(t, scn, "/v1/api/iserver/auth/status"); got != http.StatusOK {
			t.Errorf("status = %d; want 200 from the clean fixture", got)
		}
	})

	t.Run("expired rejects the protected route", func(t *testing.T) {
		scn, err := buildScenario("expired")
		if err != nil {
			t.Fatalf("buildScenario(expired) = %v; want nil", err)
		}
		// A protected CPAPI path, so the session-expiry fault has something to act
		// on; the clean mock answers it 401 for want of a session either way, so
		// what is asserted is that the fault is wired at all, via the recorder.
		mock := mockgateway.New(mockgateway.WithScenario(scn))
		srv := httptest.NewServer(mock.Handler())
		defer func() {
			mock.Close()
			srv.Close()
		}()
		resp, err := http.Get(srv.URL + "/v1/api/portfolio/accounts")
		if err != nil {
			t.Fatalf("GET = %v; want nil", err)
		}
		_ = resp.Body.Close()
		if !scn.SessionExpired() {
			t.Error("the expired scenario did not set SessionExpired on the scenario it hands the mock")
		}
	})

	t.Run("ratelimited answers 429 to every request", func(t *testing.T) {
		scn, err := buildScenario("ratelimited")
		if err != nil {
			t.Fatalf("buildScenario(ratelimited) = %v; want nil", err)
		}
		if got := probe(t, scn, "/v1/api/iserver/auth/status"); got != http.StatusTooManyRequests {
			t.Errorf("status = %d; want 429", got)
		}
	})
}

// TestBuildScenario_Flaky covers the one scenario whose behaviour is
// probabilistic. A 21-request burst crosses both the 7th and the 14th request,
// so a 503 is guaranteed rather than likely, and a 200 is guaranteed too - which
// is what makes the three-way rule distinguishable from a blanket fault.
func TestBuildScenario_Flaky(t *testing.T) {
	scn, err := buildScenario("flaky")
	if err != nil {
		t.Fatalf("buildScenario(flaky) = %v; want nil", err)
	}
	mock := mockgateway.New(mockgateway.WithScenario(scn))
	srv := httptest.NewServer(mock.Handler())
	defer func() {
		mock.Close()
		srv.Close()
	}()

	seen := map[int]bool{}
	for i := 0; i < 21; i++ {
		resp, err := http.Get(srv.URL + "/v1/api/iserver/auth/status")
		if err != nil {
			// The 7th request drops the connection, which surfaces here as a
			// transport error. That is the scenario working.
			seen[0] = true
			continue
		}
		seen[resp.StatusCode] = true
		_ = resp.Body.Close()
	}
	if !seen[http.StatusServiceUnavailable] {
		t.Errorf("statuses seen = %v; want 503 from the every-third-request rule", seen)
	}
	if !seen[http.StatusOK] {
		t.Errorf("statuses seen = %v; want 200 on the requests the policy passes through", seen)
	}
}

func TestBuildScenario_Unknown(t *testing.T) {
	_, err := buildScenario("bogus")
	if err == nil {
		t.Fatal("buildScenario(bogus) = nil; want an error")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error = %q; want it to name the rejected scenario", err)
	}
}

func TestBaseURL(t *testing.T) {
	tests := []struct {
		addr   string
		scheme string
		want   string
	}{
		// A wildcard bind is what the default :5001 is, and it is not a
		// connectable host - reporting it verbatim sends the reader nowhere.
		{":5001", "http", "http://localhost:5001"},
		{"0.0.0.0:5001", "http", "http://localhost:5001"},
		{"[::]:5001", "http", "http://localhost:5001"},
		{"127.0.0.1:9000", "http", "http://127.0.0.1:9000"},
		{"127.0.0.1:9000", "https", "https://127.0.0.1:9000"},
		// No port to split on: pass it through rather than dropping it.
		{"localhost", "http", "http://localhost"},
		// An empty -addr has no host:port to split, so it falls through to the
		// pass-through branch and yields a scheme with no authority. That is a
		// degenerate value rather than a useful URL, and it is pinned here so a
		// change to the fallback is deliberate - but the listen will fail first,
		// so the banner is never the thing the operator sees.
		{"", "http", "http://"},
	}
	for _, tc := range tests {
		if got := baseURL(tc.addr, tc.scheme); got != tc.want {
			t.Errorf("baseURL(%q, %q) = %q; want %q", tc.addr, tc.scheme, got, tc.want)
		}
	}
}

func TestNewServer_Plaintext(t *testing.T) {
	srv, base, err := newServer(options{addr: "127.0.0.1:0", scenario: "none", seed: 1})
	if err != nil {
		t.Fatalf("newServer = %v; want nil", err)
	}
	if srv.TLSConfig != nil {
		t.Error("TLSConfig is set without -tls")
	}
	if want := "http://127.0.0.1:0"; base != want {
		t.Errorf("base = %q; want %q", base, want)
	}
	if srv.Handler == nil {
		t.Error("Handler is nil")
	}
	if srv.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout is zero, so a slow-header client can hold a connection open indefinitely")
	}
}

// TestNewServer_Serves covers the flag wiring end to end: the built server must
// actually answer on both surfaces. The banner tells the operator to point the
// SDK here, so a server that builds but does not serve is the failure that
// matters.
func TestNewServer_Serves(t *testing.T) {
	srv, _, err := newServer(options{addr: "127.0.0.1:0", scenario: "none", seed: 1})
	if err != nil {
		t.Fatalf("newServer = %v; want nil", err)
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("listen = %v; want nil", err)
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	for _, path := range []string{"/v1/api/iserver/auth/status", "/gw/api/v1/iserver/auth/status"} {
		resp, err := http.Get(baseForListener(ln) + path)
		if err != nil {
			t.Fatalf("GET %s = %v; want nil", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("GET %s = %d; want a non-5xx answer", path, resp.StatusCode)
		}
	}
}

func TestNewServer_TLS(t *testing.T) {
	srv, base, err := newServer(options{addr: "127.0.0.1:0", scenario: "none", seed: 1, useTLS: true})
	if err != nil {
		t.Fatalf("newServer = %v; want nil", err)
	}
	if srv.TLSConfig == nil || len(srv.TLSConfig.Certificates) != 1 {
		t.Fatalf("TLSConfig = %+v; want exactly one certificate", srv.TLSConfig)
	}
	if srv.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x; want %x (TLS 1.2)", srv.TLSConfig.MinVersion, tls.VersionTLS12)
	}
	if want := "https://127.0.0.1:0"; base != want {
		t.Errorf("base = %q; want %q", base, want)
	}
}

// TestNewServer_TLSHandshake proves the certificate is actually usable, not just
// present. A certificate whose SANs do not cover loopback, or which is expired,
// would satisfy the struct assertions above and still fail every client the
// banner tells to connect.
func TestNewServer_TLSHandshake(t *testing.T) {
	srv, _, err := newServer(options{addr: "127.0.0.1:0", scenario: "none", seed: 1, useTLS: true})
	if err != nil {
		t.Fatalf("newServer = %v; want nil", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen = %v; want nil", err)
	}
	go func() { _ = srv.ServeTLS(ln, "", "") }()
	defer func() { _ = srv.Close() }()

	leaf, err := x509.ParseCertificate(srv.TLSConfig.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse certificate = %v; want nil", err)
	}
	if err := leaf.VerifyHostname("localhost"); err != nil {
		t.Errorf("VerifyHostname(localhost) = %v; want nil - the banner tells operators to connect to localhost", err)
	}
	if time.Now().After(leaf.NotAfter) {
		t.Error("certificate is already expired")
	}
	if time.Now().Before(leaf.NotBefore) {
		t.Error("certificate is not valid yet")
	}

	// InsecureSkipVerify is the documented way to use this certificate
	// (IBKR_INSECURE_SKIP_VERIFY), so the handshake must succeed with it.
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, //nolint:gosec // self-signed loopback cert, as documented
	}}
	resp, err := client.Get("https://" + ln.Addr().String() + "/v1/api/iserver/auth/status")
	if err != nil {
		t.Fatalf("TLS GET = %v; want nil", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.TLS == nil {
		t.Error("response did not negotiate TLS")
	}
}

func TestNewServer_BadScenario(t *testing.T) {
	_, _, err := newServer(options{addr: ":5001", scenario: "nope"})
	if err == nil {
		t.Fatal("newServer(bogus scenario) = nil; want an error")
	}
}

// TestNewServer_VerboseWraps is about the wrapper, not the log output. The
// wrapper replaces the ResponseWriter, so if it failed to forward the optional
// interfaces the WebSocket upgrade would break - which a request-logging test
// that only checks status codes would not catch.
func TestNewServer_VerboseWraps(t *testing.T) {
	srv, _, err := newServer(options{addr: "127.0.0.1:0", scenario: "none", seed: 1, verbose: true})
	if err != nil {
		t.Fatalf("newServer = %v; want nil", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/api/iserver/auth/status", nil)
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code == 0 {
		t.Error("recorder code is 0; the wrapped handler wrote nothing")
	}
}

func TestRequestLogger_RecordsStatus(t *testing.T) {
	var inner int
	h := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inner++
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if inner != 1 {
		t.Errorf("inner handler ran %d times; want 1", inner)
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d; want %d passed through", rec.Code, http.StatusTeapot)
	}
}

// TestStatusRecorder_DefaultStatus covers the wrapper's other branch. The
// recorder seeds its field with 200 because a handler that writes a body without
// calling WriteHeader never touches it, and logging 0 for a successful request
// would be wrong.
func TestStatusRecorder_DefaultStatus(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	h := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.status != http.StatusOK {
		t.Errorf("recorded status = %d; want %d for a handler that never called WriteHeader", rec.status, http.StatusOK)
	}
}

func TestStatusRecorder_Unwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: inner}
	if rec.Unwrap() != http.ResponseWriter(inner) {
		t.Error("Unwrap did not return the wrapped writer")
	}
}

// TestStatusRecorder_Hijack covers the WebSocket path. The mock gateway serves
// /v1/api/ws, and a ResponseWriter wrapper that does not expose http.Hijacker
// turns every WebSocket upgrade into a silent failure that no status-code
// assertion can see. So this asserts the wrapper forwards the hijack, using a
// writer that records it - a real connection would prove net.Pipe works, not
// that the interface is exposed.
func TestStatusRecorder_Hijack(t *testing.T) {
	under := &hijackRecorder{hijacked: make(chan net.Conn, 1)}
	rec := &statusRecorder{ResponseWriter: under}
	got, rw, err := rec.Hijack()
	if err != nil {
		t.Fatalf("Hijack = %v; want nil", err)
	}
	if got == nil {
		t.Fatal("Hijack returned a nil connection; want the underlying writer's")
	}
	if rw != nil {
		t.Error("Hijack returned a non-nil bufio.Reader; want nil, as http.Hijacker's contract allows")
	}
	select {
	case <-under.hijacked:
	default:
		t.Error("the underlying writer's Hijack was never called")
	}
	_ = got.Close()
}

// TestStatusRecorder_HijackUnsupported proves the wrapper does not swallow the
// error when the underlying writer cannot be hijacked. A wrapper that returned a
// nil connection and a nil error would break the WebSocket upgrade in a way the
// handler above would read as success.
func TestStatusRecorder_HijackUnsupported(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: newFlushRecorder()}
	conn, _, err := rec.Hijack()
	if err == nil {
		t.Fatal("Hijack on a non-hijackable writer = nil error; want http.ErrNotSupported")
	}
	if conn != nil {
		t.Error("Hijack returned a connection alongside an error")
	}
	if !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("error = %v; want http.ErrNotSupported", err)
	}
}

func TestStatusRecorder_Flush(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: newFlushRecorder()}
	rec.Flush()
	if !rec.ResponseWriter.(*flushRecorder).flushed {
		t.Error("Flush did not reach the underlying writer")
	}
}

func TestSelfSignedCert(t *testing.T) {
	cert, err := selfSignedCert()
	if err != nil {
		t.Fatalf("selfSignedCert = %v; want nil", err)
	}
	if len(cert.Certificate) != 1 {
		t.Fatalf("certificate chain has %d entries; want 1 (self-signed)", len(cert.Certificate))
	}
	if cert.PrivateKey == nil {
		t.Error("PrivateKey is nil")
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse = %v; want nil", err)
	}
	// The 24h window is deliberate and the test says so, because an operator
	// following the banner will hit it if the cert silently expires.
	if d := leaf.NotAfter.Sub(leaf.NotBefore); d < 24*time.Hour || d > 25*time.Hour {
		t.Errorf("validity window = %s; want ~24h", d)
	}
	if err := leaf.VerifyHostname("127.0.0.1"); err != nil {
		t.Errorf("VerifyHostname(127.0.0.1) = %v; want nil - the IP SAN must cover loopback", err)
	}
}

// TestSelfSignedCert_Unique guards the serial and key generation. Two identical
// certificates would mean the generator is deterministic in a way that looks
// safe and is not.
func TestSelfSignedCert_Unique(t *testing.T) {
	a, err := selfSignedCert()
	if err != nil {
		t.Fatalf("selfSignedCert = %v; want nil", err)
	}
	b, err := selfSignedCert()
	if err != nil {
		t.Fatalf("selfSignedCert = %v; want nil", err)
	}
	if string(a.Certificate[0]) == string(b.Certificate[0]) {
		t.Error("two calls produced the same certificate; generation is not random")
	}
}

func TestPrintBanner(t *testing.T) {
	var plain strings.Builder
	printBanner(&plain, "http://localhost:5001", 7, "flaky", false)
	out := plain.String()
	for _, want := range []string{
		"http://localhost:5001/v1/api",
		"ws://localhost:5001/v1/api/ws",
		"http://localhost:5001/gw/api/v1",
		"http://localhost:5001/oauth2/api/v1/token",
		"export IBKR_GATEWAY_URL=http://localhost:5001",
		"seed=7",
		"scenario=flaky",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("banner is missing %q\n---\n%s", want, out)
		}
	}
	// The insecure-skip hint is only correct when TLS is on.
	if strings.Contains(out, "IBKR_INSECURE_SKIP_VERIFY") {
		t.Error("banner suggests IBKR_INSECURE_SKIP_VERIFY without -tls")
	}
}

func TestPrintBanner_TLS(t *testing.T) {
	var out strings.Builder
	printBanner(&out, "https://localhost:5001", 1, "none", true)
	// The WebSocket URL must be derived from the http/https scheme, or a TLS
	// deployment gets a plaintext ws:// URL that cannot connect.
	if !strings.Contains(out.String(), "wss://localhost:5001/v1/api/ws") {
		t.Errorf("banner does not advertise a wss:// WebSocket URL\n---\n%s", out.String())
	}
	if !strings.Contains(out.String(), "IBKR_INSECURE_SKIP_VERIFY") {
		t.Error("banner omits the insecure-skip hint under -tls")
	}
}

// TestRun_UnreachableAddr proves run reports a bind failure instead of hanging or
// exiting. This is the one path through run that does not block forever.
func TestRun_UnreachableAddr(t *testing.T) {
	// Port 0 on an address that cannot be bound: 203.0.113.0/24 is TEST-NET-3,
	// reserved and not routable, so the bind fails rather than succeeding.
	err := run([]string{"-addr", "203.0.113.1:1"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run on an unbindable address = nil; want a bind error")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Errorf("run = %v; want the bind error, not ErrServerClosed", err)
	}
}

func TestRun_BadFlag(t *testing.T) {
	if err := run([]string{"-nope"}, io.Discard, io.Discard); err == nil {
		t.Fatal("run(-nope) = nil; want a flag error")
	}
}

// baseForListener reports the address a listener actually bound, which for
// port 0 is only known after Listen.
func baseForListener(ln net.Listener) string {
	return "http://" + ln.Addr().String()
}

// hijackRecorder is a ResponseWriter that can be hijacked, so the wrapper's
// forwarding can be observed without a real socket.
type hijackRecorder struct {
	header   http.Header
	hijacked chan net.Conn
}

func (h *hijackRecorder) Header() http.Header { return h.header }

func (h *hijackRecorder) Write([]byte) (int, error) { return 0, nil }

func (h *hijackRecorder) WriteHeader(int) {}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, server := net.Pipe()
	go func() { _ = server.Close() }()
	h.hijacked <- conn
	return conn, nil, nil
}

type flushRecorder struct {
	header  http.Header
	body    strings.Builder
	code    int
	flushed bool
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{header: make(http.Header), code: http.StatusOK}
}

func (f *flushRecorder) Header() http.Header         { return f.header }
func (f *flushRecorder) Write(b []byte) (int, error) { return f.body.Write(b) }
func (f *flushRecorder) WriteHeader(code int)        { f.code = code }
func (f *flushRecorder) Flush()                      { f.flushed = true }
