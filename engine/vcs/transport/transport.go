// Package transport provides the HTTP transports the VCS providers use to
// reach a forge.
//
// A provider client is built per request, so it cannot own the connection
// pool. The transports here are process-wide and outlive any single request;
// only the credentials belong to the client above them.
package transport

import (
	"net/http"
	"sync/atomic"
	"time"
)

const (
	// DefaultPoolSize is the number of transports used when
	// vcs.forge.connectionPoolSize is unset.
	//
	// Each transport holds its own connections, so this is the number of
	// connections concurrent requests can spread over. A single transport
	// gives an HTTP/2 forge one connection carrying every request.
	DefaultPoolSize = 8

	// DefaultMaxIdleConnsPerHost is the number of idle connections each
	// transport keeps per host when vcs.forge.maxIdleConnsPerHost is unset.
	// The standard library's default is 2.
	DefaultMaxIdleConnsPerHost = 32

	idleConnTimeout = 90 * time.Second
)

// pool is replaced wholesale by Configure and never mutated in place, so a
// concurrent Pooled always reads a fully built pool.
var (
	pool atomic.Pointer[[]*http.Transport]
	next atomic.Uint64
)

func init() {
	Configure(DefaultPoolSize, DefaultMaxIdleConnsPerHost)
}

// Configure builds the pool from the configured sizes. A size that is zero or
// negative falls back to its default.
//
// Call it before any provider builds a client.
func Configure(poolSize, maxIdleConnsPerHost int) {
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}
	if maxIdleConnsPerHost <= 0 {
		maxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}

	transports := make([]*http.Transport, poolSize)
	for i := range transports {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConns = maxIdleConnsPerHost * poolSize
		t.MaxIdleConnsPerHost = maxIdleConnsPerHost
		t.IdleConnTimeout = idleConnTimeout
		transports[i] = t
	}
	pool.Store(&transports)
}

// Pooled returns the next transport of the pool, in rotation. Each transport
// is safe for concurrent use; rotating spreads concurrent requests over the
// pool's connections instead of one.
func Pooled() http.RoundTripper {
	transports := *pool.Load()
	return transports[int(next.Add(1)%uint64(len(transports)))]
}

// size reports how many transports the pool holds.
func size() int {
	return len(*pool.Load())
}
