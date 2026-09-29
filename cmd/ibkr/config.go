// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// defaultGatewayURL is used when the config file is missing or names no
// gateway. It is a constant rather than a literal repeated at both defaulting
// sites so the two cannot drift - a client built against one default and a
// config file written by the other would silently disagree.
const defaultGatewayURL = "https://localhost:5000"

// cliConfig holds the persistent CLI configuration.
type cliConfig struct {
	GatewayURL     string `json:"gateway_url"`
	RestGatewayURL string `json:"rest_gateway_url"`
	AccountID      string `json:"account_id"`
}

// configPath returns the config file location from IBKR_CONFIG env or default.
func configPath() string {
	if v := os.Getenv("IBKR_CONFIG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ibkr/config.json"
	}
	return filepath.Join(home, ".ibkr", "config.json")
}

// loadConfig reads the config file, returning defaults if missing.
func loadConfig() (cliConfig, error) {
	var cfg cliConfig
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cliConfig{
				GatewayURL: defaultGatewayURL,
			}, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = defaultGatewayURL
	}
	return cfg, nil
}

// saveConfig writes the config file.
func saveConfig(cfg cliConfig) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	perm := os.FileMode(0o600)
	if runtime.GOOS == "windows" {
		perm = 0o644
	}
	return os.WriteFile(p, data, perm)
}

func runConfig(e *env, args []string) error {
	if len(args) == 0 {
		return runConfigShow(e)
	}

	switch args[0] {
	case "show":
		return runConfigShow(e)
	case "set":
		return runConfigSet(e, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintf(e.stderr, `Usage: ibkr config <subcommand>

Subcommands:
  show        Display current configuration
  set KEY VAL Set a configuration value

Keys:
  gateway     Client Portal Gateway URL
  rest        REST gateway URL
  account     Default account ID

Config file location: ~/.ibkr/config.json (or $IBKR_CONFIG)
`)
		return nil
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func runConfigShow(e *env) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	w := e.stdout
	_, _ = fmt.Fprintf(w, "Config file: %s\n\n", configPath())
	_, _ = fmt.Fprintf(w, "  gateway:     %s\n", cfg.GatewayURL)
	_, _ = fmt.Fprintf(w, "  rest:        %s\n", cfg.RestGatewayURL)
	_, _ = fmt.Fprintf(w, "  account:     %s\n", cfg.AccountID)
	return nil
}

func runConfigSet(e *env, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ibkr config set <key> <value>")
	}

	key := args[0]
	value := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	switch key {
	case "gateway":
		cfg.GatewayURL = value
	case "rest":
		cfg.RestGatewayURL = value
	case "account":
		cfg.AccountID = value
	default:
		return fmt.Errorf("unknown config key %q (valid: gateway, rest, account)", key)
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	// To the injected writer, not the process stdout. fmt.Printf here wrote
	// straight past env, so the confirmation was uncapturable by a test and
	// invisible to anything consuming the command's output - the same defect
	// that was fixed for the orders cancel message.
	_, _ = fmt.Fprintf(e.stdout, "config set: %s = %s\n", key, value)
	return nil
}
