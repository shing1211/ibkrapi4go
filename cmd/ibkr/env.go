// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// env carries everything a subcommand needs from the process.
//
// Until this existed, every subcommand read os.Args and wrote to os.Stdout
// directly, and `orders` and `portfolio` went further: they rewrote os.Args in
// place to hand a subcommand's arguments to the next layer down. That is not
// merely untidy. `append(os.Args[:2], args[1:]...)` aliases the real argv backing
// array, so dispatching a second command in the same process corrupts argv - which
// is also what the test harness itself reads. The CLI had no tests because it
// could not have them.
//
// Passing argv and the output streams explicitly removes the global state, and
// newClient is the seam a test replaces to point the command at a mock gateway.
type env struct {
	// args is the full argument vector, with args[0] the program name, matching
	// the shape of os.Args so the index arithmetic in each subcommand is
	// unchanged and reviewable against the original.
	args   []string
	stdout io.Writer
	stderr io.Writer

	// newClient builds the SDK client. Production reads the global flags and the
	// config file; a test substitutes a client wired to the mock gateway.
	newClient func() (*ibkr.Client, error)
}

// newEnv builds the production environment.
func newEnv(args []string, stdout, stderr io.Writer) *env {
	e := &env{args: args, stdout: stdout, stderr: stderr}
	e.newClient = func() (*ibkr.Client, error) { return newClientFromArgs(e.args) }
	return e
}

// arg returns argv[n:], or an empty slice when argv is shorter than n. The
// original code indexed os.Args[n:] directly, which panicked on a short argv; a
// test invoking `ibkr accounts` must not crash the process.
func (e *env) arg(n int) []string {
	if n >= len(e.args) {
		return nil
	}
	return e.args[n:]
}

// ibkrPrintln writes a line to the command's stdout. It replaces the bare
// fmt.Println calls in the subcommands, which wrote to the process stdout and so
// could not be captured by a test.
func ibkrPrintln(e *env, msg string) {
	if e == nil || e.stdout == nil {
		return
	}
	_, _ = fmt.Fprintln(e.stdout, msg)
}
