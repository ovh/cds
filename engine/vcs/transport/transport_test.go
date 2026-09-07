package transport

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPooledRotates checks Pooled hands out every transport in the pool rather
// than the same one each time.
func TestPooledRotates(t *testing.T) {
	seen := make(map[http.RoundTripper]struct{})
	for i := 0; i < PoolSize*3; i++ {
		seen[Pooled()] = struct{}{}
	}
	require.Len(t, seen, PoolSize, "every transport in the pool must be handed out")
}

// TestPooledOpensSeveralConnectionsToAnHTTP2Host checks the pool spreads a
// burst of concurrent requests over several TCP connections, where a single
// transport carries the same burst on one.
//
// The lanes are warmed before the connections are counted. A transport
// dialling a cold host opens one connection per concurrent request whatever
// its configuration, so an unwarmed run passes without proving anything.
func TestPooledOpensSeveralConnectionsToAnHTTP2Host(t *testing.T) {
	var mu sync.Mutex
	conns := make(map[string]struct{})

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(50 * time.Millisecond)
		}
		fmt.Fprint(w, "ok")
	}))
	srv.EnableHTTP2 = true
	srv.Config.ConnState = func(c net.Conn, state http.ConnState) {
		if state != http.StateActive {
			return
		}
		mu.Lock()
		conns[c.RemoteAddr().String()] = struct{}{}
		mu.Unlock()
	}
	srv.StartTLS()
	defer srv.Close()

	// The pool's transports do not trust the test server's certificate, so
	// lend each of them the server's TLS config for the duration.
	serverTLS := srv.Client().Transport.(*http.Transport).TLSClientConfig
	for _, tr := range pool {
		restore := tr.TLSClientConfig
		tr.TLSClientConfig = serverTLS.Clone()
		defer func(tr *http.Transport, cfg *tls.Config) { tr.TLSClientConfig = cfg }(tr, restore)
	}

	get := func(rt http.RoundTripper, path string) {
		resp, err := (&http.Client{Transport: rt}).Get(srv.URL + path)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	countAfterBurst := func(rt func(i int) http.RoundTripper) int {
		for i := 0; i < PoolSize; i++ {
			get(rt(i), "/warm")
		}
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		conns = make(map[string]struct{})
		mu.Unlock()

		var wg sync.WaitGroup
		for i := 0; i < PoolSize*3; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				get(rt(i), "/slow")
			}(i)
		}
		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		return len(conns)
	}

	shared := pool[0]
	require.Equal(t, 1, countAfterBurst(func(int) http.RoundTripper { return shared }),
		"a single transport must carry the whole burst on one connection")

	transports := pool
	require.Greater(t, countAfterBurst(func(i int) http.RoundTripper { return transports[i%len(transports)] }), 1,
		"the pool must spread a burst over several connections")
}
