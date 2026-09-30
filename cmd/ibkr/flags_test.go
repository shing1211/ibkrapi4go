// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		gateway  string
		account  string
		insecure bool
		cmdIdx   int
	}{
		{
			name:   "no flags, command first",
			args:   []string{"accounts"},
			cmdIdx: 1,
		},
		{
			name:     "all flags before the command",
			args:     []string{"-gateway", "https://g:1", "-account", "U1", "-insecure", "positions"},
			gateway:  "https://g:1",
			account:  "U1",
			insecure: true,
			// cmdIdx is one past the command, i.e. where that command's own
			// arguments begin: "positions" is args[5], so its args start at 6.
			cmdIdx: 6,
		},
		{
			name:    "long flag spellings",
			args:    []string{"--gateway", "https://g:1", "orders"},
			gateway: "https://g:1",
			cmdIdx:  3,
		},
		{
			// A value-taking flag with nothing after it is dropped rather than
			// panicking or swallowing the next argument.
			name:   "flag with a missing value is ignored",
			args:   []string{"-gateway"},
			cmdIdx: 1,
		},
		{
			name:   "no command at all",
			args:   nil,
			cmdIdx: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotGateway, gotAccount, gotInsecure, gotIdx :=
				parseGlobalFlags(tc.args)
			if gotGateway != tc.gateway {
				t.Errorf("gateway = %q; want %q", gotGateway, tc.gateway)
			}
			if gotAccount != tc.account {
				t.Errorf("account = %q; want %q", gotAccount, tc.account)
			}
			if gotInsecure != tc.insecure {
				t.Errorf("insecure = %t; want %t", gotInsecure, tc.insecure)
			}
			if gotIdx != tc.cmdIdx {
				t.Errorf("cmdIdx = %d; want %d", gotIdx, tc.cmdIdx)
			}
		})
	}
}

// TestParseGlobalFlags_StopsAtFirstNonFlag documents that flag parsing ends at
// the first non-flag argument, so a flag placed after the command belongs to
// that command and is not consumed as a global one.
func TestParseGlobalFlags_StopsAtFirstNonFlag(t *testing.T) {
	gateway, _, _, idx := parseGlobalFlags([]string{"-insecure", "orders", "-gateway", "https://ignored"})
	if gateway != "" {
		t.Errorf("gateway = %q; want empty - the flag after the command belongs to the subcommand", gateway)
	}
	if idx != 2 {
		t.Errorf("cmdIdx = %d; want 2 (the index of \"orders\")", idx)
	}
}

// TestParseGlobalFlags_NoCommandReturnsLen is the one case where the returned
// index is not one past a command: with no non-flag in args there is no command,
// and the index is len(args) - the position a command would have to be inserted
// at. It is documented on the function and has no production caller, so it is
// pinned here rather than left to the table above, whose "no command at all" row
// uses nil args and so only covers the zero case.
func TestParseGlobalFlags_NoCommandReturnsLen(t *testing.T) {
	args := []string{"-insecure", "-gateway", "https://g:1"}
	_, _, _, idx := parseGlobalFlags(args)
	if idx != len(args) {
		t.Errorf("cmdIdx = %d; want %d (len(args))", idx, len(args))
	}
}

// TestParseGlobalFlags_RepeatedFlag pins that the last occurrence wins, so a
// change to that behaviour is deliberate.
func TestParseGlobalFlags_RepeatedFlag(t *testing.T) {
	gateway, _, _, _ := parseGlobalFlags(
		[]string{"-gateway", "https://first", "-gateway", "https://second", "accounts"})
	if want := "https://second"; gateway != want {
		t.Errorf("gateway = %q; want %q (last occurrence wins)", gateway, want)
	}
}

// TestUsageListsEveryRoutableCommand guards against a command being added to the
// dispatch switch but not to the usage text, or listed but unroutable. A command
// that is documented but unreachable is a help entry that always errors.
func TestUsageListsEveryRoutableCommand(t *testing.T) {
	var out, errOut strings.Builder
	if err := run([]string{"ibkr", "help"}, &out, &errOut); err != nil {
		t.Fatalf("run help = %v; want nil", err)
	}
	usage := errOut.String()
	for _, cmd := range []string{
		"accounts", "positions", "orders", "stream", "portfolio", "config", "completion",
	} {
		if !strings.Contains(usage, cmd) {
			t.Errorf("command %q is routable but absent from the usage text", cmd)
		}
	}
}
