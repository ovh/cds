package httpcontrol

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/facebookgo/ensure"
)

type mockNetError struct {
	temporary bool
	timeout   bool
}

func (t mockNetError) Error() string   { return "" }
func (t mockNetError) Temporary() bool { return t.temporary }
func (t mockNetError) Timeout() bool   { return t.timeout }

func TestShouldRetry(t *testing.T) {
	r := Transport{RetryAfterTimeout: true}
	cases := []error{
		mockNetError{temporary: true},
		mockNetError{timeout: true},
		&url.Error{Err: mockNetError{timeout: true}},
		errors.New("request canceled while waiting for connection"),
		&net.OpError{Err: errors.New("use of closed network connection")},
	}
	for _, s := range knownFailureSuffixes {
		cases = append(cases, errors.New(s))
	}
	for i, err := range cases {
		ensure.True(t, r.shouldRetryError(err), fmt.Sprintf("case %d", i))
	}
}

func TestShouldNotRetryRandomError(t *testing.T) {
	var r Transport
	ensure.False(t, r.shouldRetryError(errors.New("")))
}

func TestCancelRequest(t *testing.T) {
	var called bool
	timer := time.AfterFunc(time.Hour, func() { called = true })
	var r Transport
	r.CancelRequest(&http.Request{
		Body: &bodyCloser{
			timer: timer,
		},
	})
	ensure.False(t, called)
	ensure.False(t, timer.Stop())
}
