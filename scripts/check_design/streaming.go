// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// streaming.go verifies docs/design/05-streaming.md against the code it
// describes.
//
// This document is the internals map of the WebSocket layer, and the parameters
// in it are the ones a reader cannot derive: how big a buffer is, how often a
// ping goes out, how long a pong has to come back, how far a reconnect backs off.
// Each of those is one number in a struct literal far from the document, so each
// is pinned here. A buffer that silently grew to 4096 would change the SDK's
// memory profile; a pong deadline that doubled would change when a healthy
// connection is torn down.
//
// The checks are:
//
//   - the subscription surface the components diagram lists, read out of the
//     diagram and compared with the methods Subscription actually declares.
//   - the per-subscription buffer default, in both of the two places the code
//     applies it, so a change to one of them is caught.
//   - the ping interval, the pong deadline and the reconnect backoff, taken from
//     the document's own numbers and compared with the defaults DialWS installs
//     and the sequence backoffDelay produces from them.
//
// What is deliberately NOT checked:
//
//   - The reverse direction of the subscription surface. The diagram shows what a
//     caller needs, and Subscription also carries AccountUpdates, PortfolioUpdates,
//     Dropped, Deliver, Fail, Wants and WantsSystem; the document says nothing
//     about them and does not claim the list is complete, so demanding a reverse
//     match would be enforcing a claim it does not make.
//   - `Close()`'s result type. The diagram writes `Close()` with no result where
//     pkg/ibkr/ws.go:186 declares `Close() error`. The diagram is a picture of a
//     type, not a signature listing, and the two other entries are the ones that
//     carry a channel a caller ranges over — the parts a diagram can be trusted
//     to spell out. Only the method's existence is compared for Close; the
//     result is a recorded difference, not an unenforced claim.
//   - "Writes are serialized by the writer goroutine; callers never write
//     directly", "Channels are closed exactly once (guarded by `sync.Once`)" and
//     "Channel sends use a select on `ctx.Done()`". Each is true and each is
//     about a body, but they are statements about the *shape* of many functions
//     rather than about a named one, and a check that has to enumerate the
//     functions to have something to read would be a whole-file audit rather than
//     a document check. The two facts with a single named home — the buffer
//     default and the ping/backoff parameters — are the ones pinned here.
//   - "Triggered by read/write error or ping timeout", "On success: re-issue all
//     active subscriptions with fresh ids" and "Emits a reconnected notice on each
//     subscription's `Errors()` channel". These describe a state machine spread
//     across the read, write and reconnect loops; a check over them would have to
//     re-derive the state machine, which is a different program from the one that
//     tells a maintainer their document is stale.
//   - The Testing section, for the reason 09's is not checked: it is a statement
//     about the test suite.

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	streamDoc  = "docs/design/05-streaming.md"
	streamSrc  = "pkg/ibkr/ws.go"
	streamInt  = "internal/ws.go"
	subscribeR = "*Subscription"
)

// checkStreaming runs the docs/design/05-streaming.md checks as one unit.
func checkStreaming(repoRoot string) error {
	return collectDocErrors(repoRoot, streamDoc, []docCheck{
		{"subscription surface", checkSubscriptionSurface},
		{"buffer default", checkStreamingBufferDefault},
		{"ping, pong and reconnect backoff", checkStreamingTimeouts},
	})
}

// --- claim 1: the subscription surface ----------------------------------------

// subMethodRe recognises an entry of the components diagram: a method name, and
// optionally the channel type it yields. The diagram is a plain fenced block, not
// a ```go one, so there is no signature to parse — only this shape. A `Name()`
// mention in the surrounding prose carries no `<-chan`, so it does not match, and
// the three methods the document commits to a type for are exactly the three the
// check compares a type for.
//
// The channel direction and the element type are captured separately and rejoined.
// One group would stop at the space in `<-chan Update` — or, being greedy, run on
// into the `//` comment that follows — so the direction is its own group and the
// comparison is against the two halves put back together.
var subMethodRe = regexp.MustCompile(`^\s*[│├└─\s]*([A-Z]\w*)\(\)(?:\s*(<-chan)\s+([A-Za-z_]\w*))?`)

// subMethod is one entry of the documented subscription surface.
type subMethod struct {
	name    string
	chanTyp string
	line    int
}

func (m subMethod) render() string {
	if m.chanTyp == "" {
		return m.name + "()"
	}
	return m.name + "() " + m.chanTyp
}

// checkSubscriptionSurface verifies the Subscription half of the components
// diagram against the methods pkg/ibkr/ws.go declares.
//
// The direction is diagram to code: every method the diagram lists must exist, and
// the three it gives a channel type for must yield that type. A caller reading
// `Updates() <-chan Update` off a range statement either compiles or does not, and
// this is the check that says which.
func checkSubscriptionSurface(repoRoot string) error {
	doc, err := parseDoc(repoRoot, streamDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, streamSrc)
	if err != nil {
		return err
	}
	methods, err := docSubscriptionSurface(doc)
	if err != nil {
		return err
	}
	for _, want := range methods {
		fn := code.method("Subscription", want.name)
		if fn == nil {
			return fmt.Errorf("streaming surface drift: %s lists Subscription.%s, which %s declares no such method",
				doc.at(want.line), want.name, streamSrc)
		}
		if want.chanTyp == "" {
			continue
		}
		got := fieldTypes(fn.Type.Results)
		if len(got) != 1 || got[0] != want.chanTyp {
			return fmt.Errorf("streaming surface drift: %s lists Subscription.%s as %q but %s declares %q",
				doc.at(want.line), want.name, want.render(), code.at(fn), codeFunc(fn).render())
		}
	}
	return nil
}

// docSubscriptionSurface reads the diagram's Subscription entries.
//
// The scan stops at the end of the fenced block the diagram lives in, so a later
// fenced block in the document cannot contribute a method. `wsConn (internal/)`
// and its goroutine lines are inside the same block and are not method entries:
// they either do not end in `()` or their leading text starts a word the pattern
// does not accept, and the `reader goroutine : ...` line fails to match outright.
func docSubscriptionSurface(doc *docFile) ([]subMethod, error) {
	var out []subMethod
	in, seen := false, false
	for i, l := range doc.lines {
		t := strings.TrimSpace(l)
		switch {
		case !in && strings.HasPrefix(t, "Subscription (public)"):
			in, seen = true, true
		case in && strings.HasPrefix(t, "```"):
			in = false
		case in:
			m := subMethodRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			typ := ""
			if m[2] != "" {
				typ = m[2] + " " + m[3]
			}
			out = append(out, subMethod{name: m[1], chanTyp: typ, line: i + 1})
		}
	}
	if !seen {
		return nil, fmt.Errorf("%s: the components diagram has no `Subscription (public)` block; the check cannot run", doc.rel)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: the Subscription block of the components diagram lists no methods; the check cannot run", doc.rel)
	}
	return out, nil
}

// --- claim 2: the buffer default ----------------------------------------------

// bufferDefaultRe reads the documented per-subscription buffer default.
var bufferDefaultRe = regexp.MustCompile(`(?i)buffer default\s+(\d+)`)

// checkStreamingBufferDefault verifies "Per-subscription buffer default 256".
//
// The number appears twice in the code — the streaming-limits default a caller can
// override, and the fallback inside newSubscription for a subscription built
// without limits — and both are checked, for the reason checkBaseDelayFallback
// gives about the second place a retry default is applied: two copies of a
// constant is a drift risk, and only the one a caller can reach is the one anyone
// thinks about.
func checkStreamingBufferDefault(repoRoot string) error {
	doc, err := parseDoc(repoRoot, streamDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, streamSrc)
	if err != nil {
		return err
	}
	claim, ok := docSentence(bufferDefaultRe, doc)
	if !ok {
		return fmt.Errorf("%s: the buffer-and-drop section no longer states a per-subscription buffer default; "+
			"the buffer check cannot run", doc.rel)
	}
	m := bufferDefaultRe.FindStringSubmatch(strings.Join(doc.lines, "\n"))
	want, err := strconv.Atoi(m[1])
	if err != nil {
		return fmt.Errorf("%s:%d: the documented buffer default %q is not a number: %w", doc.rel, claim, m[1], err)
	}

	// The limits default: read from the literal defaultStreamingLimits returns, so
	// a different struct type returned here cannot pass as the limits.
	limits := code.funcDecl("defaultStreamingLimits")
	if limits == nil {
		return fmt.Errorf("%s: no defaultStreamingLimits function is declared; the buffer check cannot run", streamSrc)
	}
	lit := firstReturnedStructLit(limits.Body, "StreamingLimits")
	if lit == nil {
		return fmt.Errorf("%s: defaultStreamingLimits returns no StreamingLimits literal; the buffer check cannot run",
			streamSrc)
	}
	assigned := map[string]keyedValue{}
	for _, kv := range keyedValues(lit) {
		assigned[kv.key] = kv
	}
	kv, ok := assigned["BufferSize"]
	if !ok {
		return fmt.Errorf("buffer drift: %s documents a per-subscription buffer default of %d, but %s's "+
			"defaultStreamingLimits sets no BufferSize, so a zero StreamingLimits would buffer nothing",
			doc.at(claim), want, streamSrc)
	}
	got, ok := intValue(kv.expr)
	if !ok {
		return fmt.Errorf("buffer drift: %s documents a per-subscription buffer default of %d, but %s assigns %s "+
			"to BufferSize, which is not a constant", doc.at(claim), want, code.at(kv.node), exprString(kv.expr))
	}
	if got != want {
		return fmt.Errorf("buffer drift: %s documents a per-subscription buffer default of %d but %s assigns %d",
			doc.at(claim), want, code.at(kv.node), got)
	}

	// The second copy: newSubscription's own floor, which is what a subscription
	// built with a non-positive buffer actually gets.
	sub := code.funcDecl("newSubscription")
	if sub == nil {
		return fmt.Errorf("%s: no newSubscription function is declared; the buffer check cannot run", streamSrc)
	}
	floor, node := assignedIntInGuard(sub.Body, "buffer")
	if node == nil {
		return fmt.Errorf("buffer drift: %s documents a per-subscription buffer default of %d, but %s's "+
			"newSubscription assigns no floor to its buffer parameter, so a non-positive buffer would make an "+
			"unbuffered channel and the documented drop policy could not happen", doc.at(claim), want, streamSrc)
	}
	if floor != want {
		return fmt.Errorf("buffer drift: %s documents a per-subscription buffer default of %d but the floor at %s is %d",
			doc.at(claim), want, code.at(node), floor)
	}
	return nil
}

// --- claim 3: ping, pong and the reconnect backoff ----------------------------

// pingRe reads the ping interval and the pong deadline out of the diagram. The
// two are read as a pair because the diagram states them in one clause, and a
// check that took only the ping would leave the deadline free to drift.
var pingRe = regexp.MustCompile(`(?i)(\d+\s*[a-z]+)\s+ping,\s*(\d+\s*[a-z]+)\s+pong deadline`)

// backoffRe reads the backoff ladder. The document writes the unit once, after the
// last step ("1,2,4,8,16,30s cap"), so the numbers and the unit are captured
// separately and the unit is applied to every step. The last step is the cap,
// which is what the code's max is; the ones before it are the successive
// doublings, which is what the code's shift has to produce.
var backoffRe = regexp.MustCompile(`(?i)backoff:\s*([0-9,]+)s\s+cap`)

// checkStreamingTimeouts verifies the diagram's "30s ping, 10s pong deadline" and
// the reconnect ladder "1,2,4,8,16,30s cap, with jitter".
//
// Every number comes out of the document, so editing the document is what moves
// the expectation, and each is compared with the declaration that supplies it:
//
//   - the ping interval and the pong deadline against the defaults DialWS installs
//     when the caller supplies none, each read from the guard on the field the
//     WSOptions declaration names;
//   - the ladder against backoffDelay, which has to double the base and clamp at
//     the cap for the documented sequence to be the one a caller observes;
//   - the jitter against the return expression, which has to be a random draw over
//     the computed delay rather than the delay itself.
func checkStreamingTimeouts(repoRoot string) error {
	doc, err := parseDoc(repoRoot, streamDoc)
	if err != nil {
		return err
	}
	intr, err := parseGo(repoRoot, streamInt)
	if err != nil {
		return err
	}
	pingLine, ping, pong, err := docPingPong(doc)
	if err != nil {
		return err
	}
	dial := intr.funcDecl("DialWS")
	if dial == nil {
		return fmt.Errorf("%s: no DialWS function is declared; the timeout check cannot run", streamInt)
	}
	// The field names are resolved from the WSOptions declaration rather than
	// written here, so a renamed option is reported instead of silently matching a
	// different assignment.
	for _, c := range []struct {
		what  string
		field string
		want  time.Duration
	}{
		{"ping interval", wsOptionsDurationField(intr, "PingInterval"), ping},
		{"pong deadline", wsOptionsDurationField(intr, "PongTimeout"), pong},
	} {
		if c.field == "" {
			return fmt.Errorf("%s: no WSOptions duration field declares the documented %s; the timeout check "+
				"cannot run", streamInt, c.what)
		}
		node, got, err := defaultAssignedInDial(dial, c.field)
		if err != nil {
			return fmt.Errorf("timeout drift: %s: the documented %s cannot be read: %w", doc.at(pingLine), c.what, err)
		}
		if got != c.want {
			return fmt.Errorf("timeout drift: %s documents a %s of %s but %s installs %s at %s",
				doc.at(pingLine), c.what, c.want, streamInt, got, intr.at(node))
		}
	}

	ladder, err := docBackoffLadder(doc)
	if err != nil {
		return err
	}
	delay := intr.funcDecl("backoffDelay")
	if delay == nil {
		return fmt.Errorf("%s: no backoffDelay function is declared; the backoff check cannot run", streamInt)
	}
	if !doublesByShifting(delay.Body) {
		return fmt.Errorf("backoff drift: %s documents the reconnect ladder %s, but %s's backoffDelay does not "+
			"double the base delay; the sequence a caller observes would not be the documented one",
			doc.rel, ladder.render(), streamInt)
	}
	if !clampsDelayAtMax(delay.Body, delayLocal(delay)) {
		return fmt.Errorf("backoff drift: %s documents the reconnect ladder %s with a cap of %s, but %s's "+
			"backoffDelay has no `if d > max` clamp, so the ladder has no cap",
			doc.rel, ladder.render(), ladder.cap, streamInt)
	}
	if !returnsRandomBelowDelay(delay) {
		return fmt.Errorf("backoff drift: %s documents the reconnect ladder %s \"with jitter\", but %s's "+
			"backoffDelay does not return a random value below the computed delay, so every client would back off "+
			"in lockstep", doc.rel, ladder.render(), streamInt)
	}
	// The base and the cap are the ladder's first and last elements, and both are
	// read from the same declarations the timeout check reads.
	for _, c := range []struct {
		what  string
		field string
		want  time.Duration
	}{
		{"reconnect backoff base", wsOptionsDurationField(intr, "ReconnectBase"), ladder.steps[0]},
		{"reconnect backoff cap", wsOptionsDurationField(intr, "ReconnectMax"), ladder.cap},
	} {
		if c.field == "" {
			return fmt.Errorf("%s: no WSOptions duration field declares the documented %s; the "+
				"backoff check cannot run", streamInt, c.what)
		}
		node, got, err := defaultAssignedInDial(dial, c.field)
		if err != nil {
			return fmt.Errorf("backoff drift: %s: %w", doc.rel, err)
		}
		if got != c.want {
			return fmt.Errorf("backoff drift: %s documents the reconnect ladder %s, so its %s is %s, but %s installs %s at %s",
				doc.rel, ladder.render(), c.what, c.want, streamInt, got, intr.at(node))
		}
	}
	// The middle of the ladder is the claim about the shape: doubling the base has
	// to produce exactly the steps the document lists before the cap.
	walked := ladder.steps[0]
	for i := 1; i < len(ladder.steps); i++ {
		if ladder.steps[i] == ladder.cap {
			if i != len(ladder.steps)-1 {
				return fmt.Errorf("backoff drift: %s: the documented reconnect ladder %s has its cap in the middle, "+
					"so the steps before it are not a doubling of %s", doc.rel, ladder.render(), ladder.steps[0])
			}
			break
		}
		walked *= 2
		if ladder.steps[i] != walked {
			return fmt.Errorf("backoff drift: %s documents the reconnect ladder %s, but step %d is %s where "+
				"doubling the previous step gives %s", doc.rel, ladder.render(), i+1, ladder.steps[i], walked)
		}
	}
	return nil
}

func docPingPong(doc *docFile) (int, time.Duration, time.Duration, error) {
	text := strings.Join(doc.lines, "\n")
	m := pingRe.FindStringSubmatch(text)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("%s: the components diagram no longer states a ping interval and a pong deadline; "+
			"the timeout check cannot run", doc.rel)
	}
	ping, err := time.ParseDuration(strings.Join(strings.Fields(m[1]), ""))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s: the documented ping interval %q is not a duration: %w", doc.rel, m[1], err)
	}
	pong, err := time.ParseDuration(strings.Join(strings.Fields(m[2]), ""))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s: the documented pong deadline %q is not a duration: %w", doc.rel, m[2], err)
	}
	return strings.Count(text[:strings.Index(text, m[0])], "\n") + 1, ping, pong, nil
}

// backoffLadder is the documented reconnect sequence: the steps before the cap
// and the cap itself.
type backoffLadder struct {
	steps []time.Duration
	cap   time.Duration
}

func (l backoffLadder) render() string {
	var parts []string
	for _, s := range l.steps {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, ",") + " cap"
}

func docBackoffLadder(doc *docFile) (backoffLadder, error) {
	m := backoffRe.FindStringSubmatch(strings.Join(doc.lines, "\n"))
	if m == nil {
		return backoffLadder{}, fmt.Errorf("%s: the reconnect section no longer states a backoff ladder with a cap; "+
			"the backoff check cannot run", doc.rel)
	}
	var steps []time.Duration
	for _, part := range strings.Split(m[1], ",") {
		d, err := time.ParseDuration(part + "s")
		if err != nil {
			return backoffLadder{}, fmt.Errorf("%s: the documented backoff step %q is not a duration: %w", doc.rel, part, err)
		}
		steps = append(steps, d)
	}
	if len(steps) < 2 {
		return backoffLadder{}, fmt.Errorf("%s: the documented backoff ladder names one step; there is nothing to "+
			"compare against a cap", doc.rel)
	}
	return backoffLadder{steps: steps[:len(steps)-1], cap: steps[len(steps)-1]}, nil
}

// wsOptionsDurationField returns the name of the WSOptions field of the given
// name when it is declared as a time.Duration, or "".
func wsOptionsDurationField(code *goSrc, field string) string {
	fields, err := code.structFields("WSOptions")
	if err != nil {
		return ""
	}
	for _, f := range fields {
		if f.name == field && f.typ == "time.Duration" {
			return field
		}
	}
	return ""
}

// defaultAssignedInDial returns the duration DialWS installs for a WSOptions field
// when the caller supplied none, found in the `if opts.<field> <= 0` guard. The
// guard is located by the field name rather than by position, so a reordered
// default block does not make the check read a different field.
func defaultAssignedInDial(dial *ast.FuncDecl, field string) (ast.Node, time.Duration, error) {
	opts := paramOfType(dial.Type, "WSOptions")
	if opts == "" {
		return nil, 0, fmt.Errorf("DialWS takes no WSOptions parameter, so the default cannot be read")
	}
	for _, stmt := range dial.Body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok || !isFieldLEZero(guard.Cond, opts, field) {
			continue
		}
		for _, inner := range guard.Body.List {
			as, ok := inner.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			if !isFieldOf(as.Lhs[0], opts, field) {
				continue
			}
			d, ok := wsDurationValue(as.Rhs[0])
			if !ok {
				return nil, 0, fmt.Errorf("the default for %s is %s, which is not a duration constant",
					field, exprString(as.Rhs[0]))
			}
			return as, d, nil
		}
		return nil, 0, fmt.Errorf("the `if %s.%s <= 0` guard assigns no duration to %s", opts, field, field)
	}
	return nil, 0, fmt.Errorf("DialWS has no `if %s.%s <= 0` default", opts, field)
}

// wsDurationValue reads a Go duration constant in either of the two spellings the
// streaming defaults use: a product such as `30 * time.Second` and a bare
// `time.Second`.
//
// A separate reader rather than a widening of durationValue, which the
// 06-errors-retries.md defaults check reads: teaching that one a new shape would
// change what it accepts underneath a check that already depends on it. Here the
// widening is the point — `ReconnectBase` really is assigned a bare `time.Second`,
// and a reader that only understood products would report it as no default at all.
func wsDurationValue(e ast.Expr) (time.Duration, bool) {
	if d, ok := durationValue(e); ok {
		return d, true
	}
	return timeUnit(e)
}

// isFieldLEZero reports whether cond is `<recv>.<field> <= 0`.
func isFieldLEZero(cond ast.Expr, recv, field string) bool {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.LEQ {
		return false
	}
	if v, ok := intValue(b.Y); !ok || v != 0 {
		return false
	}
	return isFieldOf(b.X, recv, field)
}

// firstReturnedStructLit returns the composite literal of the named type a
// function returns, or nil.
func firstReturnedStructLit(body *ast.BlockStmt, typ string) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		if cl, ok := ret.Results[0].(*ast.CompositeLit); ok && structLitType(cl) == typ {
			found = cl
		}
		return true
	})
	return found
}

// assignedIntInGuard returns the integer a function assigns to a local inside a
// non-positive guard on that local, and the assignment's node.
func assignedIntInGuard(body *ast.BlockStmt, name string) (int, ast.Node) {
	for _, stmt := range body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok || !isIntLEZero(guard.Cond, name) {
			continue
		}
		for _, inner := range guard.Body.List {
			as, ok := inner.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			if !isIdent(as.Lhs[0], name) {
				continue
			}
			if v, ok := intValue(as.Rhs[0]); ok {
				return v, as
			}
			return 0, as
		}
	}
	return 0, nil
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// doublesByShifting reports whether a body computes a delay as `base << n`.
func doublesByShifting(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		sh, ok := n.(*ast.BinaryExpr)
		if !ok || sh.Op != token.SHL {
			return true
		}
		if _, ok := sh.Y.(*ast.Ident); ok {
			found = true
		}
		return true
	})
	return found
}

// clampsDelayAtMax reports whether a body contains `if delay > max { delay = max }`.
//
// The delay is the local the shift accumulates into and max is the identifier the
// guard compares against; both are resolved from the clamp itself rather than
// written here, so a clamp on the wrong variable — or one that assigned something
// other than the bound — is reported rather than accepted.
func clampsDelayAtMax(body *ast.BlockStmt, delay string) bool {
	if delay == "" {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		guard, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		b, ok := guard.Cond.(*ast.BinaryExpr)
		if !ok || b.Op != token.GTR {
			return true
		}
		if !isIdent(b.X, delay) {
			return true
		}
		bound, ok := b.Y.(*ast.Ident)
		if !ok {
			return true
		}
		for _, inner := range guard.Body.List {
			as, ok := inner.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			if isIdent(as.Lhs[0], delay) && isIdent(as.Rhs[0], bound.Name) {
				found = true
			}
		}
		return true
	})
	return found
}

// returnsRandomBelowDelay reports whether backoffDelay's return value is a random
// draw bounded by the computed delay.
//
// The draw is found inside the returned expression rather than as the expression
// itself, because the code converts it: `return time.Duration(rand.Int63n(int64(d)
// + 1))`. Demanding the call be the whole result would report a jittered backoff
// as a deterministic one — the check would be right about the fact and wrong about
// the code, which is the worst combination.
func returnsRandomBelowDelay(fn *ast.FuncDecl) bool {
	delay := delayLocal(fn)
	if delay == "" {
		return false
	}
	found := false
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		ast.Inspect(ret, func(n ast.Node) bool {
			if found {
				return false
			}
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 || calleeName(call.Fun) != "Int63n" {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "rand" {
				return true
			}
			if mentionsIdent(call.Args[0], delay) {
				found = true
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// delayLocal returns the identifier backoffDelay accumulates the delay in.
//
// It is the *target* of the assignment the shift feeds, not the shift's left
// operand: `d := base << attempt` stores the doubled base in `d`, and the clamp
// that gives the ladder its cap assigns to `d` as well. Reading the shift's
// operand would return `base` — the uncapped input — and the clamp would then look
// like it was on the wrong variable.
func delayLocal(fn *ast.FuncDecl) string {
	name := ""
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if name != "" {
			return false
		}
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		sh, ok := as.Rhs[0].(*ast.BinaryExpr)
		if !ok || sh.Op != token.SHL {
			return true
		}
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			name = id.Name
		}
		return true
	})
	return name
}

func mentionsIdent(e ast.Expr, name string) bool {
	if name == "" {
		return false
	}
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			found = true
		}
		return !found
	})
	return found
}
