// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The config subcommand is the one part of the CLI that writes to the user's own
// filesystem rather than to a gateway, and it was the last command with no
// coverage at all. It is also the only place where a test can do real damage if
// it reaches for the default path: ~/.ibkr/config.json holds the operator's live
// gateway URL and account ID.
//
// Every test here points IBKR_CONFIG at t.TempDir(). configPath honours that
// variable, so the seam already existed - the gap was that nothing used it.

func TestConfigPath_HonoursEnv(t *testing.T) {
	want := filepath.Join(t.TempDir(), "nested", "cfg.json")
	t.Setenv("IBKR_CONFIG", want)
	if got := configPath(); got != want {
		t.Errorf("configPath() = %q; want %q", got, want)
	}
}

// TestConfigPath_Default covers the fallback, which is what a test that forgot to
// set IBKR_CONFIG would land on. It asserts the path shape rather than reading
// the user's home directory.
func TestConfigPath_Default(t *testing.T) {
	t.Setenv("IBKR_CONFIG", "")
	got := configPath()
	if !strings.HasSuffix(filepath.ToSlash(got), ".ibkr/config.json") {
		t.Errorf("configPath() = %q; want it to end in .ibkr/config.json", got)
	}
	if filepath.IsAbs(got) {
		t.Skipf("configPath() = %q, which is absolute, so the home directory is available", got)
	}
	// No home directory available: the relative path is the documented fallback
	// rather than a panic.
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	t.Setenv("IBKR_CONFIG", path)

	want := cliConfig{
		GatewayURL:     "https://gateway.example:5000",
		RestGatewayURL: "https://rest.example:5000",
		AccountID:      "U1234567",
	}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig = %v; want nil", err)
	}

	// The parent directory did not exist, so this also covers the MkdirAll.
	//nolint:gosec // path is t.TempDir() joined with a constant in this test, never caller input
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config = %v; want nil", err)
	}
	var got cliConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse saved config = %v; want nil", err)
	}
	if got != want {
		t.Errorf("saved config = %+v; want %+v", got, want)
	}
}

// TestSaveConfig_Permissions covers the file mode. The config file holds an
// account ID, and the mode is the only thing keeping it off a shared machine.
func TestSaveConfig_Permissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	if err := saveConfig(cliConfig{GatewayURL: "https://g:1"}); err != nil {
		t.Fatalf("saveConfig = %v; want nil", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat = %v; want nil", err)
	}
	if runtime.GOOS == "windows" {
		// Windows has no POSIX mode; saveConfig already documents the 0644
		// fallback there, so there is nothing to assert.
		t.Skipf("POSIX file modes are not meaningful on %s", runtime.GOOS)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o; want 600 - the file holds an account ID", perm)
	}
}

func TestSaveConfig_Overwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	if err := saveConfig(cliConfig{GatewayURL: "https://first:1", AccountID: "U1"}); err != nil {
		t.Fatalf("first saveConfig = %v; want nil", err)
	}
	if err := saveConfig(cliConfig{GatewayURL: "https://second:2"}); err != nil {
		t.Fatalf("second saveConfig = %v; want nil", err)
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig = %v; want nil", err)
	}
	if cfg.GatewayURL != "https://second:2" {
		t.Errorf("gateway = %q; want the second write to have replaced the first", cfg.GatewayURL)
	}
	if cfg.AccountID != "" {
		t.Errorf("account = %q; want it cleared - a rewrite replaces the file, it does not merge", cfg.AccountID)
	}
}

func TestLoadConfig_DefaultsWhenMissing(t *testing.T) {
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "absent.json"))
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig = %v; want nil for a missing file", err)
	}
	if cfg.GatewayURL != defaultGatewayURL {
		t.Errorf("gateway = %q; want the default %q", cfg.GatewayURL, defaultGatewayURL)
	}
}

// TestLoadConfig_DefaultsEmptyGateway covers the second defaulting site. A config
// file that exists but has no gateway - written by an older version, or
// hand-edited - must still produce a usable URL rather than an empty one that
// every request would fail against.
func TestLoadConfig_DefaultsEmptyGateway(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	if err := os.WriteFile(path, []byte(`{"account_id":"U9"}`), 0o600); err != nil {
		t.Fatalf("write config = %v; want nil", err)
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig = %v; want nil", err)
	}
	if cfg.GatewayURL != defaultGatewayURL {
		t.Errorf("gateway = %q; want the default %q", cfg.GatewayURL, defaultGatewayURL)
	}
	if cfg.AccountID != "U9" {
		t.Errorf("account = %q; want U9 preserved", cfg.AccountID)
	}
}

func TestLoadConfig_Malformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write config = %v; want nil", err)
	}
	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig on malformed JSON = nil; want a parse error")
	}
}

// TestLoadConfig_Unreadable covers the error branch that is neither "missing" nor
// "malformed". A directory in place of the file is the portable way to produce an
// EISDIR on every platform this runs on.
func TestLoadConfig_Unreadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "config.json")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("mkdir = %v; want nil", err)
	}
	t.Setenv("IBKR_CONFIG", dir)
	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig on a directory = nil; want a read error")
	}
}

func TestRunConfigShow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	if err := saveConfig(cliConfig{
		GatewayURL:     "https://gateway.example:5000",
		RestGatewayURL: "https://rest.example:5000",
		AccountID:      "U1234567",
	}); err != nil {
		t.Fatalf("saveConfig = %v; want nil", err)
	}

	e, out, _ := newTestEnv("ibkr", "config", "show")
	if err := runConfigShow(e); err != nil {
		t.Fatalf("runConfigShow = %v; want nil", err)
	}
	got := out.String()
	for _, want := range []string{path, "https://gateway.example:5000", "https://rest.example:5000", "U1234567"} {
		if !strings.Contains(got, want) {
			t.Errorf("config show output is missing %q\n---\n%s", want, got)
		}
	}
}

// TestRunConfigSet_RoundTrip is the behaviour a user depends on: set a value, and
// read it back on the next invocation.
func TestRunConfigSet_RoundTrip(t *testing.T) {
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	cases := []struct {
		key   string
		value string
		check func(cliConfig) bool
	}{
		{"gateway", "https://new-gateway:1", func(c cliConfig) bool { return c.GatewayURL == "https://new-gateway:1" }},
		{"rest", "https://new-rest:2", func(c cliConfig) bool { return c.RestGatewayURL == "https://new-rest:2" }},
		{"account", "UNEW", func(c cliConfig) bool { return c.AccountID == "UNEW" }},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			e, out, _ := newTestEnv("ibkr", "config", "set", tc.key, tc.value)
			if err := runConfigSet(e, []string{tc.key, tc.value}); err != nil {
				t.Fatalf("runConfigSet = %v; want nil", err)
			}
			if !strings.Contains(out.String(), tc.value) {
				t.Errorf("confirmation = %q; want it to echo the value written", out.String())
			}
			cfg, err := loadConfig()
			if err != nil {
				t.Fatalf("loadConfig = %v; want nil", err)
			}
			if !tc.check(cfg) {
				t.Errorf("config after set %s = %+v; want the value persisted", tc.key, cfg)
			}
		})
	}
}

// TestRunConfigSet_WritesToInjectedStdout covers the defect this subcommand
// carried. The confirmation used fmt.Printf, which writes to the process stdout
// and bypasses env entirely, so the message was uncapturable by a test and
// invisible to any consumer reading the command's output. Every other message in
// the CLI goes through e.stdout.
func TestRunConfigSet_WritesToInjectedStdout(t *testing.T) {
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	e, out, _ := newTestEnv("ibkr", "config", "set", "account", "U1")
	if err := runConfigSet(e, []string{"account", "U1"}); err != nil {
		t.Fatalf("runConfigSet = %v; want nil", err)
	}
	if want := "config set: account = U1"; !strings.Contains(out.String(), want) {
		t.Errorf("stdout = %q; want it to contain %q", out.String(), want)
	}
}

// TestRunConfigSet_MissingValue covers the arity check. Writing a partial config
// because a value was forgotten is worse than refusing.
func TestRunConfigSet_MissingValue(t *testing.T) {
	t.Setenv("IBKR_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	e, _, _ := newTestEnv("ibkr", "config", "set", "account")
	if err := runConfigSet(e, []string{"account"}); err == nil {
		t.Fatal("runConfigSet with no value = nil; want a usage error")
	}
	if _, err := os.Stat(configPath()); err == nil {
		t.Error("a config file was written despite the usage error")
	}
}

func TestRunConfigSet_UnknownKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)
	e, _, _ := newTestEnv("ibkr", "config", "set", "nonsense", "x")
	err := runConfigSet(e, []string{"nonsense", "x"})
	if err == nil {
		t.Fatal("runConfigSet with an unknown key = nil; want an error")
	}
	// The message has to name the valid keys, or the user cannot recover.
	for _, want := range []string{"gateway", "rest", "account"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q; want it to list the valid key %q", err, want)
		}
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("a config file was written for a rejected key")
	}
}

func TestRunConfig_Dispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("IBKR_CONFIG", path)

	t.Run("no subcommand shows", func(t *testing.T) {
		e, out, _ := newTestEnv("ibkr", "config")
		if err := runConfig(e, nil); err != nil {
			t.Fatalf("runConfig = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "gateway:") {
			t.Errorf("output = %q; want the show listing", out.String())
		}
	})

	t.Run("show", func(t *testing.T) {
		e, out, _ := newTestEnv("ibkr", "config", "show")
		if err := runConfig(e, []string{"show"}); err != nil {
			t.Fatalf("runConfig show = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "gateway:") {
			t.Errorf("output = %q; want the show listing", out.String())
		}
	})

	t.Run("help", func(t *testing.T) {
		e, _, errOut := newTestEnv("ibkr", "config", "--help")
		if err := runConfig(e, []string{"--help"}); err != nil {
			t.Fatalf("runConfig --help = %v; want nil", err)
		}
		help := errOut.String()
		for _, want := range []string{"show", "set", "gateway", "rest", "account", "IBKR_CONFIG"} {
			if !strings.Contains(help, want) {
				t.Errorf("help text is missing %q\n---\n%s", want, help)
			}
		}
	})

	t.Run("unknown subcommand", func(t *testing.T) {
		e, _, _ := newTestEnv("ibkr", "config", "nope")
		if err := runConfig(e, []string{"nope"}); err == nil {
			t.Fatal("runConfig with an unknown subcommand = nil; want an error")
		}
	})
}
