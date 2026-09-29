// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// These drive the subcommands past client construction against a real
// internal/mockgateway, which is what subcommands_test.go deliberately stops
// short of. That file covers the part of the CLI that needs no gateway:
// validation, dispatch, help and the exit-status contract. Everything after
// `e.newClient()` was unexecuted - all 9 call sites, in accounts.go, orders.go,
// portfolio.go, positions.go and stream.go - because no test ever replaced the
// seam.
//
// The seam itself already existed and worked. It needed using, not building.

// newGatewayEnv returns an env plus the global-flag arguments that point a command
// at a mock gateway.
//
// The gateway URL is passed as `--gateway` rather than written into the config file,
// because that is the path the CLI is supposed to use and it was completely broken:
// parseGlobalFlags was only ever called as parseGlobalFlags(e.args), whose default
// branch returns at i=0 - argv[0] - so no global flag was read and
// `ibkr --gateway URL accounts` was rejected as an unknown command. Routing the
// tests through the config file instead would have hidden that behind a path that
// works, which is the whole reason this run exists.
//
// The config file is still isolated, because loadConfig falls back to
// $HOME/.ibkr/config.json and is called from two places - newClientFromArgs and
// mustAccount. Without this a developer's real account id leaks in and the test
// passes or fails depending on their machine. The account itself stays in the
// config: it is what mustAccount reads for the commands that take no -account.
func newGatewayEnv(t *testing.T, opts ...mockgateway.Option) (*env, *bytes.Buffer, *bytes.Buffer, []string) {
	t.Helper()

	srv := httptest.NewServer(mockgateway.New(opts...).Handler())
	t.Cleanup(srv.Close)

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"account_id":"U1234567"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("IBKR_CONFIG", cfgPath)

	var out, errOut bytes.Buffer
	e := newEnv([]string{"ibkr", "accounts"}, &out, &errOut)
	return e, &out, &errOut, []string{"--gateway", srv.URL}
}

// wantContains fails with both streams, because a CLI that prints an error to
// the wrong writer looks like a success when you only check the right one.
func wantContains(t *testing.T, what string, out, errOut *bytes.Buffer, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out.String(), w) {
			t.Errorf("%s: stdout does not contain %q\nstdout:\n%s\nstderr:\n%s", what, w, out.String(), errOut.String())
		}
	}
}

// withFlags prepends the global flags to a command's own arguments. The order
// matters only in that the flags have to come before the first non-flag token,
// because that is where parseGlobalFlags starts reading.
func withFlags(flags []string, args ...string) []string {
	return append(append([]string{}, flags...), args...)
}

func TestAccounts_ListsAccountsFromGateway(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	if err := runAccounts(e, gflags); err != nil {
		t.Fatalf("runAccounts = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	// The default fixture returns U1234567 (alias Main) and U7654321. The command
	// emits indented JSON, so this also pins that the encoder writes to e.stdout.
	wantContains(t, "accounts", out, errOut, "U1234567", "U7654321")
}

func TestPositions_PrintsTableFromGateway(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	if err := runPositions(e, withFlags(gflags, "-account", "U1234567")); err != nil {
		t.Fatalf("runPositions = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	// The fixture position is AAPL x10.5 at avgCost 145.25.
	wantContains(t, "positions", out, errOut, "ACCOUNT", "DESCRIPTION", "AAPL", "145.25", "10.5")
}

func TestPortfolio_SummaryLedgerAllocationFromGateway(t *testing.T) {
	cases := []struct {
		sub  string
		want []string
	}{
		// summary prints a header then one line per metric; netliquidation is 1575.00.
		{"summary", []string{"Portfolio Summary", "U1234567", "1575.00"}},
		// ledger and allocation are emitted as indented JSON. The ledger keys are
		// Go field names, not wire names: LedgerCurrency carries no json tags, so
		// unlike every other JSON-emitting command in the CLI this one is not
		// lowerCamelCase.
		{"ledger", []string{"NetLiquidationValue", "1575.00", "USD"}},
		{"allocation", []string{"STK", "1000.5"}},
	}
	for _, tc := range cases {
		t.Run(tc.sub, func(t *testing.T) {
			e, out, errOut, gflags := newGatewayEnv(t)
			// The subcommand name comes first: the documented form is
			// `ibkr <command> [subcommand] [flags]`, and runPortfolio switches on
			// args[0], so a leading --gateway would be taken as the subcommand.
			if err := runPortfolio(e, append([]string{tc.sub}, gflags...)); err != nil {
				t.Fatalf("runPortfolio %s = %v; want nil\nstderr:\n%s", tc.sub, err, errOut.String())
			}
			wantContains(t, tc.sub, out, errOut, tc.want...)
		})
	}
}

func TestOrders_ListFromGateway(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	if err := runOrdersList(e, gflags); err != nil {
		t.Fatalf("runOrdersList = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	// The fixture order is BUY LMT x10 @ 150.00 on U1234567, conid 265598. The
	// table has no ticker column, so AAPL is not expected here even though the
	// fixture carries it.
	wantContains(t, "orders list", out, errOut,
		"ORDER_ID", "999", "U1234567", "265598", "BUY", "LMT", "150.00", "PreSubmitted", "DAY")
}

func TestOrders_SubmitFromGateway(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	args := withFlags(gflags, "-conid", "265598", "-side", "buy", "-qty", "10", "-type", "LMT", "-price", "150.00")
	if err := runOrdersSubmit(e, args); err != nil {
		t.Fatalf("runOrdersSubmit = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	// The submit fixture returns order_id 999. The side is upper-cased by the flag
	// parser, so an unstamped "buy" would be a separate failure.
	wantContains(t, "orders submit", out, errOut, "999", "PreSubmitted")
}

// TestOrders_CancelWritesToTheInjectedStdout covers the one output path that
// bypasses the writer. Every other message in the CLI goes through e.stdout via
// ibkrPrintln or fmt.Fprintf, but the cancel confirmation used fmt.Printf, so it
// went to the process stdout instead. That made the message uncapturable and
// inconsistent with the env contract, and it is invisible to any consumer that
// captures the command's output.
func TestOrders_CancelWritesToTheInjectedStdout(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	if err := runOrdersCancel(e, withFlags(gflags, "-orderid", "999")); err != nil {
		t.Fatalf("runOrdersCancel = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "cancelled") {
		t.Errorf("cancel confirmation missing from the injected stdout\nstdout:\n%s\nstderr:\n%s",
			out.String(), errOut.String())
	}
}

func TestStream_SnapshotFromGateway(t *testing.T) {
	e, out, errOut, gflags := newGatewayEnv(t)
	if err := runStream(e, withFlags(gflags, "-conid", "265598", "-fields", "last,bid")); err != nil {
		t.Fatalf("runStream = %v; want nil\nstderr:\n%s", err, errOut.String())
	}
	// The snapshot fixture carries field 31 (last) at 150.25 for conid 265598.
	wantContains(t, "stream", out, errOut, "265598", "150.25")
}

// TestPortfolio_NoArgsDefaultsToSummaryWithoutPanicking covers the dispatch
// default. runOrders has the same shape and handles it correctly; runPortfolio
// did `args[1:]` inside its `len(args) == 0` branch, which is a slice bounds
// panic. `ibkr portfolio` with no arguments therefore crashed the process.
//
// The gateway URL goes in the config file here rather than in the arguments,
// because passing arguments at all would mean this is no longer the no-args path:
// the flags have to precede the subcommand name, and a non-empty args slice takes
// the switch branch instead.
//
// The recover is deliberate: a panic here would otherwise abort the whole test
// binary and take the other cases with it, so it is reported as an ordinary
// failure with the value that caused it.
func TestPortfolio_NoArgsDefaultsToSummaryWithoutPanicking(t *testing.T) {
	srv := httptest.NewServer(mockgateway.New().Handler())
	t.Cleanup(srv.Close)

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := fmt.Sprintf(`{"gateway_url":%q,"account_id":"U1234567"}`, srv.URL)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("IBKR_CONFIG", cfgPath)

	var out, errOut bytes.Buffer
	e := newEnv([]string{"ibkr", "portfolio"}, &out, &errOut)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		if err := runPortfolio(e, nil); err != nil {
			t.Fatalf("runPortfolio with no args = %v; want nil\nstderr:\n%s", err, errOut.String())
		}
	}()

	if recovered != nil {
		t.Fatalf("runPortfolio panicked with no args: %v", recovered)
	}
	wantContains(t, "portfolio default", &out, &errOut, "Portfolio Summary")
}

// TestSubcommands_GatewayUnreachable covers the failure every one of these paths
// shares: the client is constructed, then the request cannot be made. The error
// has to reach stderr and come back non-nil, or the command looks like it
// succeeded with no output.
func TestSubcommands_GatewayUnreachable(t *testing.T) {
	// A server bound and then closed immediately, so the port is not listening.
	// The Close before the test runs is the load-bearing part: leaving it up in a
	// cleanup would make every request hang until the CLI's 15s context deadline
	// instead of being refused, and 7 cases at 15s each is a slow suite.
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	gflags := []string{"--gateway", dead.URL}
	dead.Close()
	t.Cleanup(dead.Close) // idempotent

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"account_id":"U1234567"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("IBKR_CONFIG", cfgPath)

	cases := []struct {
		name string
		run  func(*env, []string) error
	}{
		{"accounts", func(e *env, a []string) error { return runAccounts(e, a) }},
		{"positions", func(e *env, a []string) error { return runPositions(e, a) }},
		{"portfolio summary", func(e *env, a []string) error {
			return runPortfolio(e, append([]string{"summary"}, a...))
		}},
		{"orders list", func(e *env, a []string) error { return runOrdersList(e, a) }},
		{"stream", func(e *env, a []string) error {
			// The global flags must precede -conid: parseGlobalFlags stops at the
			// first non-flag it does not own, so a trailing --gateway would never
			// be read and the command would hit the default gateway instead.
			return runStream(e, append(append([]string{}, a...), "-conid", "265598"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			e := newEnv([]string{"ibkr", "accounts"}, &out, &errOut)
			if err := tc.run(e, gflags); err == nil {
				t.Errorf("%s against a dead gateway returned nil; want an error", tc.name)
			}
			if !strings.Contains(errOut.String(), "error:") {
				t.Errorf("%s wrote no error to stderr; got %q", tc.name, errOut.String())
			}
		})
	}
}

// TestAccounts_UnauthorizedIsReportedNotSwallowed covers a gateway that answers
// with 401. The command must surface it rather than print an empty result and
// return nil, which is the failure mode that would let a caller believe the
// account list is genuinely empty.
func TestAccounts_UnauthorizedIsReportedNotSwallowed(t *testing.T) {
	scn := mockgateway.NewScenario()
	scn.SetPolicy(func(op string, _ *mockgateway.Request) *mockgateway.Fault {
		if op == mockgateway.OpGetBrokerageAccounts {
			return &mockgateway.Fault{Status: http.StatusUnauthorized, Body: `{"error":"not authenticated"}`}
		}
		return nil
	})

	e, _, errOut, gflags := newGatewayEnv(t, mockgateway.WithScenario(scn))
	if err := runAccounts(e, gflags); err == nil {
		t.Error("runAccounts against a 401 gateway returned nil; want an error")
	}
	if !strings.Contains(errOut.String(), "error:") {
		t.Errorf("no error written to stderr; got %q", errOut.String())
	}
}

// TestSubcommands_BadAccountIDIsRejectedBeforeAnyRequest pins that an unusable
// account id fails on the client side. mustAccount rejects an empty one, and the
// gateway must not be contacted at all in that case.
func TestSubcommands_BadAccountIDIsRejectedBeforeAnyRequest(t *testing.T) {
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "absent.json"))

	var out, errOut bytes.Buffer
	e := newEnv([]string{"ibkr"}, &out, &errOut)
	if err := runPositions(e, nil); err == nil {
		t.Error("runPositions with no account configured returned nil; want an error")
	}
	if out.Len() != 0 {
		t.Errorf("wrote %q to stdout before failing; want no output", out.String())
	}
}

// TestGlobalFlags_AccountReachesOrdersSubmit covers the one part of the global
// flag surface the happy-path tests do not: `-account` is both a declared global
// flag and a flag that positions, portfolio and orders each parse themselves.
//
// `orders submit` and `orders cancel` used to ignore it - both called
// mustAccount("") - so `ibkr orders submit --account U999 ...` was accepted,
// printed no complaint, and placed the order against whatever account_id the
// config file held, or failed with "account ID required" when the config had none.
//
// The observable is which error comes back. With no account in the config, the
// pre-fix code failed inside mustAccount; with the flag parsed it gets past that
// and fails later, on the request. The gateway is a closed local port so that
// failure is immediate rather than a 30s request timeout.
func TestGlobalFlags_AccountReachesOrdersSubmit(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()
	t.Cleanup(dead.Close) // idempotent

	// No account_id on purpose: that is what separates "flag was ignored" from
	// "flag was honoured".
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "absent.json"))

	cases := []struct {
		name string
		run  func(*env) error
	}{
		{"submit", func(e *env) error {
			return runOrdersSubmit(e, []string{
				"--gateway", deadURL, "-account", "U999",
				"-conid", "265598", "-side", "buy", "-qty", "1",
			})
		}},
		{"cancel", func(e *env) error {
			return runOrdersCancel(e, []string{
				"--gateway", deadURL, "-account", "U999", "-orderid", "999",
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			e := newEnv([]string{"ibkr", "orders", tc.name}, &out, &errOut)
			err := tc.run(e)
			if err == nil {
				t.Fatalf("%s returned nil against a closed gateway; want a request error", tc.name)
			}
			if strings.Contains(err.Error(), "account ID required") {
				t.Errorf("%s failed in mustAccount, so -account was ignored: %v", tc.name, err)
			}
		})
	}
}
