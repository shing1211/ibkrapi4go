// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRun_NoArgsPrintsUsage covers the bare invocation: no command, no error.
func TestRun_NoArgsPrintsUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"ibkr"}, &out, &errOut); err != nil {
		t.Fatalf("run with no args = %v; want nil", err)
	}
	if !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("usage not written to stderr; got %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q; want nothing", out.String())
	}
}

// TestRun_Version pins the version output, including that it goes to stdout and
// not stderr.
func TestRun_Version(t *testing.T) {
	for _, flag := range []string{"--version", "-v", "version"} {
		var out, errOut bytes.Buffer
		if err := run([]string{"ibkr", flag}, &out, &errOut); err != nil {
			t.Fatalf("run %s = %v; want nil", flag, err)
		}
		if got, want := out.String(), "ibkr "+Version+"\n"; got != want {
			t.Errorf("run %s stdout = %q; want %q", flag, got, want)
		}
	}
}

// TestRun_Help covers the three help spellings and the bare-help case.
func TestRun_Help(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "help"} {
		var out, errOut bytes.Buffer
		if err := run([]string{"ibkr", flag}, &out, &errOut); err != nil {
			t.Fatalf("run %s = %v; want nil", flag, err)
		}
		if !strings.Contains(errOut.String(), "Usage:") {
			t.Errorf("run %s did not print usage to stderr", flag)
		}
	}
}

// TestRun_SubHelp covers per-command help. An unknown subcommand must be an
// error, not a silent success: that is the difference between a typo being
// reported and being ignored.
func TestRun_SubHelp(t *testing.T) {
	for _, sub := range []string{"accounts", "positions", "orders", "stream", "portfolio", "config"} {
		var out, errOut bytes.Buffer
		if err := run([]string{"ibkr", "help", sub}, &out, &errOut); err != nil {
			t.Errorf("help %s = %v; want nil", sub, err)
		}
		if !strings.Contains(errOut.String(), "Usage: ibkr "+sub) {
			t.Errorf("help %s output = %q; want a usage line for it", sub, errOut.String())
		}
	}
}

func TestRun_HelpUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run([]string{"ibkr", "help", "nope"}, &out, &errOut)
	if err == nil {
		t.Fatal("help for an unknown subcommand = nil; want an error")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("err = %v; want it to name the unknown subcommand", err)
	}
}

// TestRun_UnknownCommand pins the error text and that usage still follows, so a
// typo is both reported and explained.
func TestRun_UnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run([]string{"ibkr", "acounts"}, &out, &errOut)
	if err == nil {
		t.Fatal("run with an unknown command = nil; want an error")
	}
	if got, want := err.Error(), `unknown command "acounts"`; got != want {
		t.Errorf("err = %q; want %q", got, want)
	}
	if !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("unknown command did not print usage; got %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "acounts") {
		t.Errorf("stderr does not echo the bad command; got %q", errOut.String())
	}
}

// TestRun_CompletionNeedsNoGateway is the one command that is pure computation,// so it is the only one whose full path can be asserted without a gateway. It
// also proves the dispatch table routes "completion" rather than treating it as
// unknown, and that the shell name comes from the caller's arguments rather than
// from os.Args - without that seam this test read the test binary's own flags
// and failed with `unsupported shell "-test.timeout=10m0s"`.
func TestRun_CompletionNeedsNoGateway(t *testing.T) {
	t.Run("no subcommand prints usage", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if err := run([]string{"ibkr", "completion"}, &out, &errOut); err != nil {
			t.Fatalf("run completion = %v; want nil", err)
		}
		if !strings.Contains(errOut.String(), "bash|zsh|fish") {
			t.Errorf("usage = %q; want it to name the supported shells", errOut.String())
		}
	})

	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if err := run([]string{"ibkr", "completion", shell}, &out, &errOut); err != nil {
				t.Fatalf("run completion %s = %v; want nil", shell, err)
			}
			got := out.String()
			if got == "" {
				t.Fatal("no completion script written to stdout")
			}
			if !strings.Contains(got, "ibkr") {
				t.Errorf("script does not mention the command name; got %.80q", got)
			}
			// Every advertised top-level command must appear, or the generated
			// script is stale relative to the dispatch table.
			for _, cmd := range []string{"accounts", "positions", "orders", "stream", "portfolio", "config"} {
				if !strings.Contains(got, cmd) {
					t.Errorf("%s completion omits the %q command", shell, cmd)
				}
			}
		})
	}

	t.Run("unsupported shell is an error", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := run([]string{"ibkr", "completion", "powershell"}, &out, &errOut)
		if err == nil {
			t.Fatal("run completion powershell = nil; want an error")
		}
		if !strings.Contains(err.Error(), "powershell") {
			t.Errorf("err = %v; want it to name the unsupported shell", err)
		}
	})
}
