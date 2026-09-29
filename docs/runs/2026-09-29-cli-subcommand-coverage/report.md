# Report: four bugs behind a seam that was never replaced

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `f95f4c4` (`v1.1.23`)
- **Outcome**: complete

`cmd/ibkr` coverage went 33.0% -> 76.1%; total 66.7% -> 69.5% against a 62% floor.
Four real bugs turned up, and the biggest one was not where the backlog said to look.

## The backlog item was wrong, and that is what started this

`v1.1.24`'s `next-phase.md` carried this from `breaker-clock`:

> **`cmd/ibkr` at 33%** - the largest number, and a design change: every non-help path
> builds a live client, so it needs a client-factory seam first.

The seam existed since 1.1.14. `cmd/ibkr/env.go` carries

```go
// newClient builds the SDK client. Production reads the global flags and the
// config file; a test substitutes a client wired to the mock gateway.
newClient func() (*ibkr.Client, error)
```

doc-commented as *the seam a test replaces*, with all 9 call sites going through it. So
the recorded blocker - build the seam - was a year-old fix, and the actual gap was that no
test ever called it. That is a test-writing job, not a design change, and the difference is
the difference between a bounded task and an architectural one.

The same backlog also listed a `stream.go` gap naming a file that does not exist and two
functions that are not defined anywhere. Two of five inherited items were wrong on arrival.

## Global flags were never read at all

The first harness attempt connected to `https://localhost:5000` instead of the mock. That
was not a harness mistake, and chasing it found the most serious bug in the run:

```go
// cmd/ibkr/main.go
gateway, rest, account, insecure, _ := parseGlobalFlags(args)
```

`parseGlobalFlags` was only ever called as `parseGlobalFlags(e.args)`, and its `default`
branch returns on the first argument that is not a global flag. `e.args[0]` is the program
name, so the loop returned at `i=0` and read nothing.

| Invocation | Behaviour before this run |
|------------|---------------------------|
| `ibkr --gateway URL accounts` | `unknown command "--gateway"` |
| `ibkr accounts --gateway URL` | runs, connects to `https://localhost:5000` |
| `ibkr accounts` | same |

So `--gateway`, `--rest`, `--insecure` and global `--account` were all inert, and a CLI
whose entire purpose is talking to a gateway could only be pointed elsewhere by editing
`~/.ibkr/config.json`. The parser's own tests pass because they call it with argv *minus*
the program name - `[]string{"-gateway", "https://g:1", "orders"}` - so the shape that
ships was never the shape under test.

The fix changes the seam to `newClient func(args []string)` and has each of the 9 call
sites pass the subcommand's own arguments, which are the ones the documented form
`ibkr <command> [subcommand] [flags]` puts the flags in. Subcommand depth stops mattering,
and `-account` stops being ambiguous between a global flag and the per-subcommand flag
each of `positions`, `portfolio` and `orders` parses for itself.

### `--account` was ignored by the order commands

`runOrdersSubmit` and `runOrdersCancel` both called `mustAccount("")` - they never parsed
`-account` at all, unlike every other command. So `ibkr orders submit --account U999 ...`
was accepted, printed no complaint, and placed the order against whatever `account_id`
the config held, or failed with `account ID required` when the config had none. Both now
parse the flag and pass it to `mustAccount`.

## Two more bugs, in the subcommands themselves

- **`ibkr portfolio` panicked with no arguments.** `runPortfolio` did `args[1:]` inside
  its `len(args) == 0` branch - `slice bounds out of range [1:0]`. `runOrders` has the
  identical shape four lines above and handles it correctly. Now `nil`.
- **`orders cancel` wrote to the process stdout.** `fmt.Printf("order %s cancelled\n", ...)`
  where every other message in the CLI goes through `e.stdout` via `ibkrPrintln` or
  `fmt.Fprintf`. The confirmation was uncapturable, invisible to anything consuming the
  command's output, and the only output path in the CLI that ignored the `env` contract.

## Three findings filed rather than fixed

- **~2.0s fixed latency per command** against a mock that answers instantly:
  `Session.Initialize` 1.007s, `cli.Close` 995ms. The mock's recorder shows only 4
  requests and the auth poll succeeding on the first try, so this is *not* the one-second
  poll penalty an earlier run fixed in `internal/session.go` - it is in the tickle-start
  and logout paths. Not root-caused. This is why the package's tests go 0.4s -> 22.9s.
- **`parseGlobalFlags` returns `cmdIdx` one past** the first non-flag, while its doc
  comment and `TestParseGlobalFlags_StopsAtFirstNonFlag`'s comment both say "the index of".
  No production caller uses it, so it is latent; found by writing a test whose first
  expectation was wrong.
- **`portfolio ledger` emits Go field names** (`NetLiquidationValue`) because
  `LedgerCurrency` has no json tags, unlike every other JSON-emitting command.

## `--rest` is left inert, deliberately

`--rest` is parsed into `cfg.RestGatewayURL`, settable through `ibkr config set`, printed
by `ibkr config`, and never used: `pkg/ibkr` exposes no option to receive it, and
`WithGatewayURL` covers both API surfaces per ADR 0001. It is left as-is with a comment at
the point of the omission, because the honest fix is an SDK option or removing the flag -
not a second dead assignment in the CLI. Wiring it would have made a gate look satisfied
while the behaviour stayed wrong.

## Two corrections to my own work

- The first test for the `--account` fix installed a fake `newClient` that called
  `mustAccount("U999")` with the expected value hardcoded. It could not fail. Replaced
  with one that asserts *which error comes back*: past `mustAccount` means a request
  error, stuck in it means the flag was ignored.
- I moved a dead test server's `Close` into `t.Cleanup`, which left the port listening.
  Requests then hung to the CLI's 15s context deadline instead of being refused, and 7
  cases took 128s instead of 22.5s. The `Close` before the test runs is the load-bearing
  part, and the comment now says so.

## What this run is really evidence of

Two of the five inherited backlog items were wrong, one of them describing a fix that
shipped a year earlier. An unverified backlog is worse than a short one, because it looks
like work. Every item here was checked against the tree before being acted on, and the one
that changed the shape of the task was the one that had been carried longest.
