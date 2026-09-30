// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// red_test.go proves each check in errors_retries.go is load-bearing.
//
// The method here is mutation, not assertion-on-fixture: every case copies the
// real repository into a scratch tree, breaks exactly one fact, and requires the
// check to notice. A check that has never been observed failing is not a check,
// and a green test that cannot fail is worse than a red one — so these cases
// assert on the *message*, not merely on "an error happened", because a check
// that fires for the wrong reason is as much a defect as one that stays silent.
//
// One rule keeps the results honest: a mutation whose anchor does not occur
// exactly once fails the test. A silently-inert edit would leave the check
// green for the wrong reason and the case would pass while proving nothing.
//
// The whole suite shares one scratch tree, built once and restored after every
// mutation, because copying 80-odd files per case costs more than the checks
// do. Subtests therefore run serially; nothing here calls t.Parallel.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// scratchFiles is everything the checks read. pkg/ibkr, client, internal and cmd
// are taken whole because a check has to scan the packages, not just the
// declarations: checkNoImplicitBreakerDefault and checkConfirmationIsExplicit look
// for a writer of the breaker field and a second caller of the reply endpoint,
// checkImportBoundary looks for *every* package that imports the generated code,
// and the 09 endpoint check resolves a path through client/client.gen.go — a
// partial copy could hide a writer, an importer or an endpoint.
//
// go.mod is copied because the import boundary derives the generated package's
// import path from the module line rather than hardcoding it, so a scratch tree
// without one has no boundary to check.
var scratchFiles = []string{
	"go.mod",
	errDoc,
	ordersDoc,
	clientDoc,
	wrapDoc,
	streamDoc,
	concDoc,
	moneyDoc,
	"internal",
	"pkg/ibkr",
	"client",
	"cmd",
}

// scratchRoot is the shared tree. TestMain builds it; mutate restores what it
// touches.
var scratchRoot string

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "check_design_scratch")
	if err != nil {
		fmt.Fprintf(os.Stderr, "scratch tree: %v\n", err)
		os.Exit(1)
	}
	src, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "locate repo root: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	for _, rel := range scratchFiles {
		if err := copyPath(filepath.Join(src, filepath.FromSlash(rel)), filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			fmt.Fprintf(os.Stderr, "copy %s: %v\n", rel, err)
			_ = os.RemoveAll(root)
			os.Exit(1)
		}
	}
	scratchRoot = root
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

// findRepoRoot walks up from the working directory to the module root, so the
// harness does not depend on being invoked from a particular directory.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}

// copyPath copies a file, or a directory tree. A directory is copied whole,
// subdirectories included: the 04 import-boundary check scans every package in
// the module, and a scratch tree that quietly dropped internal/mockgateway would
// make that check pass because the copy was incomplete rather than because the
// boundary holds. Only .go files are copied from a directory; a check never reads
// anything else, and go.mod is listed in scratchFiles as a file in its own right.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		b, err := os.ReadFile(src) //nolint:gosec // repo-local scratch path, not user input
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o600) //nolint:gosec // G703: repo-local scratch path under t.TempDir(), never caller input
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// mutate replaces old with new in the named file of the shared scratch tree,
// and arranges for the original bytes to be restored when the test ends —
// including when the test fails, so one case cannot leak into the next.
func mutate(t *testing.T, rel, old, repl string) {
	t.Helper()
	p := filepath.Join(scratchRoot, filepath.FromSlash(rel))
	orig, err := os.ReadFile(p) //nolint:gosec // repo-local scratch path, not user input
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(p, orig, 0o600); err != nil { //nolint:gosec // G703: repo-local scratch path under t.TempDir(), never caller input
			t.Fatalf("restore %s: %v", p, err)
		}
	})
	s := string(orig)
	// The document in the working tree is CRLF while .gitattributes normalizes
	// the committed blob to LF, so an anchor spanning lines has to be spelled
	// in the file's own line endings. Getting this wrong would make every
	// multi-line case pass vacuously.
	if strings.Contains(s, "\r\n") {
		old = strings.ReplaceAll(old, "\n", "\r\n")
		repl = strings.ReplaceAll(repl, "\n", "\r\n")
	}
	if n := strings.Count(s, old); n != 1 {
		t.Fatalf("anchor %q occurs %d times in %s, want exactly 1", old, n, rel)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(s, old, repl, 1)), 0o600); err != nil { //nolint:gosec // G703: repo-local scratch path under t.TempDir(), never caller input
		t.Fatalf("write %s: %v", p, err)
	}
}

// edit is one mutation to apply before running the check.
type edit struct {
	rel  string
	old  string
	repl string
}

func replace(rel, old, repl string) edit { return edit{rel: rel, old: old, repl: repl} }

// redCase is a mutation that must be caught, and the substring of the failure
// that identifies which check caught it.
type redCase struct {
	name  string
	edits []edit
	want  string
}

// stripLineNumbers removes the `:NNN` suffix the checkers append to a file
// reference, so a red case can assert on the substance of a message rather than
// on an incidental line number.
//
// Every checker's diagnostics cite `path:line`, which means any edit above a
// declaration - adding a doc comment, say - shifted every pinned line number and
// failed these cases with "failed for the wrong reason", even though the checker
// had in fact detected the mutation. The line number carries no meaning for the
// assertion: what matters is that the checker named the right file, symbol and
// problem.
var lineNumberRE = regexp.MustCompile(`:\d+`)

func stripLineNumbers(s string) string { return lineNumberRE.ReplaceAllString(s, ":") }

// greenCase is a tree that must pass, which is what stops a change to the
// checker from passing by breaking every case at once.
type greenCase struct {
	name  string
	edits []edit
}

const (
	doc      = errDoc
	retry    = retrySrcPath
	breaker  = breakerPath
	trans    = transportPath
	ierrors  = errorsSrcPath
	pubErr   = "pkg/ibkr/errors.go"
	pubTrade = "pkg/ibkr/trade.go"
)

// TestErrorRetriesChecksAreLoadBearing mutates one documented or coded fact per
// case and requires the message that names it.
func TestErrorRetriesChecksAreLoadBearing(t *testing.T) {
	for _, tc := range redCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			err := checkErrorRetries(scratchRoot)
			if err == nil {
				t.Fatalf("checkErrorRetries passed; expected a failure containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("checkErrorRetries failed for the wrong reason.\nwant substring: %q\ngot: %v", tc.want, err)
			}
		})
	}
}

// TestErrorRetriesPassesOnUnmutatedTree is the control. Without it a checker
// that rejected every tree would satisfy every red case above.
func TestErrorRetriesPassesOnUnmutatedTree(t *testing.T) {
	if err := checkErrorRetries(scratchRoot); err != nil {
		t.Fatalf("checkErrorRetries on the unmutated tree: %v", err)
	}
}

// TestErrorRetriesPassesOnGreenCases covers edits that are true statements about
// the code, so the checker has to accept them. Each one is a variation on the
// no-default syntax, which is the part of the checker this run changed.
func TestErrorRetriesPassesOnGreenCases(t *testing.T) {
	for _, tc := range greenCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			if err := checkErrorRetries(scratchRoot); err != nil {
				t.Fatalf("checkErrorRetries rejected a true statement about the code: %v", err)
			}
		})
	}
}

func redCases() []redCase {
	return []redCase{
		// --- claim 1: the Error struct's field set --------------------------
		{
			name:  "error/doc-drops-a-field",
			edits: []edit{replace(doc, "    RequestID  string\n    Err        error\n", "    Err        error\n")},
			want:  "error struct drift",
		},
		{
			name:  "error/doc-invents-a-field",
			edits: []edit{replace(doc, "    Err        error\n}", "    Err        error\n    Extra      int\n}")},
			want:  "error struct drift",
		},
		{
			name: "error/doc-reorders-fields",
			edits: []edit{replace(doc,
				"    Op         string\n    Code       string\n",
				"    Code       string\n    Op         string\n")},
			want: "error struct drift",
		},
		{
			name:  "error/doc-wrong-type",
			edits: []edit{replace(doc, "    HTTPStatus int\n", "    HTTPStatus string\n")},
			want:  "documents field HTTPStatus as string",
		},
		{
			name:  "error/code-drops-a-field",
			edits: []edit{replace(ierrors, "\tRequestID  string\n", "")},
			want:  "error struct drift",
		},
		{
			name:  "error/code-renames-a-field",
			edits: []edit{replace(ierrors, "\tRequestID  string\n", "\tRequestIdent string\n")},
			want:  "error struct drift",
		},

		// --- claim 1b: the public alias -------------------------------------
		{
			name:  "alias/stops-being-an-alias",
			edits: []edit{replace(pubErr, "type Error = internal.Error", "type Error internal.Error")},
			want:  "is not an alias",
		},
		{
			name:  "alias/points-somewhere-else",
			edits: []edit{replace(pubErr, "type Error = internal.Error", "type Error = fmt.Stringer")},
			want:  "does not alias internal.Error",
		},

		// --- claim 2a: the RetryPolicy field set and types --------------------
		{
			name: "retry/doc-documents-a-field-the-code-lacks",
			edits: []edit{replace(doc,
				"    Metrics       Metrics       // optional;",
				"    Timeout       int           // default 1s\n    Metrics       Metrics       // optional;")},
			want: `documents field "Timeout", which internal/retry.go does not declare`,
		},
		{
			name:  "retry/doc-wrong-type",
			edits: []edit{replace(doc, "    MaxAttempts   int           // default 3", "    MaxAttempts   int64         // default 3")},
			want:  "documents field MaxAttempts as int64",
		},

		// --- claim 2b: each of the five defaults, checked on both sides ------
		// One case per defaulted field. All five must keep verifying: a change
		// that exempted one of them would show up as a missing red here.
		{
			name:  "default/maxattempts-drifts-in-code",
			edits: []edit{replace(retry, "MaxAttempts:   3,", "MaxAttempts:   5,")},
			want:  `documents MaxAttempts default as "3" but`,
		},
		{
			name:  "default/maxattempts-drifts-in-doc",
			edits: []edit{replace(doc, "    MaxAttempts   int           // default 3", "    MaxAttempts   int           // default 4")},
			want:  `documents MaxAttempts default as "4" but`,
		},
		{
			name:  "default/basedelay-drifts-in-code",
			edits: []edit{replace(retry, "BaseDelay:     200 * time.Millisecond,", "BaseDelay:     250 * time.Millisecond,")},
			want:  "documents BaseDelay default as",
		},
		{
			name:  "default/basedelay-drifts-in-doc",
			edits: []edit{replace(doc, "    BaseDelay     time.Duration // default 200ms", "    BaseDelay     time.Duration // default 300ms")},
			want:  "documents BaseDelay default as",
		},
		{
			name:  "default/maxdelay-drifts-in-code",
			edits: []edit{replace(retry, "MaxDelay:      5 * time.Second,", "MaxDelay:      9 * time.Second,")},
			want:  "documents MaxDelay default as",
		},
		{
			name:  "default/maxdelay-drifts-in-doc",
			edits: []edit{replace(doc, "    MaxDelay      time.Duration // default 5s", "    MaxDelay      time.Duration // default 7s")},
			want:  "documents MaxDelay default as",
		},
		{
			name:  "default/jitter-drifts-in-code",
			edits: []edit{replace(retry, "Jitter:        true,", "Jitter:        false,")},
			want:  "documents Jitter default as",
		},
		{
			name:  "default/jitter-drifts-in-doc",
			edits: []edit{replace(doc, "    Jitter        bool          // default true (full jitter)", "    Jitter        bool          // default false")},
			want:  "documents Jitter default as",
		},
		{
			name:  "default/retrystatus-drifts-in-code",
			edits: []edit{replace(retry, "RetryOnStatus: []int{429, 500, 502, 503, 504},", "RetryOnStatus: []int{429, 500, 502, 503},")},
			want:  "documents RetryOnStatus default as",
		},
		{
			name:  "default/retrystatus-reordered-in-code",
			edits: []edit{replace(retry, "RetryOnStatus: []int{429, 500, 502, 503, 504},", "RetryOnStatus: []int{504, 503, 502, 500, 429},")},
			want:  "documents RetryOnStatus default as",
		},
		{
			name:  "default/retrystatus-drifts-in-doc",
			edits: []edit{replace(doc, "// default 429,500,502,503,504", "// default 429,500,502,503")},
			want:  "documents RetryOnStatus default as",
		},
		{
			name:  "default/code-stops-assigning-a-defaulted-field",
			edits: []edit{replace(retry, "\t\tJitter:        true,\n", "")},
			want:  "documents a default for Jitter but internal/retry.go:DefaultRetryPolicy never assigns it",
		},

		// --- claim 2c: the second place BaseDelay is applied -----------------
		{
			name:  "basedelay/fallback-drifts",
			edits: []edit{replace(retry, "\t\tbase = 200 * time.Millisecond", "\t\tbase = 300 * time.Millisecond")},
			want:  "documents BaseDelay default as 200ms but the fallback",
		},
		{
			name: "basedelay/fallback-guard-removed",
			edits: []edit{replace(retry,
				"\tif base <= 0 {\n\t\tbase = 200 * time.Millisecond\n\t}\n", "")},
			want: "delayFor has no `if base <= 0` fallback",
		},
		{
			name:  "basedelay/doc-drops-the-default",
			edits: []edit{replace(doc, "    BaseDelay     time.Duration // default 200ms", "    BaseDelay     time.Duration // see DefaultRetryPolicy")},
			want:  "has no readable `default` comment",
		},

		// --- claim 2d: the no-default marker, which must not be a loophole ---
		{
			name:  "no-default/code-starts-assigning-a-marked-field",
			edits: []edit{replace(retry, "\t\tJitter:        true,\n", "\t\tJitter:        true,\n\t\tMetrics:      nil,\n")},
			want:  "marks Metrics as having no assigned default, but",
		},
		{
			name:  "no-default/field-claims-neither",
			edits: []edit{replace(doc, "// optional; nil (the zero value) is valid and reports nothing", "// observability sink")},
			want:  "neither a `default` clause nor a no-default marker",
		},
		{
			name:  "no-default/marker-misspelt",
			edits: []edit{replace(doc, "// optional; nil (the zero value)", "// optionall; nil (the zero value)")},
			want:  "neither a `default` clause nor a no-default marker",
		},
		{
			name:  "no-default/marker-without-separator",
			edits: []edit{replace(doc, "// optional; nil (the zero value)", "// optional nil (the zero value)")},
			want:  "neither a `default` clause nor a no-default marker",
		},
		{
			name:  "no-default/marker-not-first-clause",
			edits: []edit{replace(doc, "// optional; nil (the zero value)", "// observability sink; the zero value is nil")},
			want:  "neither a `default` clause nor a no-default marker",
		},
		{
			name:  "no-default/mentioning-defaults-is-not-a-marker",
			edits: []edit{replace(doc, "// optional; nil (the zero value)", "// see DefaultRetryPolicy; the zero value is nil")},
			want:  "neither a `default` clause nor a no-default marker",
		},
		{
			name:  "no-default/empty-comment",
			edits: []edit{replace(doc, "    Metrics       Metrics       // optional; nil (the zero value) is valid and reports nothing", "    Metrics       Metrics")},
			want:  "neither a `default` clause nor a no-default marker",
		},

		// --- claim 3: the safe-method set ------------------------------------
		{
			name:  "safe-methods/doc-adds-post",
			edits: []edit{replace(doc, "**Safe methods only**: `GET`, `HEAD`, `OPTIONS`.", "**Safe methods only**: `GET`, `HEAD`, `OPTIONS`, `POST`.")},
			want:  "safe-method drift",
		},
		{
			name:  "safe-methods/doc-drops-options",
			edits: []edit{replace(doc, "**Safe methods only**: `GET`, `HEAD`, `OPTIONS`.", "**Safe methods only**: `GET`, `HEAD`.")},
			want:  "safe-method drift",
		},
		{
			name:  "safe-methods/code-drops-options",
			edits: []edit{replace(retry, "	case http.MethodGet, http.MethodHead, http.MethodOptions:", "	case http.MethodGet, http.MethodHead:")},
			want:  "safe-method drift",
		},
		{
			name: "safe-methods/code-drops-the-default-clause",
			edits: []edit{replace(retry,
				"	case http.MethodGet, http.MethodHead, http.MethodOptions:\n		return true\n	default:\n		return false\n	}",
				"	case http.MethodGet, http.MethodHead, http.MethodOptions:\n		return true\n	}")},
			want: "has no default clause, so it is not a whitelist",
		},
		{
			name:  "safe-methods/doc-lists-no-methods",
			edits: []edit{replace(doc, "**Safe methods only**: `GET`, `HEAD`, `OPTIONS`.", "**Safe methods only**: none.")},
			want:  "lists no backtick-quoted methods",
		},

		// --- claim 4: ADR 0009, order mutations are never retried ------------
		{
			name: "no-retry/guard-removed",
			edits: []edit{replace(retry,
				"			if !p.enabled() || !isSafeMethod(req.Method) || req.Body != nil {\n				return base.RoundTrip(req)\n			}\n", "")},
			want: "not the safe-method guard",
		},
		{
			name:  "no-retry/guard-inverted",
			edits: []edit{replace(retry, "if !p.enabled() || !isSafeMethod(req.Method) || req.Body != nil {", "if p.enabled() && isSafeMethod(req.Method) && req.Body == nil {")},
			want:  "does not test !isSafeMethod",
		},
		{
			name:  "no-retry/guard-no-longer-returns-the-base-transport",
			edits: []edit{replace(retry, "\t\t\t\treturn base.RoundTrip(req)\n", "\t\t\t\treturn RoundTripFunc(req)\n")},
			want:  "does not return base.RoundTrip(req)",
		},
		{
			name:  "reconcile/mutate-bypasses-netdo",
			edits: []edit{replace(pubTrade, "	resp, err := m.client.netDo(ctx, op, fn)", "	resp, err := fn()")},
			want:  "does not go through Client.netDo",
		},
		{
			name:  "reconcile/code-changes",
			edits: []edit{replace(pubTrade, `Code:    "ambiguous",`, `Code:    "rejected",`)},
			want:  `document "ambiguous"`,
		},
		{
			name: "reconcile/code-drops-the-instruction",
			edits: []edit{replace(pubTrade,
				"the order may or may not have been accepted — reconcile via Trade().OpenOrders before retrying",
				"the order may or may not have been accepted — retry later")},
			want: "does not tell the caller to reconcile",
		},
		{
			name: "reconcile/error-removed",
			edits: []edit{replace(pubTrade,
				"			return nil, &Error{\n				Op:      op,\n				Code:    \"ambiguous\",\n				Message: \"ambiguous outcome; the order may or may not have been accepted — reconcile via Trade().OpenOrders before retrying\",\n				Err:     err,\n			}\n",
				"			return nil, err\n")},
			want: "never constructs an Error for a failed mutation",
		},

		// --- claim 5: the breaker is disabled by default --------------------
		{
			name:  "breaker/threshold-guard-removed",
			edits: []edit{replace(breaker, "\tif threshold <= 0 {\n\t\treturn nil\n\t}\n", "")},
			want:  "has no `if threshold <= 0 { return nil }` guard",
		},
		{
			name:  "breaker/threshold-guard-returns-a-breaker",
			edits: []edit{replace(breaker, "\tif threshold <= 0 {\n\t\treturn nil\n\t}", "\tif threshold <= 0 {\n\t\treturn &Breaker{}\n\t}")},
			want:  "does not return nil",
		},
		{
			name:  "breaker/middleware-removed",
			edits: []edit{replace(trans, "\tif cfg.Breaker != nil {\n\t\tms = append(ms, CircuitBreaker(cfg.Breaker))\n\t}\n", "")},
			want:  "no longer installs the CircuitBreaker middleware",
		},
		{
			name:  "breaker/installed-unconditionally",
			edits: []edit{replace(trans, "\tif cfg.Breaker != nil {\n\t\tms = append(ms, CircuitBreaker(cfg.Breaker))\n\t}", "\tms = append(ms, CircuitBreaker(cfg.Breaker))")},
			want:  "installed unconditionally",
		},
		{
			name:  "breaker/installed-under-a-different-condition",
			edits: []edit{replace(trans, "\tif cfg.Breaker != nil {\n\t\tms = append(ms, CircuitBreaker(cfg.Breaker))", "\tif cfg.Limiter != nil {\n\t\tms = append(ms, CircuitBreaker(cfg.Breaker))")},
			want:  "not a nil check on cfg.Breaker",
		},
		{
			name: "breaker/an-implicit-default-appears",
			edits: []edit{replace(pubErr,
				"// Error implements the error interface.\nfunc (e *ConfigError) Error() string {",
				"func withDefaultBreaker(cfg *ClientConfig) { cfg.breaker = NewBreaker(3, 0) }\n\n// Error implements the error interface.\nfunc (e *ConfigError) Error() string {")},
			want: "writes the client config's \"breaker\" field",
		},
	}
}

func greenCases() []greenCase {
	const metricsDoc = "// optional; nil (the zero value) is valid and reports nothing"
	return []greenCase{
		{
			name:  "no-default/the-spelling-in-the-document-passes",
			edits: nil,
		},
		{
			name:  "no-default/explicit-spelling-passes",
			edits: []edit{replace(doc, metricsDoc, "// no assigned default; nil reports nothing")},
		},
		{
			name:  "no-default/colon-spelling-passes",
			edits: []edit{replace(doc, metricsDoc, "// no default: the zero value is a valid no-op")},
		},
		{
			name: "no-default/marker-plus-prose-passes",
			edits: []edit{replace(doc, metricsDoc,
				"// no assigned default: the zero value is intended, so nil is left unassigned")},
		},
		{
			name: "basedelay/fallback-in-a-different-but-equal-form-passes",
			edits: []edit{replace(retry,
				"\t\tbase = 200 * time.Millisecond",
				"\t\tbase = time.Millisecond * 200")},
		},
	}
}
