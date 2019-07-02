// Package httpcontrol allows a HTTP transport supporting connection pooling,
// timeouts & retries.
//
// This Transport is built on top of the standard library transport and
// augments it with additional features.
package httpcontrol

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Stats for a RoundTrip.
type Stats struct {
	// The RoundTrip request.
	Request *http.Request

	// May not always be available.
	Response *http.Response

	// Will be set if the RoundTrip resulted in an error. Note that these are
	// RoundTrip errors and we do not care about the HTTP Status.
	Error error

	// Each duration is independent and the sum of all of them is the total
	// request duration. One or more durations may be zero.
	Duration struct {
		Header, Body time.Duration
	}

	Retry struct {
		// Will be incremented for each retry. The initial request will have this
		// set to 0, and the first retry to 1 and so on.
		Count uint

		// Will be set if and only if an error was encountered and a retry is
		// pending.
		Pending bool
	}
}

// A human readable representation often useful for debugging.
func (s *Stats) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %s", s.Request.Method, s.Request.URL)

	if s.Response != nil {
		fmt.Fprintf(&buf, " got response with status %s", s.Response.Status)
	}

	return buf.String()
}

// Transport is an implementation of RoundTripper that supports http, https,
// and http proxies (for either http or https with CONNECT). Transport can
// cache connections for future re-use, provides various timeouts, retry logic
// and the ability to track request statistics.
type Transport struct {

	// Proxy specifies a function to return a proxy for a given
	// *http.Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	// If Proxy is nil or returns a nil *url.URL, no proxy is used.
	Proxy func(*http.Request) (*url.URL, error)

	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client. If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// DisableKeepAlives, if true, prevents re-use of TCP connections
	// between different HTTP requests.
	DisableKeepAlives bool

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) to keep per-host.  If zero,
	// http.DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int

	// Dial connects to the address on the named network.
	//
	// See func Dial for a description of the network and address
	// parameters.
	Dial func(network, address string) (net.Conn, error)

	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete.
	//
	// The default is no timeout.
	//
	// With or without a timeout, the operating system may impose
	// its own earlier timeout. For instance, TCP timeouts are
	// often around 3 minutes.
	DialTimeout time.Duration

	// DialKeepAlive specifies the keep-alive period for an active
	// network connection.
	// If zero, keep-alives are not enabled. Network protocols
	// that do not support keep-alives ignore this field.
	DialKeepAlive time.Duration

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration

	// RequestTimeout, if non-zero, specifies the amount of time for the entire
	// request. This includes dialing (if necessary), the response header as well
	// as the entire body.
	RequestTimeout time.Duration

	// RetryAfterTimeout, if true, will enable retries for a number of failures
	// that are probably safe to retry for most cases but, depending on the
	// context, might not be safe. Retried errors: net.Errors where Timeout()
	// returns `true` or timeouts that bubble up as url.Error but were originally
	// net.Error, OpErrors where the request was cancelled (either by this lib or
	// by the calling code, or finally errors from requests that were cancelled
	// before the remote side was contacted.
	RetryAfterTimeout bool

	// MaxTries, if non-zero, specifies the number of times we will retry on
	// failure. Retries are only attempted for temporary network errors or known
	// safe failures.
	MaxTries uint

	// Stats allows for capturing the result of a request and is useful for
	// monitoring purposes.
	Stats func(*Stats)

	startOnce sync.Once
	transport *http.Transport
}

var knownFailureSuffixes = []string{
	syscall.ECONNREFUSED.Error(),
	syscall.ECONNRESET.Error(),
	syscall.ETIMEDOUT.Error(),
	"no such host",
	"remote error: handshake failure",
	io.ErrUnexpectedEOF.Error(),
	io.EOF.Error(),
}

func (t *Transport) shouldRetryError(err error) bool {
	if neterr, ok := err.(net.Error); ok {
		if neterr.Temporary() {
			return true
		}
	}

	if t.RetryAfterTimeout {
		if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
			return true
		}

		// http://stackoverflow.com/questions/23494950/specifically-check-for-timeout-error
		if urlerr, ok := err.(*url.Error); ok {
			if neturlerr, ok := urlerr.Err.(net.Error); ok && neturlerr.Timeout() {
				return true
			}
		}
		if operr, ok := err.(*net.OpError); ok {
			if strings.Contains(operr.Error(), "use of closed network connection") {
				return true
			}
		}

		// The request timed out before we could connect
		if strings.Contains(err.Error(), "request canceled while waiting for connection") {
			return true
		}
	}

	s := err.Error()
	for _, suffix := range knownFailureSuffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// Start the Transport.
func (t *Transport) start() {
	if t.Dial == nil {
		dialer := &net.Dialer{
			Timeout:   t.DialTimeout,
			KeepAlive: t.DialKeepAlive,
		}
		t.Dial = dialer.Dial
	}
	t.transport = &http.Transport{
		Dial:                  t.Dial,
		Proxy:                 t.Proxy,
		TLSClientConfig:       t.TLSClientConfig,
		DisableKeepAlives:     t.DisableKeepAlives,
		DisableCompression:    t.DisableCompression,
		MaxIdleConnsPerHost:   t.MaxIdleConnsPerHost,
		ResponseHeaderTimeout: t.ResponseHeaderTimeout,
	}
}

// CloseIdleConnections closes the idle connections.
func (t *Transport) CloseIdleConnections() {
	t.startOnce.Do(t.start)
	t.transport.CloseIdleConnections()
}

// CancelRequest cancels an in-flight request by closing its connection.
func (t *Transport) CancelRequest(req *http.Request) {
	t.startOnce.Do(t.start)
	if bc, ok := req.Body.(*bodyCloser); ok {
		bc.timer.Stop()
	}
	t.transport.CancelRequest(req)
}

func (t *Transport) tries(req *http.Request, try uint) (*http.Response, error) {
	startTime := time.Now()
	var timer *time.Timer
	if t.RequestTimeout != 0 {
		timer = time.AfterFunc(t.RequestTimeout, func() {
			t.CancelRequest(req)
		})
	}
	res, err := t.transport.RoundTrip(req)
	headerTime := time.Now()
	if err != nil {
		if timer != nil {
			timer.Stop()
		}
		var stats *Stats
		if t.Stats != nil {
			stats = &Stats{
				Request:  req,
				Response: res,
				Error:    err,
			}
			stats.Duration.Header = headerTime.Sub(startTime)
			stats.Retry.Count = try
		}

		if try < t.MaxTries && req.Method == "GET" && t.shouldRetryError(err) {
			if t.Stats != nil {
				stats.Retry.Pending = true
				t.Stats(stats)
			}
			return t.tries(req, try+1)
		}

		if t.Stats != nil {
			t.Stats(stats)
		}
		return nil, err
	}

	res.Body = &bodyCloser{
		ReadCloser: res.Body,
		timer:      timer,
		res:        res,
		transport:  t,
		startTime:  startTime,
		headerTime: headerTime,
	}
	return res, nil
}

// RoundTrip implements the RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.startOnce.Do(t.start)
	return t.tries(req, 0)
}

type bodyCloser struct {
	io.ReadCloser
	timer      *time.Timer
	res        *http.Response
	transport  *Transport
	startTime  time.Time
	headerTime time.Time
}

func (b *bodyCloser) Close() error {
	if b.timer != nil {
		b.timer.Stop()
	}
	err := b.ReadCloser.Close()
	closeTime := time.Now()
	if b.transport.Stats != nil {
		stats := &Stats{
			Request:  b.res.Request,
			Response: b.res,
		}
		stats.Duration.Header = b.headerTime.Sub(b.startTime)
		stats.Duration.Body = closeTime.Sub(b.startTime) - stats.Duration.Header
		b.transport.Stats(stats)
	}
	return err
}

// TransportFlag - A Flag configured Transport instance.
func TransportFlag(name string) *Transport {
	t := &Transport{TLSClientConfig: &tls.Config{}}
	flag.BoolVar(
		&t.TLSClientConfig.InsecureSkipVerify,
		name+".insecure-tls",
		false,
		name+" skip tls certificate verification",
	)
	flag.BoolVar(
		&t.DisableKeepAlives,
		name+".disable-keepalive",
		false,
		name+" disable keep-alives",
	)
	flag.BoolVar(
		&t.DisableCompression,
		name+".disable-compression",
		false,
		name+" disable compression",
	)
	flag.IntVar(
		&t.MaxIdleConnsPerHost,
		name+".max-idle-conns-per-host",
		http.DefaultMaxIdleConnsPerHost,
		name+" max idle connections per host",
	)
	flag.DurationVar(
		&t.DialTimeout,
		name+".dial-timeout",
		2*time.Second,
		name+" dial timeout",
	)
	flag.DurationVar(
		&t.DialKeepAlive,
		name+".dial-keepalive",
		0,
		name+" dial keepalive connection",
	)
	flag.DurationVar(
		&t.ResponseHeaderTimeout,
		name+".response-header-timeout",
		3*time.Second,
		name+" response header timeout",
	)
	flag.DurationVar(
		&t.RequestTimeout,
		name+".request-timeout",
		30*time.Second,
		name+" request timeout",
	)
	flag.UintVar(
		&t.MaxTries,
		name+".max-tries",
		0,
		name+" max retries for known safe failures",
	)
	return t
}
