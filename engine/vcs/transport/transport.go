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
	// PoolSize is the number of transports kept for reaching a forge.
	//
	// Each transport holds its own connections, so this is the number of
	// connections concurrent requests can spread over. A single transport
	// gives an HTTP/2 forge one connection carrying every request.
	PoolSize = 8

	// maxIdleConnsPerHost is the number of idle connections each transport
	// keeps per host. The standard library's default is 2.
	maxIdleConnsPerHost = 32

	idleConnTimeout = 90 * time.Second
)

var (
	pool = newPool()
	next atomic.Uint64
)

func newPool() []*http.Transport {
	transports := make([]*http.Transport, PoolSize)
	for i := range transports {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConns = maxIdleConnsPerHost * PoolSize
		t.MaxIdleConnsPerHost = maxIdleConnsPerHost
		t.IdleConnTimeout = idleConnTimeout
		transports[i] = t
	}
	return transports
}

// Pooled returns the next transport of the pool, in rotation. Each transport
// is safe for concurrent use; rotating spreads concurrent requests over the
// pool's connections instead of one.
func Pooled() http.RoundTripper {
	return pool[int(next.Add(1)%uint64(PoolSize))]
}
