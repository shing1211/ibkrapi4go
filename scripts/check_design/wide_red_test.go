// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// wide_red_test.go proves each check in client_composition.go,
// generated_wrapping.go, streaming.go and concurrency.go is load-bearing, by the
// same mutation method red_test.go and orders_red_test.go use: copy the real
// repository into the shared scratch tree, break exactly one documented or coded
// fact, and require the message that names it.
//
// The controls mirror the ones in the other two harness files, and they are the
// reason a case here means anything:
//
//   - TestWideDocsPassOnUnmutatedTree stops a checker that rejected every tree
//     from satisfying every red case.
//   - wideGreenCases holds edits that are *true statements about the code* — a
//     renamed parameter, a manager field written on several lines, an unexported
//     generated field — so a checker weakened into matching names or positions
//     fails them.
//   - TestStructBlockCheckerRunsBothDirections is the negative control for the one
//     direction that is genuinely new in this batch: the struct block's
//     code-to-document half. It breaks only that half, requires the real check to
//     fail, and requires a deliberately weakened document-to-code-only comparison
//     to pass — because that is what checkPublicOrderAPI did before this run, and
//     the reason two OrderRequest fields went undocumented.
//   - TestStructBlockCheckerSubsetControlIsNotInert keeps the control honest: it
//     has to fail on a document-side defect too, or "the subset check passed"
//     above would be satisfied by a function that never fails anything.
package main

import (
	"fmt"
	"strings"
	"testing"
)

const (
	cdoc     = clientDoc
	wdoc     = wrapDoc
	sdoc     = streamDoc
	kdoc     = concDoc
	cclient  = clientSrcPath
	cresp    = respSrcPath
	cerrs    = pubErrSrcPath
	wsPub    = streamSrc
	wsInt    = streamInt
	transp   = transportSrc
	obsv     = observabilityP
	oauth    = oauthSrc
	pubOauth = "pkg/ibkr/oauth.go"
	pool     = poolSrcPath
	gwcmd    = "cmd/ibkr-mock-gateway/main.go"
	mockT    = "internal/mockgateway/shape_test.go"
	mockT2   = "internal/mockgateway/decode_path_test.go"
)

// TestWideDocsChecksAreLoadBearing mutates one documented or coded fact per case
// and requires the message that names it.
func TestWideDocsChecksAreLoadBearing(t *testing.T) {
	for _, tc := range wideRedCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			err := checkWideDocs(scratchRoot)
			if err == nil {
				t.Fatalf("checkWideDocs passed; expected a failure containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("checkWideDocs failed for the wrong reason.\nwant substring: %q\ngot: %v", tc.want, err)
			}
		})
	}
}

// checkWideDocs runs the four documents this batch added as one unit, so a red case
// that happens to break two of them still reports the one it was written for.
func checkWideDocs(repoRoot string) error {
	var errs []string
	for _, c := range []struct {
		doc string
		fn  func(string) error
	}{
		{clientDoc, checkClientComposition},
		{wrapDoc, checkGeneratedWrapping},
		{streamDoc, checkStreaming},
		{concDoc, checkConcurrency},
	} {
		if err := c.fn(repoRoot); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

// TestWideDocsPassOnUnmutatedTree is the control for every red case above.
func TestWideDocsPassOnUnmutatedTree(t *testing.T) {
	if err := checkWideDocs(scratchRoot); err != nil {
		t.Fatalf("checkWideDocs on the unmutated tree: %v", err)
	}
}

// TestWideDocsPassOnGreenCases covers edits that are true statements about the
// code, which the checkers have to accept.
func TestWideDocsPassOnGreenCases(t *testing.T) {
	for _, tc := range wideGreenCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			if err := checkWideDocs(scratchRoot); err != nil {
				t.Fatalf("checkWideDocs rejected a true statement about the code: %v", err)
			}
		})
	}
}

// TestWideDocsAggregatorReportsEveryCheck exists because a check that is written,
// reviewed and never wired into the aggregator is not a check. It breaks two
// independent facts in two different documents' checks and requires both messages
// from one run.
func TestWideDocsAggregatorReportsEveryCheck(t *testing.T) {
	mutate(t, cdoc, "func (c *Client) REST() (*RESTSurface, error)", "func (c *Client) REST() (RESTSurface, error)")
	mutate(t, sdoc, "Per-subscription buffer default 256.", "Per-subscription buffer default 512.")
	err := checkWideDocs(scratchRoot)
	if err == nil {
		t.Fatal("checkWideDocs passed; expected both the accessor and the buffer failure")
	}
	for _, want := range []string{"client accessor drift", "buffer drift"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the aggregator did not report %q.\ngot: %v", want, err)
		}
	}
}

// --- the both-directions control ---------------------------------------------

// TestStructBlockCheckerRunsBothDirections breaks only the code-to-document half
// of the Client struct comparison and requires the real check to fail. The control
// is checkStructNamesSubset, the document-to-code direction alone: a check with
// only that direction is exactly what passed while two OrderRequest fields went
// undocumented, so if it can see these cases they prove nothing about the new one.
func TestStructBlockCheckerRunsBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit edit
	}{
		{"code-adds-a-field-at-the-end", replace(cclient,
			"\twsMu sync.Mutex\n", "\twsMu sync.Mutex\n\tLease sync.Mutex\n")},
		{"code-adds-a-field-at-the-top", replace(cclient,
			"\tcfg        config\n", "\tcfg        config\n\tlease      sync.Mutex\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutate(t, tc.edit.rel, tc.edit.old, tc.edit.repl)
			doc, err := parseDoc(scratchRoot, clientDoc)
			if err != nil {
				t.Fatalf("parseDoc: %v", err)
			}
			code, err := parseGo(scratchRoot, clientSrcPath)
			if err != nil {
				t.Fatalf("parseGo: %v", err)
			}
			if err := checkClientStructBlock(scratchRoot); err == nil {
				t.Fatal("checkClientStructBlock passed on a tree with a field the document omits")
			}
			if err := checkStructNamesSubset(doc, code, "Client"); err != nil {
				t.Fatalf("the weakened subset check also failed, so this case does not isolate the "+
					"code-to-document direction: %v", err)
			}
		})
	}
}

// TestStructBlockCheckerSubsetControlIsNotInert keeps the control honest. It
// renames a field in the *document*, which the subset check does read, and
// requires the control to fail. Without this, "the subset check passed" above
// would be satisfied by a function that never fails anything — and this is a
// names-only comparison, so a type drift would not exercise it.
func TestStructBlockCheckerSubsetControlIsNotInert(t *testing.T) {
	mutate(t, cdoc, "    cfg        config\n", "    cfgStore   config\n")
	doc, err := parseDoc(scratchRoot, clientDoc)
	if err != nil {
		t.Fatalf("parseDoc: %v", err)
	}
	code, err := parseGo(scratchRoot, clientSrcPath)
	if err != nil {
		t.Fatalf("parseGo: %v", err)
	}
	if err := checkStructNamesSubset(doc, code, "Client"); err == nil {
		t.Fatal("checkStructNamesSubset passed on a document that renames a field; the control is inert")
	}
}

// checkStructNamesSubset is the weaker comparison the controls above need: every
// field the document's struct block lists must be declared, and nothing whatsoever
// is said about the fields the code declares. It is here only so a test can show
// the difference between the two directions; no check in this program calls it.
func checkStructNamesSubset(doc *docFile, code *goSrc, name string) error {
	docFields, err := doc.structBlockNames(name)
	if err != nil {
		return err
	}
	realFields, err := code.structFields(name)
	if err != nil {
		return err
	}
	for _, df := range docFields {
		if !declaresField(realFields, df.name) {
			return fmt.Errorf("%s documents %s field %q, which is not declared", doc.rel, name, df.name)
		}
	}
	return nil
}

// --- cases -------------------------------------------------------------------

func wideRedCases() []redCase {
	return []redCase{
		// --- 02: the Client struct block -------------------------------------
		{
			name:  "struct/doc-invents-a-field",
			edits: []edit{replace(cdoc, "    rest   *RESTSurface  // lazily dialed (oauth2Bearer)", "    venue   string      // never declared\n    rest   *RESTSurface  // lazily dialed (oauth2Bearer)")},
			want:  `documents Client field "venue"`,
		},
		{
			// The direction that was missing: a declared field neither documented
			// nor listed among the note's omissions.
			name:  "struct/code-adds-an-undocumented-field",
			edits: []edit{replace(cclient, "\twsMu sync.Mutex\n", "\twsMu sync.Mutex\n\tLease sync.Mutex\n")},
			want:  "neither documents nor lists among the fields its completeness note omits",
		},
		{
			// The note is the document's own list of omissions, so a field the note
			// names but the code does not declare is a note that can no longer be
			// read as such — and would otherwise let a real field through.
			//
			// The anchor is the note's only backticked name. It was rewritten when
			// the human corrected the note (2026-09-27): the note used to name
			// `wsMu` and `restMu` as omissions while the block above it listed both,
			// and it now names only the pool `release` callback.
			//
			// The bogus name is *added* rather than substituted for `release`, and
			// that is load-bearing. The old note named three fields, so swapping
			// one left `release` still allowed the code-to-document direction to
			// pass and the stale-note comparison to be the only thing left to
			// report. This note names one field, so replacing it would instead make
			// the real `release` unaccounted-for and the check would report *that*
			// first — the case would go red for the wrong reason, which the `want`
			// substring below is there to catch.
			name:  "struct/the-note-omits-a-field-that-does-not-exist",
			edits: []edit{replace(cdoc, "> `release` callback is omitted for clarity.", "> `release` and the `wsConnQ` field are omitted for clarity.")},
			want:  `says the block omits "wsConnQ", but pkg/ibkr/client.go declares no such field`,
		},
		{
			// With the note gone the check has no list of omissions to read, and
			// must say so rather than quietly demand every field.
			name:  "struct/the-completeness-note-is-gone",
			edits: []edit{replace(cdoc, "> **Note**: The struct diagram shows all significant fields. The pool", "> **Note**: Some fields are shown. The pool")},
			want:  "no note scoping the struct diagram's completeness was found",
		},
		{
			// Both lines, because the note spans them: dropping only the backticks
			// on the second leaves the first to carry the sentence, and a
			// single-line anchor would have stopped applying the moment the note
			// was rewrapped.
			name: "struct/the-note-names-no-omission",
			edits: []edit{replace(cdoc,
				"> **Note**: The struct diagram shows all significant fields. The pool\n> `release` callback is omitted for clarity.",
				"> **Note**: The struct diagram shows all significant fields. The pool\n> callback is omitted for clarity.")},
			want: "names no omitted field",
		},
		{
			name:  "struct/doc-lists-a-field-twice",
			edits: []edit{replace(cdoc, "    ws     *internal.WSConn  // lazily dialed\n", "    ws     *internal.WSConn  // lazily dialed\n    ws     *internal.WSConn  // listed twice\n")},
			want:  "lists field",
		},

		// --- 02: the accessor list --------------------------------------------
		{
			name:  "accessor/doc-retypes-a-result",
			edits: []edit{replace(cdoc, "func (c *Client) REST() (*RESTSurface, error)", "func (c *Client) REST() (RESTSurface, error)")},
			want:  "client accessor drift",
		},
		{
			name:  "accessor/doc-invents-an-accessor",
			edits: []edit{replace(cdoc, "func (c *Client) Trade() *TradeManager", "func (c *Client) Trading() *TradeManager")},
			want:  "no Client.Trading method is declared anywhere in pkg/ibkr",
		},
		{
			name:  "accessor/doc-gives-an-accessor-a-parameter",
			edits: []edit{replace(cdoc, "func (c *Client) Session() *SessionManager", "func (c *Client) Session(ctx context.Context) *SessionManager")},
			want:  "client accessor drift",
		},
		{
			// REST lives in rest.go, not client.go. A check that read one file would
			// have reported this as a missing method.
			name:  "accessor/rest-accessor-moves-file",
			edits: []edit{replace("pkg/ibkr/rest.go", "func (c *Client) REST() (*RESTSurface, error)", "func (c *Client) Rest() (*RESTSurface, error)")},
			want:  "no Client.REST method is declared anywhere in pkg/ibkr",
		},

		// --- 02: the NewClient contract ---------------------------------------
		{
			name:  "newclient/doc-changes-the-signature",
			edits: []edit{replace(cdoc, "func NewClient(opts ...Option) (*Client, error)", "func NewClient(opts ...Option) (*Client)")},
			want:  "client construction drift",
		},
		{
			// The option's error has to reach the caller, or a caller matching on
			// *ConfigError with errors.As would miss it.
			name:  "newclient/the-option-error-is-wrapped",
			edits: []edit{replace(cclient, "		if err := o(&cfg); err != nil {\n			return nil, err\n		}", "		if err := o(&cfg); err != nil {\n			return nil, fmt.Errorf(\"option: %w\", err)\n		}")},
			want:  `the guard returns fmt.Errorf("option: %w", err), which is neither the option's error nor a nil`,
		},
		{
			name:  "newclient/the-option-error-is-replaced",
			edits: []edit{replace(cclient, "		if err := o(&cfg); err != nil {\n			return nil, err\n		}", "		if err := o(&cfg); err != nil {\n			return nil, &ConfigError{Field: \"Options\", Message: \"rejected\"}\n		}")},
			want:  "the guard replaces the option's error with a new ConfigError",
		},
		{
			name: "newclient/the-option-guard-is-gone",
			edits: []edit{replace(cclient,
				"	for _, o := range opts {\n		if err := o(&cfg); err != nil {\n			return nil, err\n		}\n	}\n",
				"	for _, o := range opts {\n		_ = o(&cfg)\n	}\n")},
			want: "contains no `if err := <option>(&cfg); err != nil` guard",
		},
		{
			// The claim is "no I/O", so a transport built by dialing breaks it even
			// though every signature still matches.
			name: "newclient/construction-starts-io",
			edits: []edit{replace(cclient,
				"	base, jar := baseTransport(cfg)\n",
				"	base, jar := baseTransport(cfg)\n	if _, err := base.RoundTrip(reqForProbe()); err != nil {\n		return nil, err\n	}\n")},
			want: "so NewClient performs I/O or starts the session",
		},
		{
			name: "newclient/construction-authenticates",
			edits: []edit{replace(cclient,
				"	applyEnv(&cfg)\n",
				"	applyEnv(&cfg)\n	if err := cfg.tokenSource.Initialize(context.Background()); err != nil {\n		return nil, err\n	}\n")},
			want: "so NewClient performs I/O or starts the session",
		},
		{
			name:  "newclient/doc-drops-the-signature",
			edits: []edit{replace(cdoc, "func NewClient(opts ...Option) (*Client, error)", "NewClient takes options and returns a client.")},
			want:  "no documented `func NewClient` signature",
		},

		// --- 02: the options block ---------------------------------------------
		// The block named two options that do not compile — WithOAuth2JWTKeyPath
		// is declared nowhere, and WithOAuth2JWTKey was documented as taking
		// []byte when it takes a *rsa.PrivateKey — so it went unchecked until the
		// human corrected the document on 2026-09-27. These cases are what the
		// block would have needed, and the one that reads
		// `*rsa.PrivateKey -> []byte` in the code is that exact defect, caught
		// from the other side.
		{
			name:  "options/doc-invents-an-option",
			edits: []edit{replace(cdoc, "WithGatewayURL(string)", "WithVaultEndpoint(string)")},
			want:  "documents WithVaultEndpoint, but no package-level func WithVaultEndpoint is declared anywhere in pkg/ibkr",
		},
		{
			name:  "options/doc-drops-a-parameter",
			edits: []edit{replace(cdoc, "WithGatewayURL(string)", "WithGatewayURL()")},
			want:  "documents WithGatewayURL() but pkg/ibkr/client.go:84 declares func WithGatewayURL(string)",
		},
		{
			name:  "options/doc-retypes-a-parameter",
			edits: []edit{replace(cdoc, "WithRateLimit(rps float64, burst int)", "WithRateLimit(rps float64, burst int64)")},
			want:  "documents WithRateLimit(float64, int64) but pkg/ibkr/client.go:158 declares func WithRateLimit(float64, int)",
		},
		{
			// A grouped parameter is two parameters of one type. A reader that
			// split the list on commas and kept the leading identifier would read
			// `budget` as a type and miss every regrouped form of this line.
			name:  "options/doc-invents-a-grouped-parameter",
			edits: []edit{replace(cdoc, "WithCircuitBreakerBudget(budget, size int)", "WithCircuitBreakerBudget(budget int, size int, floor int)")},
			want:  "documents WithCircuitBreakerBudget(int, int, int) but",
		},
		{
			// The defect the document itself had, reintroduced on the code side.
			// pkg/ibkr/oauth.go, not internal/oauth.go: the public constructors
			// are the ones the options block names, and internal/oauth.go declares
			// the token source behind them.
			name:  "options/the-key-option-stops-taking-a-private-key",
			edits: []edit{replace(pubOauth, "func WithOAuth2JWTKey(key *rsa.PrivateKey) Option {", "func WithOAuth2JWTKey(key []byte) Option {")},
			want:  "documents WithOAuth2JWTKey(*rsa.PrivateKey) but pkg/ibkr/oauth.go:80 declares func WithOAuth2JWTKey([]byte)",
		},
		{
			// A second result is what `NewClient(opts ...Option)` cannot accept,
			// and the document's own construction signature is what says so.
			name:  "options/the-option-stops-being-an-option",
			edits: []edit{replace(cclient, "func WithGlobalRateLimit(rps float64) Option {", "func WithGlobalRateLimit(rps float64) (Option, error) {")},
			want:  "whose single result is (Option, error) rather than Option",
		},
		{
			name:  "options/doc-lists-an-option-twice",
			edits: []edit{replace(cdoc, "WithUserAgent(string)\n", "WithUserAgent(string)\nWithUserAgent(string)\n")},
			want:  "documents WithUserAgent at both",
		},
		{
			// The reader has to reach the block's last line, not stop after the
			// first few. The reported line number is the assertion: 119 is the
			// final option line of the block, and the other cases in this group
			// all sit at or before 107.
			name:  "options/the-last-option-line-is-read",
			edits: []edit{replace(cdoc, "WithOAuth2JWTKeyReader(r io.Reader)", "WithOAuth2JWTKeyReader(r *rsa.PrivateKey)")},
			want:  "docs/design/02-client.md:119 documents WithOAuth2JWTKeyReader(*rsa.PrivateKey)",
		},
		{
			// The line stops being an option line, so it stops being compared.
			// Reading and filtering in one pass would drop it silently and leave
			// twenty-one verified options looking like twenty-two.
			name:  "options/a-line-stops-being-an-option",
			edits: []edit{replace(cdoc, "WithStreamingLimits(StreamingLimits)", "WithStreamingLimits")},
			want:  `reads "WithStreamingLimits", which is not an option line of the form`,
		},

		// --- 02: the close contract -------------------------------------------
		{
			name:  "close/doc-changes-the-signature",
			edits: []edit{replace(cdoc, "func (c *Client) Close() error", "func (c *Client) Close(ctx context.Context) error")},
			want:  "close drift",
		},
		{
			// Swap to Load and the second Close no longer observes the first, so the
			// guard is a no-op and the document's idempotence claim is false.
			name:  "close/the-idempotence-guard-stops-swapping",
			edits: []edit{replace(cclient, "\tif c.closed.Swap(true) {\n\t\treturn nil\n\t}", "\tif c.closed.Load() {\n\t\treturn nil\n\t}")},
			want:  "not a swap of c.closed",
		},
		{
			name:  "close/the-guard-no-longer-returns-nil",
			edits: []edit{replace(cclient, "\tif c.closed.Swap(true) {\n\t\treturn nil\n\t}", "\tif c.closed.Swap(true) {\n\t\treturn c.session.Close(context.Background())\n\t}")},
			want:  "the idempotence guard does not return nil",
		},
		{
			name:  "close/the-guard-is-no-longer-first",
			edits: []edit{replace(cclient, "func (c *Client) Close() error {\n\tif c.closed.Swap(true) {", "func (c *Client) Close() error {\n\t_ = c.ws\n\tif c.closed.Swap(true) {")},
			want:  "not the idempotence guard",
		},
		{
			// ErrClosed reaching a caller is the whole "after Close, manager methods
			// return ErrClosed" claim, and it lives at the netDo guard.
			name: "close/netdo-stops-guarding",
			edits: []edit{replace(cresp,
				"	if err := c.checkOpen(); err != nil {\n		return nil, err\n	}\n", "")},
			want: "not the closed guard",
		},
		{
			name:  "close/checkopen-returns-something-else",
			edits: []edit{replace(cclient, "\t\treturn internal.ErrClosed\n", "\t\treturn ErrNotAuthenticated\n")},
			want:  "Client.checkOpen does not return internal.ErrClosed",
		},
		{
			// The public sentinel has to be the one the guard returns, or the
			// document's claim is not a caller's to rely on.
			name:  "close/the-public-sentinel-is-redefined",
			edits: []edit{replace(cerrs, "	ErrClosed           = internal.ErrClosed", "	ErrClosed           = ErrNotAuthenticated")},
			want:  `binds the public ErrClosed to "ErrNotAuthenticated" rather than to internal.ErrClosed`,
		},

		// --- 04: the import boundary ------------------------------------------
		{
			// A package outside the boundary that reaches the generated code. The
			// import is added rather than moved, so the case cannot pass by the
			// anchor shifting and the failure names a package that really exists.
			name:  "boundary/an-outside-package-imports-the-generated-code",
			edits: []edit{replace(gwcmd, "import (\n", "import (\n\t\"github.com/shing1211/ibkrapi4go/client\"\n")},
			want:  "import-boundary drift: package cmd/ibkr-mock-gateway imports the generated package",
		},
		{
			// The other direction: an allowed root that reaches the generated code
			// nowhere. Both of internal/mockgateway's test files import it, so both
			// have to go; removing one would leave the boundary satisfied.
			name: "boundary/an-allowed-importer-stops-importing",
			edits: []edit{
				replace(mockT, "\t\"github.com/shing1211/ibkrapi4go/client\"\n", ""),
				replace(mockT2, "\t\"github.com/shing1211/ibkrapi4go/client\"\n", ""),
			},
			want: "neither package internal nor anything beneath it imports the generated package",
		},
		{
			name:  "boundary/the-rule-is-gone",
			edits: []edit{replace(wdoc, "- `client/*.gen.go` is imported **only** by `pkg/ibkr` and `internal/`.", "- Generated code is imported where it is convenient.")},
			want:  "no boundary rule naming the permitted importers was found",
		},

		// --- 04: the exported surface -----------------------------------------
		{
			name: "boundary/an-exported-field-leaks-a-generated-type",
			edits: []edit{replace(cclient,
				"// NewClient builds a Client from functional options",
				"// Leaked puts a generated type on the public surface.\ntype Leaked struct {\n\tRaw client.PortfolioPositionsResponse `json:\"raw\"`\n}\n\n// NewClient builds a Client from functional options")},
			want: "exported type Leaked has an exported field of the generated type",
		},
		{
			name:  "boundary/an-exported-type-alias-leaks-a-generated-type",
			edits: []edit{replace("pkg/ibkr/alerts.go", "// AlertManager exposes alert operations. It is safe for concurrent use.\ntype AlertManager struct {", "// Leaked is a generated type wearing a public name.\ntype Leaked = client.AlertDetails\n\n// AlertManager exposes alert operations. It is safe for concurrent use.\ntype AlertManager struct {")},
			want:  "exported type Leaked is declared as",
		},
		{
			name:  "boundary/an-exported-signature-leaks-a-generated-type",
			edits: []edit{replace(cclient, "func (c *Client) GatewayURL() string { return c.cfg.gatewayURL }", "func (c *Client) GatewayURL() client.PortfolioPositionsResponse {\n\tvar zero client.PortfolioPositionsResponse\n\treturn zero\n}")},
			want:  "has the generated type client.PortfolioPositionsResponse in its signature",
		},

		// --- 05: the subscription surface --------------------------------------
		{
			name:  "surface/doc-retypes-a-channel",
			edits: []edit{replace(sdoc, "├─ Updates()      <-chan Update       // market data field updates", "├─ Updates()      <-chan string      // market data field updates")},
			want:  `lists Subscription.Updates as "Updates() <-chan string"`,
		},
		{
			name:  "surface/code-retypes-a-channel",
			edits: []edit{replace(wsPub, "func (s *Subscription) SystemUpdates() <-chan SystemUpdate { return s.systemUpdates }", "func (s *Subscription) SystemUpdates() <-chan string { return s.systemUpdates }")},
			want:  `lists Subscription.SystemUpdates as "SystemUpdates() <-chan SystemUpdate"`,
		},
		{
			name:  "surface/doc-invents-a-method",
			edits: []edit{replace(sdoc, "├─ Errors()       <-chan error        // reconnect notices, connection errors", "├─ Errors()       <-chan error        // reconnect notices\n├─ Ticks()       <-chan Update       // never declared")},
			want:  "lists Subscription.Ticks, which pkg/ibkr/ws.go declares no such method",
		},
		{
			name:  "surface/the-diagram-loses-the-subscription-block",
			edits: []edit{replace(sdoc, "Subscription (public)", "Subscription internals")},
			want:  "the components diagram has no `Subscription (public)` block",
		},

		// --- 05: the buffer default --------------------------------------------
		{
			name:  "buffer/doc-changes-the-default",
			edits: []edit{replace(sdoc, "Per-subscription buffer default 256.", "Per-subscription buffer default 1024.")},
			want:  "documents a per-subscription buffer default of 1024",
		},
		{
			name:  "buffer/code-changes-the-limits-default",
			edits: []edit{replace(wsPub, "\t\tBufferSize:          256,", "\t\tBufferSize:          512,")},
			want:  "documents a per-subscription buffer default of 256",
		},
		{
			// The second copy. A subscription built with a non-positive buffer gets
			// the floor, not the limits default, so a change here is invisible to a
			// check that only read defaultStreamingLimits.
			name:  "buffer/code-changes-the-newSubscription-floor",
			edits: []edit{replace(wsPub, "\t\tbuffer = 256\n", "\t\tbuffer = 4096\n")},
			want:  "but the floor at pkg/ibkr/ws.go:146 is 4096",
		},
		{
			name:  "buffer/the-limits-default-disappears",
			edits: []edit{replace(wsPub, "\t\tBufferSize:          256,\n", "")},
			want:  "defaultStreamingLimits sets no BufferSize",
		},
		{
			name:  "buffer/the-claim-is-gone",
			edits: []edit{replace(sdoc, "Per-subscription buffer default 256.", "Buffering is configurable per subscription.")},
			want:  "no longer states a per-subscription buffer default",
		},

		// --- 05: ping, pong and the reconnect backoff -------------------------
		{
			name:  "timeout/doc-changes-the-ping",
			edits: []edit{replace(sdoc, "├─ ping goroutine     : 30s ping, 10s pong deadline", "├─ ping goroutine     : 45s ping, 10s pong deadline")},
			want:  "documents a ping interval of 45s but internal/ws.go installs 30s",
		},
		{
			name:  "timeout/doc-changes-the-pong",
			edits: []edit{replace(sdoc, "30s ping, 10s pong deadline", "30s ping, 20s pong deadline")},
			want:  "documents a pong deadline of 20s but internal/ws.go installs 10s",
		},
		{
			name:  "timeout/code-changes-the-ping",
			edits: []edit{replace(wsInt, "\t\topts.PingInterval = 30 * time.Second", "\t\topts.PingInterval = 15 * time.Second")},
			want:  "documents a ping interval of 30s but internal/ws.go installs 15s",
		},
		{
			name:  "timeout/code-drops-the-ping-default",
			edits: []edit{replace(wsInt, "\tif opts.PingInterval <= 0 {\n\t\topts.PingInterval = 30 * time.Second\n\t}\n", "")},
			want:  "DialWS has no `if opts.PingInterval <= 0` default",
		},
		{
			name:  "backoff/doc-changes-the-cap",
			edits: []edit{replace(sdoc, "Backoff: 1,2,4,8,16,30s cap, with jitter.", "Backoff: 1,2,4,8,16,60s cap, with jitter.")},
			want:  "reconnect backoff cap is 1m0s, but internal/ws.go installs 30s",
		},
		{
			name:  "backoff/doc-breaks-the-doubling",
			edits: []edit{replace(sdoc, "Backoff: 1,2,4,8,16,30s cap, with jitter.", "Backoff: 1,3,9,27,30s cap, with jitter.")},
			want:  "step 2 is 3s where doubling the previous step gives 2s",
		},
		{
			name:  "backoff/code-drops-the-cap",
			edits: []edit{replace(wsInt, "\td := base << attempt\n\tif d > max {\n\t\td = max\n\t}\n", "\td := base << attempt\n")},
			want:  "the ladder has no cap",
		},
		{
			name:  "backoff/code-stops-doubling",
			edits: []edit{replace(wsInt, "\td := base << attempt", "\td := base + attempt")},
			want:  "does not double the base delay",
		},
		{
			// Full jitter is what keeps every client from retrying in lockstep.
			name:  "backoff/code-drops-the-jitter",
			edits: []edit{replace(wsInt, "\treturn time.Duration(rand.Int63n(int64(d) + 1))", "\treturn d")},
			want:  "does not return a random value below the computed delay",
		},
		{
			name:  "backoff/doc-drops-the-ladder",
			edits: []edit{replace(sdoc, "Backoff: 1,2,4,8,16,30s cap, with jitter.", "Backoff: exponential, with jitter.")},
			want:  "no longer states a backoff ladder with a cap",
		},

		// --- 08: the redaction list ---------------------------------------------
		{
			name:  "redaction/code-drops-a-named-header",
			edits: []edit{replace(transp, "var headerRedact = regexp.MustCompile(`(?i)(Authorization|Cookie|Set-Cookie)\\s*:\\s*[^\\r\\n,;]*`)", "var headerRedact = regexp.MustCompile(`(?i)(Authorization|Set-Cookie)\\s*:\\s*[^\\r\\n,;]*`)")},
			want:  "says the transport redacts Cookie, but internal/transport.go's headerRedact alternation names only",
		},
		{
			name:  "redaction/redact-stops-applying-the-token-patterns",
			edits: []edit{replace(transp, "\ts = secretRedact.ReplaceAllString(s, \"<redacted>\")\n", "")},
			want:  "redact does not apply secretRedact",
		},
		{
			// The claim reaches *ibkr.Error.Message, which is where a caller reads a
			// token the gateway echoed back.
			name:  "redaction/the-error-message-stops-being-redacted",
			edits: []edit{replace(transp, "\t\tMessage:    redact(errMsg),", "\t\tMessage:    errMsg,")},
			want:  "sets Message to errMsg, not a redacted string",
		},
		{
			name:  "redaction/the-decoded-error-log-line-stops-redacting",
			edits: []edit{replace(obsv, "\t\t\"message\", redact(e.Message),", "\t\t\"message\", e.Message,")},
			want:  "the decoded-error log line does not pass redact(...) to the logger",
		},
		{
			name:  "redaction/the-transport-failure-log-line-stops-redacting",
			edits: []edit{replace(obsv, "append(attrs, \"err\", redact(err.Error()))", "append(attrs, \"err\", err.Error())")},
			want:  "the transport-failure log line does not pass redact(...) to the logger",
		},
		{
			name:  "redaction/the-sentence-is-gone",
			edits: []edit{replace(kdoc, "- **Redaction:** the transport redacts `Authorization`, `Cookie`, and token-bearing\n  headers from logs and from `*ibkr.Error.Message`.", "- **Redaction:** sensitive headers are handled carefully.")},
			want:  "no longer names the headers the transport redacts",
		},

		// --- 08: the insecure-skip-verify warning ------------------------------
		{
			name: "tls/the-standalone-guard-stops-warning",
			edits: []edit{replace(cclient,
				"\tif cfg.insecureSkipVerify && isNonLoopback(cfg.gatewayURL) {\n\t\tcfg.logger.Warn(\"ibkr.config insecure TLS skip-verify enabled for non-loopback host\",\n\t\t\t\"gateway\", cfg.gatewayURL)\n\t}\n", "")},
			want: "NewClient has no `if <config>.insecureSkipVerify && isNonLoopback(...)` guard",
		},
		{
			// A client from the transport pool is configured by a different function,
			// so a warning in NewClient alone would leave this path silent.
			name: "tls/the-pooled-guard-stops-warning",
			edits: []edit{replace(pool,
				"\tif cfg.insecureSkipVerify && isNonLoopback(cfg.gatewayURL) {\n\t\tif cfg.logger != nil {\n\t\t\tcfg.logger.Warn(\"ibkr.config insecure TLS skip-verify enabled for non-loopback host\",\n\t\t\t\t\"gateway\", cfg.gatewayURL)\n\t\t}\n\t}\n", "")},
			want: "NewTransportPool has no `if <config>.insecureSkipVerify && isNonLoopback(...)` guard",
		},
		{
			// A guard that fires for the option alone would warn on localhost, which
			// is the one host the option is for.
			name:  "tls/the-guard-drops-the-loopback-test",
			edits: []edit{replace(cclient, "if cfg.insecureSkipVerify && isNonLoopback(cfg.gatewayURL) {", "if cfg.insecureSkipVerify {")},
			want:  "has no `if <config>.insecureSkipVerify && isNonLoopback(...)` guard",
		},
		{
			name: "tls/the-guard-stops-logging",
			edits: []edit{replace(cclient,
				"\t\tcfg.logger.Warn(\"ibkr.config insecure TLS skip-verify enabled for non-loopback host\",\n\t\t\t\"gateway\", cfg.gatewayURL)\n", "")},
			want: "logs no warning; docs/design/08-concurrency.md:48 says the SDK",
		},
		{
			name:  "tls/the-option-stops-writing-the-field",
			edits: []edit{replace(cclient, "\t\tc.insecureSkipVerify = skip\n", "\t\t_ = skip\n")},
			want:  "WithInsecureSkipVerify writes no field of the config",
		},
		{
			name:  "tls/the-rule-is-gone",
			edits: []edit{replace(kdoc, "`WithInsecureSkipVerify(true)` is for\n  `localhost` only; the SDK warns if enabled against a non-loopback host.", "`WithInsecureSkipVerify(true)` is for\n  `localhost` only.")},
			want:  "no longer states that the SDK warns when skip-verify is enabled",
		},

		// --- 08: the OAuth generations -----------------------------------------
		{
			name:  "oauth/invalidation-stops-bumping-the-generation",
			edits: []edit{replace(oauth, "\tts.generation++\n", "")},
			want:  "invalidateLocked does not increment the generation counter",
		},
		{
			// The guard has to compare against the generation the fetch *started*
			// with. Dropping the capture means there is nothing to compare, and a
			// stale result would repopulate the cache the document says it cannot.
			name:  "oauth/the-generation-is-never-captured",
			edits: []edit{replace(oauth, "\tgeneration := ts.generation\n", "\tgeneration := uint64(0)\n")},
			want:  "Token does not read the generation counter into a local before fetching",
		},
		{
			name:  "oauth/the-cache-write-escapes-the-guard",
			edits: []edit{replace(oauth, "\tif ts.generation == generation {\n\t\tif err == nil {\n\t\t\tts.token = tok\n\t\t\tts.expiry = expiry\n\t\t\tif newRefresh != \"\" {\n\t\t\t\tts.refreshToken = newRefresh\n\t\t\t}\n\t\t}\n\t}\n", "\tif err == nil {\n\t\tts.token = tok\n\t\tts.expiry = expiry\n\t\tif newRefresh != \"\" {\n\t\t\tts.refreshToken = newRefresh\n\t\t}\n\t}\n")},
			want:  "has no `if ts.generation == generation` guard",
		},
		{
			// The refresh token is part of the cache the document says a stale
			// result must not repopulate, so it is checked with the other two.
			name:  "oauth/the-refresh-token-escapes-the-guard",
			edits: []edit{replace(oauth, "\t\t\tif newRefresh != \"\" {\n\t\t\t\tts.refreshToken = newRefresh\n\t\t\t}\n", "")},
			want:  "the generation guard does not assign ts.refreshToken",
		},
		{
			// The other half of the sentence: the result still reaches its caller, so
			// the guard has to suppress the cache write and nothing else. Putting a
			// return inside it turns a cache rule into a caller-visible early exit.
			name: "oauth/a-superseded-fetch-fails-its-caller",
			edits: []edit{replace(oauth,
				"\t\t\t\tts.refreshToken = newRefresh\n\t\t\t}\n\t\t}\n\t}\n\tts.mu.Unlock()\n",
				"\t\t\t\tts.refreshToken = newRefresh\n\t\t\t}\n\t\t}\n\t\treturn tok, nil\n\t}\n\tts.mu.Unlock()\n")},
			want: "Token returns from inside the generation guard",
		},
		{
			// The inverted guard: writing the cache precisely when the generation
			// differs is the exact opposite of the rule, and a check that only asked
			// for "a comparison against the captured generation" would accept it.
			name:  "oauth/the-generation-guard-is-inverted",
			edits: []edit{replace(oauth, "\tif ts.generation == generation {", "\tif ts.generation != generation {")},
			want:  "has no `if ts.generation == generation` guard",
		},
		{
			name:  "oauth/the-counter-is-gone",
			edits: []edit{replace(oauth, "\tgeneration   uint64\n", "")},
			want:  "TokenSource declares no generation field",
		},
		{
			name:  "oauth/the-rule-is-gone",
			edits: []edit{replace(kdoc, "- OAuth token invalidation uses generations; an in-flight result from an older\n  generation is returned to its caller but never repopulates the cache.", "- OAuth token invalidation is careful about staleness.")},
			want:  "no longer states the generations rule",
		},
	}
}

func wideGreenCases() []greenCase {
	return []greenCase{
		{
			// The reader parses each option line as the declaration it abbreviates,
			// so regrouping a parameter list has to be invisible to it. Without
			// this case the reader could be a comma-splitter that kept the leading
			// identifier, and every red case above would still go red — on a
			// document nobody had to get right.
			name:  "options/a-regrouped-parameter-list-passes",
			edits: []edit{replace(cdoc, "WithCircuitBreakerBudget(budget, size int)", "WithCircuitBreakerBudget(first int, second int)")},
		},
		{
			// The block's `// Core` / `// Resilience` headers are group labels, not
			// declarations, and a maintainer regroups under them freely.
			name:  "options/a-reworded-group-header-passes",
			edits: []edit{replace(cdoc, "// Resilience\n", "// Resilience and retries\n")},
		},
		{
			// A trailing comment on an option line is prose, the same way the
			// accessor list's `// IB REST (oauth2Bearer)` is.
			name:  "options/a-reworded-trailing-comment-passes",
			edits: []edit{replace(cdoc, "WithRetryPolicy(RetryPolicy)", "WithRetryPolicy(RetryPolicy)   // see DefaultRetryPolicy")},
		},
		{
			// A parameter's *name* is not a contract. The committed document
			// already writes `token` where pkg/ibkr/oauth.go writes `refreshToken`,
			// so the check must compare types and not names.
			name:  "options/a-renamed-parameter-passes",
			edits: []edit{replace(pubOauth, "func WithOAuth2RefreshToken(refreshToken string) Option {", "func WithOAuth2RefreshToken(tok string) Option {")},
		},
		{
			// The struct block writes the fifteen manager fields as comma lists
			// across four lines, and the fifteen names have to come out of that.
			// Reordering within a list is a real edit a maintainer makes when they
			// regroup, and it must not change the field set.
			name: "struct/reordering-a-comma-list-passes",
			edits: []edit{replace(cdoc,
				"    sessionManager, accountManager, portfolioManager, tradeManager,\n",
				"    portfolioManager, sessionManager, accountManager, tradeManager,\n")},
		},
		{
			// A note name the block *also* lists is redundant rather than stale, and
			// the check must not report it: the field is documented either way, so
			// it is neither an undeclared field nor an omission that can let one
			// through. That is the branch at client_composition.go's
			// `if allowed[name] { continue }`.
			//
			// The case no longer describes the committed document. The human
			// corrected the note on 2026-09-27 — it used to omit `wsMu` and
			// `restMu` while the block above listed both — so the contradiction is
			// reintroduced here in the scratch copy to keep the tolerance pinned
			// rather than deleted along with the defect.
			//
			// The tolerance was confirmed load-bearing rather than assumed: making
			// that branch report a redundant name instead of skipping it turns this
			// case red with `note redundantly names "wsMu"`, and the probe also
			// shows the mutated name reaches the checker rather than sitting in the
			// document unread.
			//
			// What the case cannot pin is the branch's *absence*. Deleting it
			// leaves this case green, because `wsMu` is declared either way, so both
			// the allowance and the stale-note comparison come out the same.
			name: "struct/the-contradictory-omission-note-passes",
			edits: []edit{replace(cdoc,
				"> **Note**: The struct diagram shows all significant fields. The pool\n> `release` callback is omitted for clarity.",
				"> **Note**: The struct diagram shows all significant fields. The internal\n> synchronization field `wsMu` and the pool `release` callback are\n> omitted for clarity.")},
		},
		{
			// The REST line carries a trailing `//` comment, and the block carries a
			// prose comment after each manager. A reader that did not cut the comment
			// would read the last result as "(oauth2Bearer)" and reject a correct
			// document.
			name: "accessor/a-reworded-trailing-comment-passes",
			edits: []edit{replace(cdoc,
				"func (c *Client) REST() (*RESTSurface, error)   // IB REST (oauth2Bearer)",
				"func (c *Client) REST() (*RESTSurface, error)   // the IB REST surface (oauth2Bearer)")},
		},
		{
			// An unexported field may be a generated type: reaching it would take a
			// caller writing to this package. Client.generated is exactly that.
			name:  "boundary/an-unexported-generated-field-passes",
			edits: []edit{replace(cclient, "\tgenerated  *client.ClientWithResponses\n", "\tgenerated  *client.ClientWithResponses // an unexported field may hold a generated type\n")},
		},
		{
			// An unexported function may build a generated value; what it must not
			// do is return one.
			name: "boundary/an-unexported-signature-passes",
			edits: []edit{replace(cclient,
				"func (c *Client) errorFrom(resp *http.Response, op string) *Error {",
				"func (c *Client) errorFrom(resp *http.Response, op string) *Error {\n\t_ = client.PortfolioPositionsResponse{}")},
		},
		{
			// The component diagram omits Close's result, so only the method's
			// existence is claimed for it. Close returning an error is a recorded
			// difference, not a failure.
			name:  "surface/close-keeps-its-error-result",
			edits: []edit{replace(wsPub, "func (s *Subscription) Close() error {", "func (s *Subscription) Close() error {\n\t// still returns an error; the diagram elides the result")},
		},
		{
			// A package under internal/ that does not touch the generated code is
			// not a boundary violation, and must not be reported as one.
			name:  "boundary/an-internal-package-need-not-import",
			edits: []edit{replace("internal/fake/fake.go", "package fake", "package fake\n\n// this package deliberately does not import the generated code")},
		},
		{
			// The buffer default is a claim about the number, not about the field it
			// lives in; a comment can be reworded freely.
			name:  "buffer/a-reworded-comment-passes",
			edits: []edit{replace(sdoc, "Per-subscription buffer default 256.", "Per-subscription buffer default 256 (256 slots).")},
		},
	}
}
