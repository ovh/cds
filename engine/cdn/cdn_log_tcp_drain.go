package cdn

import (
	"net"
	"sync"
)

// Shutting the log intake down.
//
// Closing the listener only stops new connections. The ones already accepted are read by a
// goroutine each, and nothing used to tell them to stop: they stayed blocked on their read until
// the process died under them. A worker learns that its connection is gone from its next write
// failing, so the earlier and the cleaner that close happens, the sooner it reconnects to an
// instance that can serve it, and the fewer lines it writes into a socket nobody reads any more.
//
// This does not save the lines already in flight. It only makes the break immediate and
// unambiguous instead of leaving the client to discover it, or not.

// openLogConns holds the connections currently being read, so that a shutdown can close them.
// It is a package variable for the same reason globalRateLimit is one: the intake owns exactly
// one of each for the whole process.
var openLogConns = newConnRegistry()

type connRegistry struct {
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func newConnRegistry() *connRegistry {
	return &connRegistry{conns: make(map[net.Conn]struct{})}
}

func (r *connRegistry) add(c net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[c] = struct{}{}
}

func (r *connRegistry) remove(c net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, c)
}

func (r *connRegistry) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.conns)
}

// closeAll closes every connection it holds and returns how many it closed. The registry is
// emptied: a connection closed here still runs its own deferred close, which must not find it.
func (r *connRegistry) closeAll() int {
	r.mu.Lock()
	conns := make([]net.Conn, 0, len(r.conns))
	for c := range r.conns {
		conns = append(conns, c)
	}
	r.conns = make(map[net.Conn]struct{})
	r.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}
	return len(conns)
}
