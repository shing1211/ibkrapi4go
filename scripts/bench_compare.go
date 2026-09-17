// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type benchResult struct {
	TestName string
	NsOp     float64
	MBPerS   float64
	BPerOp   int64
	AllocsOp int64
}

type resultEntry struct {
	Action  string `json:"Action"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
	Package string `json:"Package"`
}

var nsOpRe = regexp.MustCompile(`([0-9.]+)\s+ns/op`)

func parseBenchResults(path string) (map[string]benchResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	results := make(map[string]benchResult)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry resultEntry
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Action != "output" || !strings.Contains(entry.Output, "ns/op") {
			continue
		}
		m := nsOpRe.FindStringSubmatch(entry.Output)
		if m == nil {
			continue
		}
		nsOp, _ := strconv.ParseFloat(m[1], 64)

		name := entry.Test
		if name == "" {
			continue
		}

		if _, ok := results[name]; ok {
			continue
		}

		results[name] = benchResult{
			TestName: name,
			NsOp:     nsOp,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return results, nil
}

func compare(results, baseline map[string]benchResult, threshold float64) ([]string, error) {
	var regressions []string

	allNames := make(map[string]struct{})
	for name := range baseline {
		allNames[name] = struct{}{}
	}
	for name := range results {
		allNames[name] = struct{}{}
	}

	var sorted []string
	for name := range allNames {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		cur, curOk := results[name]
		bas, basOk := baseline[name]

		if !curOk {
			fmt.Printf("%-70s  (baseline only: %s)\n", name, formatNs(bas.NsOp))
			continue
		}
		if !basOk {
			fmt.Printf("%-70s  (new benchmark: %s)\n", name, formatNs(cur.NsOp))
			continue
		}

		if bas.NsOp == 0 {
			continue
		}

		ratio := cur.NsOp / bas.NsOp
		regression := (ratio - 1) * 100

		marker := ""
		if regression > threshold {
			marker = "  <-- REGRESSION"
			regressions = append(regressions, fmt.Sprintf("%s: %.1f%% regression (%s -> %s)", name, regression, formatNs(bas.NsOp), formatNs(cur.NsOp)))
		}

		fmt.Printf("%-70s  baseline=%s  current=%s  ratio=%.3f  %+.1f%%%s\n",
			name, formatNs(bas.NsOp), formatNs(cur.NsOp), ratio, regression, marker)
	}

	return regressions, nil
}

func formatNs(ns float64) string {
	if ns >= 1e9 {
		return fmt.Sprintf("%.3fs", ns/1e9)
	}
	if ns >= 1e6 {
		return fmt.Sprintf("%.3fms", ns/1e6)
	}
	if ns >= 1e3 {
		return fmt.Sprintf("%.3fµs", ns/1e3)
	}
	return fmt.Sprintf("%.3fns", ns)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <current.json> <baseline.json>\n", os.Args[0])
		os.Exit(1)
	}

	currentPath := os.Args[1]
	baselinePath := os.Args[2]

	current, err := parseBenchResults(currentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing current results: %v\n", err)
		os.Exit(1)
	}

	baseline, err := parseBenchResults(baselinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing baseline: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Benchmark Comparison ===")
	fmt.Println()
	fmt.Printf("Current:   %s\n", currentPath)
	fmt.Printf("Baseline:  %s\n", baselinePath)
	fmt.Println()
	fmt.Println(fmt.Sprintf("%-70s  %-20s  %-20s  %-10s  %s", "Benchmark", "Baseline", "Current", "Ratio", "Change"))
	fmt.Println(strings.Repeat("-", 150))

	const threshold = 10.0
	regressions, err := compare(current, baseline, threshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing results: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	if len(regressions) > 0 {
		fmt.Printf("ERROR: %d benchmark(s) regressed by more than %.0f%%\n", len(regressions), threshold)
		for _, r := range regressions {
			fmt.Printf("  - %s\n", r)
		}
		os.Exit(1)
	}

	fmt.Printf("All benchmarks within %.0f%% threshold.\n", threshold)
	os.Exit(0)
}
