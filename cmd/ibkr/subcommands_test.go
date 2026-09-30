// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// These exercise the six subcommands that previously had no reachable seam. Each
// one used to read os.Args and write to os.Stdout directly, and `orders` and
// `portfolio` rewrote os.Args in place to hand a subcommand's arguments down - so
// they could not be tested at all. They take an explicit env now.
//
// The commands that need a live gateway are not driven here; what is covered is
// the part that does not need one: argument validation, subcommand dispatch, help
// handling, and the exit-status contract. That is where a CLI actually goes wrong.

func newTestEnv(args ...string) (*env, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return newEnv(args, &out, &errOut), &out, &errOut
}

// TestSubcommands_HelpFlags covers the -h/--help path in each leaf command. A
// help request must not require a gateway: that is what makes the CLI usable when
// the gateway is down.
func TestSubcommands_HelpFlags(t *testing.T) {
	cases := []struct {
		name string
		run  func(*env) error
		env  func(*env, []string) *env
	}{
		{"accounts", func(e *env) error { return runAccounts(e, []string{"-h"}) }, nil},
		{"positions", func(e *env) error { return runPositions(e, []string{"-h"}) }, nil},
		{"stream", func(e *env) error { return runStream(e, []string{"-h"}) }, nil},
		{"config show", func(e *env) error { return runConfig(e, []string{"-h"}) }, nil},
		{"portfolio", func(e *env) error { return runPortfolio(e, []string{"-h"}) }, nil},
		{"orders", func(e *env) error { return runOrders(e, []string{"-h"}) }, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, _, errOut := newTestEnv("ibkr", "x")
			if err := tc.run(e); err != nil {
				t.Fatalf("help returned %v; want nil", err)
			}
			if !strings.Contains(errOut.String(), "Usage:") {
				t.Errorf("no usage written to stderr; got %q", errOut.String())
			}
		})
	}
}

// TestSubcommands_UnknownSubcommandIsAnError covers the dispatch contract: a typo
// must be reported, not silently treated as the default subcommand.
func TestSubcommands_UnknownSubcommandIsAnError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		run       func(*env) error
		contained string
	}{
		{"orders", func(e *env) error { return runOrders(e, []string{"lst"}) },
			`unknown orders subcommand "lst"`},
		{"portfolio", func(e *env) error { return runPortfolio(e, []string{"nope"}) },
			`unknown portfolio subcommand "nope"`},
		{"config", func(e *env) error { return runConfig(e, []string{"shw"}) },
			`unknown config subcommand "shw"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _, _ := newTestEnv("ibkr", "x")
			err := tc.run(e)
			if err == nil {
				t.Fatalf("unknown subcommand returned nil; want an error")
			}
			if !strings.Contains(err.Error(), tc.contained) {
				t.Errorf("err = %q; want it to contain %q", err, tc.contained)
			}
		})
	}
}

// TestRun_DispatchesEveryCommand proves the dispatch table routes to a real
// implementation rather than falling through to "unknown command". Each command is
// invoked with a help flag so no gateway is needed, and the assertion is that the
// command printed its own usage - which the unknown-command path does not.
func TestRun_DispatchesEveryCommand(t *testing.T) {
	for _, cmd := range []string{"accounts", "positions", "stream", "orders", "portfolio", "config"} {
		t.Run(cmd, func(t *testing.T) {
			e, _, errOut := newTestEnv("ibkr", cmd, "-h")
			// route through the same dispatch the process uses
			if err := dispatch(e, cmd); err != nil {
				t.Fatalf("dispatch %s: %v", cmd, err)
			}
			if !strings.Contains(errOut.String(), "Usage: ibkr "+cmd) {
				t.Errorf("dispatch %s wrote %q; want its own usage", cmd, errOut.String())
			}
		})
	}
}

// TestOrders_ArgsAreNotLeakedBetweenInvocations guards the invariant that replaced
// the argv-aliasing bug: the CLI never writes to the process argument vector.
//
// The original `runOrders` handed a subcommand's arguments down with
// `os.Args = append(os.Args[:2], args[1:]...)`. That appends into the backing
// array os.Args itself points at, so dispatching a command mutated the real argv -
// which is also what the test harness reads, and the reason these commands could
// not be tested at all.
//
// Checking each env's own copy would prove nothing, since a copy is trivially
// unmodified. The assertion is on os.Args itself, before and after, across two
// invocations with different arguments.
func TestOrders_ArgsAreNotLeakedBetweenInvocations(t *testing.T) {
	before := strings.Join(os.Args, "\x00")

	first := []string{"ibkr", "orders", "submit", "-conid", "1", "-side", "BUY"}
	e1, _, _ := newTestEnv(first...)
	_ = runOrders(e1, first[2:])

	second := []string{"ibkr", "orders", "cancel", "-orderid", "7"}
	e2, _, _ := newTestEnv(second...)
	_ = runOrders(e2, second[2:])

	if after := strings.Join(os.Args, "\x00"); after != before {
		t.Errorf("the CLI mutated the process argv.\nbefore: %q\nafter:  %q", before, after)
	}
}

// TestArgParsing_ConsecutiveFlagsBothApply covers the skip-the-value step. Each
// flag parser advances its own index past the value it consumed, so that value is
// never re-read as a flag. Writing those loops as `for i, a := range args` compiles
// and looks equivalent, but assigning to a range variable does not advance the
// iteration, so the parser stops at the first value and silently ignores every
// flag after it. Two flags in one command is the smallest case that tells the
// two forms apart.
func TestArgParsing_ConsecutiveFlagsBothApply(t *testing.T) {
	t.Run("global", func(t *testing.T) {
		gateway, account, _, cmdIdx := parseGlobalFlags(
			[]string{"-gateway", "g", "-account", "a", "positions"})
		if gateway != "g" || account != "a" {
			t.Errorf("got gateway=%q account=%q; want g, a", gateway, account)
		}
		if cmdIdx != 5 {
			t.Errorf("command index = %d; want 5 (both flags consumed)", cmdIdx)
		}
	})

	t.Run("positions", func(t *testing.T) {
		// parseAccountFlag is the same shape as the others and needs no account
		// resolution, so it can be checked without a gateway.
		account, err := parseAccountFlag([]string{"-account", "U123", "-account", "U456"})
		if err != nil {
			t.Fatalf("parseAccountFlag: %v", err)
		}
		if account != "U456" {
			t.Errorf("account = %q; want U456 (the last flag must win)", account)
		}
	})
}

func TestArgValue_BoundsChecked(t *testing.T) {
	args := []string{"-account"}
	if v, ok := argValue(args, 0); ok || v != "" {
		t.Errorf("argValue(%q, 0) = %q, %v; want \"\", false - a flag with no value", args, v, ok)
	}
	if v, ok := argValue(args, 1); ok || v != "" {
		t.Errorf("argValue(%q, 1) = %q, %v; want \"\", false", args, v, ok)
	}
	if v, ok := argValue([]string{"-account", "U1"}, 0); !ok || v != "U1" {
		t.Errorf("argValue = %q, %v; want U1, true", v, ok)
	}
	if v := argAt(args, 5); v != "" {
		t.Errorf("argAt(%q, 5) = %q; want \"\"", args, v)
	}
}

// TestOrders_SubcommandArgsReachTheValidator proves a subcommand's own flags are
// parsed rather than discarded. Before the refactor the parent rewrote os.Args and
// the child read os.Args[2:]; a mistake in that hand-off showed up as a flag
// silently ignored, and here as a missing required flag.
func TestOrders_SubcommandArgsReachTheValidator(t *testing.T) {
	e, _, _ := newTestEnv("ibkr", "orders", "submit", "-conid", "1", "-side", "BUY")
	err := runOrders(e, []string{"submit", "-conid", "1", "-side", "BUY"})
	if err == nil {
		t.Fatal("submit with no -qty returned nil; want the validator to reject it")
	}
	if !strings.Contains(err.Error(), "qty") {
		t.Errorf("err = %v; want it to name the missing -qty flag", err)
	}
}

// TestEnv_ArgIsSafeOnShortArgv covers the guard that replaced direct os.Args
// indexing. A test invoking `ibkr accounts` passes a two-element argv, and
// `os.Args[2:]` on that would panic and take the process down.
func TestEnv_ArgIsSafeOnShortArgv(t *testing.T) {
	e, _, _ := newTestEnv("ibkr", "accounts")
	for _, n := range []int{0, 1, 2, 3, 99} {
		if got := e.arg(n); n >= len(e.args) && got != nil {
			t.Errorf("arg(%d) = %v; want nil for an out-of-range index", n, got)
		}
	}
	if got := e.arg(2); got != nil {
		t.Errorf("arg(2) on a 2-element argv = %v; want nil", got)
	}
	if got := e.arg(1); len(got) != 1 || got[0] != "accounts" {
		t.Errorf("arg(1) = %v; want [accounts]", got)
	}
}

// TestParseAccountFlag covers the account override, which used to read the
// os.Args its parent had rewritten.
func TestParseAccountFlag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{"none", nil, "", false},
		{"short flag", []string{"-account", "U999"}, "U999", false},
		{"long flag", []string{"--account", "U888"}, "U888", false},
		{"help returns empty", []string{"-h"}, "", false},
		{"flag with no value", []string{"-account"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAccountFlag(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v; wantErr %t", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("account = %q; want %q", got, tc.want)
			}
		})
	}
}
