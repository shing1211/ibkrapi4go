# Report: the stream hub, and a test that could not be written

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `3573421` (`v1.1.20`)
- **Outcome**: complete, uncommitted pending approval

## Two premises that were wrong

Both found by running the tests rather than by reading the code.

**`closeAll` does not leave connections registered.** The first draft asserted it
does - "flags connections closed but does not remove them" - on the grounds that
`closeAll` only calls `sc.close()` and `conn.CloseNow()`. It also calls `CloseNow`,
which unblocks `serveWS`'s read loop, and `serveWS`'s **defer** is what calls
`remove`. The hub drains as a consequence of closing, not inside `closeAll`. Reading
one function's body and stopping was the mistake; the handler on the other side of
the socket is part of the same behaviour.

**`Broadcast`'s return is an attempt count, and a test of it proved nothing.** A
mutation that made `Broadcast` return `len(conns)` without sending anything was not
caught, because from outside "counted" and "delivered" are indistinguishable. The
test now has each client read the frame.

## What is tested

`stream_hub_test.go`, three behaviours, all externally observable:

- **Fan-out** - the count is one per registered connection, *and* each client
  receives the frame. The delivery half is what caught the count-only mutation.
- **The silent skip** - `sendFrame` on a closed connection returns nil rather than
  an error, so `Broadcast`'s count is an attempt count and not a delivery count.
  Pinned at the `streamConn` level, where it is deterministic.
- **The drain** - after `closeAll`, the hub reaches zero subscribers and a subsequent
  `Broadcast` returns 0.

Four mutations, all caught on the assertion:

| Mutation | Caught by |
|---|---|
| `Broadcast` skips a connection | `Broadcast() = 1; want 2` |
| `sendFrame` errors on a closed connection | `= connection closed; want nil` |
| `closeAll` never closes the socket | `Subscribers() = 3 5s after closeAll; want 0` |
| `Broadcast` counts without writing | client received no frame |

## The test that could not be written

The lock discipline is the real requirement here: snapshot under the lock, act
outside it. Holding the mutex across the socket write would let one unresponsive
client stall every other client. It is also the property most likely to be broken by
a well-meaning refactor, since the two versions look equivalent.

Three attempts, and **the first two passed with the lock deliberately held**:

1. **Probe with a synthetic connection.** Registering a `streamConn` while a
   broadcast was in flight, to show `add` did not block. This panicked: the
   synthetic connection has a nil socket, and the in-flight `Broadcast` wrote to it.
2. **Probe with `Subscribers()` and a large payload.** 256 KiB first - the kernel
   absorbed it, nothing ever blocked, and the mutation passed. 8 MiB against clients
   that never read did make the write block, but only for tens of milliseconds
   rather than to `streamWriteTimeout` (5s), so a 3s probe budget still passed with
   the lock held. Tightening the budget to 500ms still passed, because the block is
   ~80ms - a budget under that is too tight to survive a loaded machine.
3. **Force a longer block by shrinking the client's receive buffer.** Not available:
   `coder/websocket`'s `NetConn` returns a wrapper that does not expose
   `SetReadBuffer`, so the test skipped. A skipped test is not a passing one.

So the requirement is recorded on `Broadcast` and `closeAll` themselves as a review
rule, with the reason it is not tested, and the three dead ends, in the source. That
is weaker than a test and is labelled as such.

**This is the fourth time in this sequence that a plausible-looking test turned out
to be vacuous** - after the money assertions on `float32`-exact values, the proposed
`check_money.py` gate, the `float32Loses` check on `0.007`, and now the two lock
tests. The pattern is consistent enough to be worth stating: a test that has never
been run against a mutation of the code it claims to check is a hypothesis. Every
one of these passed in normal use.

## Coverage

`internal/mockgateway` 86.6% -> 87.5%. Overall 66.3% -> **66.5%** against the 62%
floor. The headline is small because the package is; the value is three pinned
behaviours and one documented gap.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`,
`check_design`, `check_links.py`, `check_i18n.py`.

The `unused` linter caught a `blockingPayload` constant left behind when the lock
tests were deleted - dead code in the change under review, which is the `--tests=false`
gate doing its job.

## Production code

Comments only, on `Broadcast` and `closeAll`: the lock discipline, the reason it
exists, and why it is not enforced by a test. No behaviour changed.
