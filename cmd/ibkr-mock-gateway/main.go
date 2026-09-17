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
	"flag"
	"fmt"
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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ibkr-mock-gateway: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", ":5001", "listen address (host:port)")
	scenario := flag.String("scenario", "none", "fault scenario: none|expired|ratelimited|flaky")
	latency := flag.Duration("latency", 0, "fixed delay added to every response (e.g. 25ms)")
	seed := flag.Int64("seed", 1, "deterministic seed for generated values")
	useTLS := flag.Bool("tls", false, "serve TLS with an in-memory self-signed certificate for loopback")
	verbose := flag.Bool("verbose", false, "log every request")
	authRequired := flag.Bool("auth-required", false, "reject protected CPAPI routes until ssodh/init establishes a session")
	flag.Parse()

	if *latency < 0 {
		return fmt.Errorf("invalid -latency %s: must not be negative", *latency)
	}
	scn, err := buildScenario(*scenario)
	if err != nil {
		return err
	}

	opts := []mockgateway.Option{
		mockgateway.WithSeed(*seed),
		mockgateway.WithScenario(scn),
		mockgateway.WithAuthRequired(*authRequired),
	}
	if *latency > 0 {
		opts = append(opts, mockgateway.WithLatency(*latency))
	}
	mock := mockgateway.New(opts...)

	var handler http.Handler = mock.Handler()
	if *verbose {
		handler = requestLogger(handler)
	}

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	scheme := "http"
	if *useTLS {
		cert, err := selfSignedCert()
		if err != nil {
			return fmt.Errorf("generate self-signed certificate: %w", err)
		}
		httpSrv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		scheme = "https"
	}

	base := baseURL(*addr, scheme)
	printBanner(base, *seed, *scenario, *useTLS)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if *useTLS {
			errCh <- httpSrv.ListenAndServeTLS("", "")
			return
		}
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		stop()
		fmt.Fprintln(os.Stderr, "\nshutting down...")
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
func printBanner(base string, seed int64, scenario string, tlsOn bool) {
	wsBase := "ws" + strings.TrimPrefix(base, "http")
	fmt.Printf("ibkr-mock-gateway listening on %s (seed=%d, scenario=%s, tls=%t)\n", base, seed, scenario, tlsOn)
	fmt.Printf("  CPAPI            %s/v1/api\n", base)
	fmt.Printf("  CPAPI WebSocket  %s/v1/api/ws\n", wsBase)
	fmt.Printf("  IB REST          %s/gw/api/v1 and %s/gw/api/v2\n", base, base)
	fmt.Printf("  OAuth2 token     %s/oauth2/api/v1/token\n", base)
	fmt.Println()
	fmt.Println("Point the SDK at this gateway:")
	fmt.Printf("  export IBKR_GATEWAY_URL=%s\n", base)
	fmt.Printf("  export IBKR_REST_GATEWAY_URL=%s\n", base)
	if tlsOn {
		fmt.Println("  export IBKR_INSECURE_SKIP_VERIFY=true   # self-signed loopback certificate")
	}
	fmt.Println()
	fmt.Printf("Run the example:  IBKR_GATEWAY_URL=%s go run ./examples/mock\n", base)
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
