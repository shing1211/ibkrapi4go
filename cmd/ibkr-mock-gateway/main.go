// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command ibkr-mock-gateway runs the in-repo mock IBKR gateway as a standalone
// HTTP/WebSocket server. It serves both API surfaces on a single port:
//
//	CPAPI            /v1/api/*
//	CPAPI WebSocket  /v1/api/ws
//	IB REST          /gw/api/v1/* and /gw/api/v2/*
//	OAuth2 token     /oauth2/api/v1/token
//
// Point the SDK at it with IBKR_GATEWAY_URL (CPAPI) and IBKR_REST_GATEWAY_URL
// (IB REST). The ready banner printed at startup shows the exact values.
package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// shutdownTimeout bounds how long a graceful shutdown waits for in-flight
// requests.
const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "ibkr-mock-gateway: %v\n", err)
		os.Exit(1)
	}
}

// options is the parsed command line. Each field is a flag the operator can set,
// and the defaults are the ones the program has always used.
type options struct {
	addr         string
	scenario     string
	latency      time.Duration
	seed         int64
	useTLS       bool
	verbose      bool
	authRequired bool
}

// parseFlags reads options from args.
//
// It builds its own FlagSet rather than using the package-level flag functions.
// Those register against the global CommandLine and flag.Parse reads os.Args, so
// a test could not supply a different command line without mutating process
// state - which is the same coupling that made cmd/ibkr untestable, documented in
// cmd/ibkr/env.go. A local FlagSet takes its arguments as a parameter, so the
// flag surface is reachable from a test. ContinueOnError means a bad flag
// returns an error instead of calling os.Exit.
func parseFlags(args []string, out io.Writer) (options, error) {
	fs := flag.NewFlagSet("ibkr-mock-gateway", flag.ContinueOnError)
	fs.SetOutput(out)

	var o options
	fs.StringVar(&o.addr, "addr", ":5001", "listen address (host:port)")
	fs.StringVar(&o.scenario, "scenario", "none", "fault scenario: none|expired|ratelimited|flaky")
	fs.DurationVar(&o.latency, "latency", 0, "fixed delay added to every response (e.g. 25ms)")
	fs.Int64Var(&o.seed, "seed", 1, "deterministic seed for generated values")
	fs.BoolVar(&o.useTLS, "tls", false, "serve TLS with an in-memory self-signed certificate for loopback")
	fs.BoolVar(&o.verbose, "verbose", false, "log every request")
	fs.BoolVar(&o.authRequired, "auth-required", false, "reject protected CPAPI routes until ssodh/init establishes a session")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if o.latency < 0 {
		return options{}, fmt.Errorf("invalid -latency %s: must not be negative", o.latency)
	}
	return o, nil
}

// newServer builds the HTTP server for the given options, without serving it.
//
// Splitting this out of run is what makes the flag wiring testable: the parts
// worth asserting on - which mockgateway options are derived, whether -verbose
// wraps the handler, whether -tls produced a TLS config and an https base URL -
// are all decided here, and none of them require a listening socket.
func newServer(o options) (*http.Server, string, error) {
	scn, err := buildScenario(o.scenario)
	if err != nil {
		return nil, "", err
	}

	opts := []mockgateway.Option{
		mockgateway.WithSeed(o.seed),
		mockgateway.WithScenario(scn),
		mockgateway.WithAuthRequired(o.authRequired),
	}
	if o.latency > 0 {
		opts = append(opts, mockgateway.WithLatency(o.latency))
	}
	mock := mockgateway.New(opts...)

	var handler = mock.Handler()
	if o.verbose {
		handler = requestLogger(handler)
	}

	httpSrv := &http.Server{
		Addr:              o.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	scheme := "http"
	if o.useTLS {
		cert, err := selfSignedCert()
		if err != nil {
			return nil, "", fmt.Errorf("generate self-signed certificate: %w", err)
		}
		// MinVersion is set explicitly. Go's zero value already defaults a
		// server to TLS 1.2, but relying on that is implicit and gosec (G402)
		// cannot see it; stating it keeps the floor obvious and the lint honest.
		httpSrv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		scheme = "https"
	}

	return httpSrv, baseURL(o.addr, scheme), nil
}

func run(args []string, stdout, stderr io.Writer) error {
	o, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	httpSrv, base, err := newServer(o)
	if err != nil {
		return err
	}
	printBanner(stdout, base, o.seed, o.scenario, o.useTLS)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if o.useTLS {
			errCh <- httpSrv.ListenAndServeTLS("", "")
			return
		}
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		stop()
		_, _ = fmt.Fprintln(stderr, "\nshutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}

// buildScenario maps the -scenario flag to a mockgateway.Scenario.
func buildScenario(name string) (*mockgateway.Scenario, error) {
	scn := mockgateway.NewScenario()
	switch name {
	case "none", "":
		return scn, nil
	case "expired":
		scn.SetSessionExpired(true)
		return scn, nil
	case "ratelimited":
		scn.SetGlobal(&mockgateway.Fault{
			Status: http.StatusTooManyRequests,
			Body:   `{"error":"too many requests"}`,
		})
		return scn, nil
	case "flaky":
		var n atomic.Int64
		scn.SetPolicy(func(string, *mockgateway.Request) *mockgateway.Fault {
			switch i := n.Add(1); {
			case i%7 == 0:
				return &mockgateway.Fault{DropConnection: true}
			case i%3 == 0:
				return &mockgateway.Fault{
					Status: http.StatusServiceUnavailable,
					Body:   `{"error":"transient fault"}`,
				}
			default:
				return nil
			}
		})
		return scn, nil
	default:
		return nil, fmt.Errorf("invalid -scenario %q (want none, expired, ratelimited, or flaky)", name)
	}
}

// baseURL converts a listen address into a client-facing base URL. A wildcard
// or empty host is reported as localhost.
func baseURL(addr, scheme string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return scheme + "://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

// printBanner writes the exposed surfaces and example environment variables.
//
// It takes a writer rather than printing to the process stdout, so a test can
// assert on the URLs the operator is told to point the SDK at. Those are the
// whole point of the banner: a wrong scheme or a dropped port sends the reader to
// a URL that does not serve anything.
func printBanner(w io.Writer, base string, seed int64, scenario string, tlsOn bool) {
	wsBase := "ws" + strings.TrimPrefix(base, "http")
	// The write errors are discarded rather than returned. A banner is written
	// once at startup to a terminal or a pipe: if the operator's stdout is closed
	// there is nothing useful left to do, and the server is still worth starting.
	// This is the same `_ =` convention the rest of the CLI uses for its messages.
	_, _ = fmt.Fprintf(w, "ibkr-mock-gateway listening on %s (seed=%d, scenario=%s, tls=%t)\n", base, seed, scenario, tlsOn)
	_, _ = fmt.Fprintf(w, "  CPAPI            %s/v1/api\n", base)
	_, _ = fmt.Fprintf(w, "  CPAPI WebSocket  %s/v1/api/ws\n", wsBase)
	_, _ = fmt.Fprintf(w, "  IB REST          %s/gw/api/v1 and %s/gw/api/v2\n", base, base)
	_, _ = fmt.Fprintf(w, "  OAuth2 token     %s/oauth2/api/v1/token\n", base)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Point the SDK at this gateway:")
	_, _ = fmt.Fprintf(w, "  export IBKR_GATEWAY_URL=%s\n", base)
	_, _ = fmt.Fprintf(w, "  export IBKR_REST_GATEWAY_URL=%s\n", base)
	if tlsOn {
		_, _ = fmt.Fprintln(w, "  export IBKR_INSECURE_SKIP_VERIFY=true   # self-signed loopback certificate")
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Run the example:  IBKR_GATEWAY_URL=%s go run ./examples/mock\n", base)
}

// requestLogger logs one line per request. The wrapper preserves http.Hijacker
// so WebSocket upgrades keep working.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("mock request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).String(),
		)
	})
}

// statusRecorder captures the response status and forwards the optional
// interfaces http.Hijacker and http.Flusher to the underlying writer.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(r.ResponseWriter).Hijack()
}

func (r *statusRecorder) Flush() {
	_ = http.NewResponseController(r.ResponseWriter).Flush()
}

// selfSignedCert builds an in-memory ECDSA certificate valid for loopback
// hostnames. It never touches the filesystem.
func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{Organization: []string{"ibkr-mock-gateway"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}, nil
}
