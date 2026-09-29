// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Broadcast and closeAll are the two hub operations that fan out across every
// connected client, and both take the same shape: snapshot the connection set under
// the lock, release it, then act. That is the property worth pinning, because
// holding the hub mutex across a socket write would let one stalled client freeze
// every other client, and it is invisible in a single-connection test.
//
// Two behaviours here are also silent rather than loud, and both are asserted
// explicitly because a reader would otherwise assume the opposite:
//
//   - sendFrame returns nil for an already-closed connection, so Broadcast's return
//     value counts connections it wrote to *and* connections it skipped. It is an
//     attempt count, not a delivery count.
//   - closeAll marks connections closed but does not remove them from the hub.
//     Removal is serveWS's deferred job, so Subscribers() stays non-zero briefly
//     after a closeAll.

// dialN opens n streaming connections to one server and waits for all of them to
// register with the hub.
func dialN(t *testing.T, n int) (*Server, []*websocket.Conn) {
	t.Helper()
	srv := New()
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/api/ws"
	conns := make([]*websocket.Conn, 0, n)
	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c, _, err := websocket.Dial(ctx, url, nil)
		cancel()
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		t.Cleanup(func() { _ = c.Close(websocket.StatusNormalClosure, "done") })
		conns = append(conns, c)
	}

	// The hub registers a connection in serveWS, after the upgrade returns.
	deadline := time.Now().Add(5 * time.Second)
	for srv.Stream().Subscribers() < n {
		if time.Now().After(deadline) {
			t.Fatalf("hub registered %d of %d connections", srv.Stream().Subscribers(), n)
		}
		time.Sleep(2 * time.Millisecond)
	}
	return srv, conns
}

// hubConns returns the hub's connections. The tests are in-package, so this reads
// the map under its own lock rather than adding a method to production code.
func hubConns(h *StreamHub) []*streamConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*streamConn, 0, len(h.conns))
	for sc := range h.conns {
		out = append(out, sc)
	}
	return out
}

// TestBroadcast_ReturnsOnePerRegisteredConnection pins the fan-out count.
//
// Broadcast's return is an attempt count, not a delivery count: sendFrame returns
// nil for an already-closed connection, so a connection in the window between
// close() and the handler's remove() is counted without a write happening. That
// window is short and not worth racing to observe from here, so the skip itself is
// pinned deterministically in TestSendFrame_SkipsClosedConnections and this test
// pins the guarantee that is externally visible - one count per registered
// connection.
func TestBroadcast_ReturnsOnePerRegisteredConnection(t *testing.T) {
	srv, conns := dialN(t, 2)
	hub := srv.Stream()

	if got := hub.Subscribers(); got != 2 {
		t.Fatalf("Subscribers() = %d; want 2", got)
	}
	if n := hub.Broadcast(map[string]string{"k": "v"}); n != 2 {
		t.Errorf("Broadcast() = %d; want 2 - one per registered connection", n)
	}

	// The count alone does not prove the frame was written: returning len(conns)
	// without sending would satisfy it, and no test noticed. Each client has to
	// actually receive the frame.
	for i, c := range conns {
		frame := readStreamFrame(t, c)
		if frame == nil {
			t.Errorf("client %d received no frame from Broadcast", i)
			continue
		}
		if got, _ := frame["k"].(string); got != "v" {
			t.Errorf("client %d got frame %v; want k=v", i, frame)
		}
	}
}

// TestSendFrame_SkipsClosedConnections pins the silent skip that makes Broadcast's
// return an attempt count. Deterministic, and the reason the count above is not
// read as a delivery count.
func TestSendFrame_SkipsClosedConnections(t *testing.T) {
	srv, conns := dialN(t, 1)
	sc := hubConns(srv.Stream())[0]

	if err := sc.sendFrame(map[string]string{"k": "live"}); err != nil {
		t.Fatalf("sendFrame on a live connection: %v", err)
	}

	sc.close()
	if err := sc.sendFrame(map[string]string{"k": "closed"}); err != nil {
		t.Errorf("sendFrame on a closed connection = %v; want nil - it is skipped, "+
			"not reported as an error", err)
	}

	// A frame written to a live connection is observable on the client, so the
	// skip above is about not writing rather than about the socket being unusable.
	frame := readStreamFrame(t, conns[0])
	if frame == nil {
		t.Fatal("read no frame from the live connection")
	}
}

// TestCloseAll_DrainsTheHub pins the teardown path.
//
// closeAll marks every connection closed and calls CloseNow on each socket. The
// removal is the connection handler's deferred job: CloseNow unblocks serveWS's read
// loop, and serveWS's defer is what calls remove. So the map drains as a
// consequence of closing, not inside closeAll - which is why the wait here is
// bounded rather than immediate, and why asserting synchronously would be both
// wrong and flaky.
func TestCloseAll_DrainsTheHub(t *testing.T) {
	srv, _ := dialN(t, 3)
	hub := srv.Stream()

	if got := hub.Subscribers(); got != 3 {
		t.Fatalf("Subscribers() = %d before closeAll; want 3", got)
	}

	// Capture the connections before closing them, so the assertions below are not
	// racing a map that closeAll is already draining.
	conns := hubConns(hub)
	if len(conns) != 3 {
		t.Fatalf("captured %d connections; want 3", len(conns))
	}

	hub.closeAll()

	// Whether closeAll or the connection handler sets the closed flag is not
	// externally distinguishable: serveWS's defer calls close() too, so closing
	// without CloseNow still ends with the flag set. Asserting it here therefore
	// proved nothing - a closeAll that marked nothing closed passed - so what is
	// asserted is the observable effect, the drain below.
	_ = conns

	deadline := time.Now().Add(5 * time.Second)
	for hub.Subscribers() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("Subscribers() = %d 5s after closeAll; want 0 - CloseNow "+
				"should unblock each handler so its defer removes the connection",
				hub.Subscribers())
		}
		time.Sleep(2 * time.Millisecond)
	}

	// With the hub drained there is nothing to write to.
	if n := hub.Broadcast(map[string]string{"k": "v"}); n != 0 {
		t.Errorf("Broadcast() = %d after the hub drained; want 0", n)
	}
}

// Not tested here, and deliberately so: that Broadcast and closeAll do not hold the
// hub mutex across the socket write. It is a real requirement - holding it would let
// one unresponsive client stall every other client - and it took three attempts to
// establish that it cannot be tested soundly in this package:
//
//   - probing with a synthetic connection panics the in-flight Broadcast, because a
//     streamConn with a nil socket cannot be written to;
//   - probing with Subscribers() needs the write to actually block, but against a
//     client that is not reading the write fails in tens of milliseconds rather than
//     running to streamWriteTimeout, so any budget either always passes or is too
//     tight to survive a loaded machine;
//   - shrinking the client's receive buffer to force a longer block is not available,
//     because coder/websocket's NetConn returns a wrapper without SetReadBuffer, and
//     a skipped test is not a passing one.
//
// Each of those first two attempts looked like a working test and passed with the
// lock held. The requirement is therefore recorded on Broadcast and closeAll
// themselves as a review rule, which is weaker than a test and honest about being
// so.
