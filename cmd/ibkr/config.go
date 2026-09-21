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
				GatewayURL: "https://localhost:5000",
			}, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = "https://localhost:5000"
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

func runConfig() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return runConfigShow()
	}

	switch args[0] {
	case "show":
		return runConfigShow()
	case "set":
		os.Args = append(os.Args[:2], args[1:]...)
		return runConfigSet()
	case "-h", "--help", "help":
		fmt.Fprintf(os.Stderr, `Usage: ibkr config <subcommand>

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

func runConfigShow() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	w := os.Stdout
	fmt.Fprintf(w, "Config file: %s\n\n", configPath())
	fmt.Fprintf(w, "  gateway:     %s\n", cfg.GatewayURL)
	fmt.Fprintf(w, "  rest:        %s\n", cfg.RestGatewayURL)
	fmt.Fprintf(w, "  account:     %s\n", cfg.AccountID)
	return nil
}

func runConfigSet() error {
	args := os.Args[2:]
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

	fmt.Printf("config set: %s = %s\n", key, value)
	return nil
}
