package httpcontrol_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/freeport"
	"github.com/facebookgo/httpcontrol"
)

var theAnswer = []byte("42")

func sleepHandler(timeout time.Duration) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(timeout)
			w.Write(theAnswer)
		})
}

func errorHandler(timeout time.Duration) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(timeout)
			w.WriteHeader(500)
			w.Write(theAnswer)
		})
}

func partialWriteNotifyingHandler(startedWriting chan<- struct{}, finishWriting <-chan struct{}) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte{theAnswer[0]})
			startedWriting <- struct{}{}
			<-finishWriting
			w.Write(theAnswer[1:])
		})
}

func assertResponse(res *http.Response, t *testing.T) {
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	err = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, theAnswer) {
		t.Fatalf(`did not find expected bytes "%s" instead found "%s"`, theAnswer, b)
	}
}

func TestOkWithDefaults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(sleepHandler(time.Millisecond))
	defer server.Close()
	transport := &httpcontrol.Transport{}
	hit := false
	transport.Stats = func(stats *httpcontrol.Stats) {
		hit = true
		if stats.Error != nil {
			t.Fatal(stats.Error)
		}
		if stats.Request == nil {
			t.Fatal("got nil request in stats")
		}
		if stats.Response == nil {
			t.Fatal("got nil response in stats")
		}
		if stats.Retry.Count != 0 {
			t.Fatal("was expecting retry count of 0")
		}
		if stats.Retry.Pending {
			t.Fatal("was expecting no retry pending")
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(res, t)
	if !hit {
		t.Fatal("no hit")
	}
}

func TestHttpError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(errorHandler(time.Millisecond))
	defer server.Close()
	transport := &httpcontrol.Transport{}
	transport.Stats = func(stats *httpcontrol.Stats) {
		if stats.Error != nil {
			t.Fatal(stats.Error)
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(res, t)
	if res.StatusCode != 500 {
		t.Fatalf("was expecting 500 got %d", res.StatusCode)
	}
}

func TestDialNoServer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(sleepHandler(time.Millisecond))
	server.Close()
	transport := &httpcontrol.Transport{}
	transport.Stats = func(stats *httpcontrol.Stats) {
		if stats.Error == nil {
			t.Fatal("was expecting error")
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if res != nil {
		t.Fatal("was expecting nil response")
	}
	if !strings.Contains(err.Error(), "dial") {
		t.Fatal("was expecting dial related error")
	}
}

func TestResponseHeaderTimeout(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(sleepHandler(5 * time.Second))
	transport := &httpcontrol.Transport{
		ResponseHeaderTimeout: 50 * time.Millisecond,
	}
	transport.Stats = func(stats *httpcontrol.Stats) {
		if stats.Error == nil {
			t.Fatal("was expecting error")
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if res != nil {
		t.Fatal("was expecting nil response")
	}

	const expected = "timeout awaiting response headers"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf(`expected "%s" got "%s"`, expected, err)
	}
}

func TestResponseTimeout(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(sleepHandler(5 * time.Second))
	transport := &httpcontrol.Transport{
		RequestTimeout: 50 * time.Millisecond,
	}
	transport.Stats = func(stats *httpcontrol.Stats) {
		if stats.Error == nil {
			t.Fatal("was expecting error")
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if res != nil {
		t.Fatal("was expecting nil response")
	}
	if !strings.Contains(err.Error(), "use of closed network connection") {
		t.Fatalf("was expecting closed network connection related error, got %s", err)
	}
}

func TestSafeRetry(t *testing.T) {
	t.Parallel()
	port, err := freeport.Get()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := httptest.NewUnstartedServer(sleepHandler(time.Millisecond))
	transport := &httpcontrol.Transport{
		MaxTries: 2,
	}
	first := false
	second := false
	transport.Stats = func(stats *httpcontrol.Stats) {
		if !first {
			first = true
			if stats.Error == nil {
				t.Fatal("was expecting error")
			}
			if !stats.Retry.Pending {
				t.Fatal("was expecting pending retry", stats.Error)
			}
			server.Listener, err = net.Listen("tcp", addr)
			if err != nil {
				t.Fatal(err)
			}
			server.Start()
			return
		}

		if !second {
			second = true
			if stats.Error != nil {
				t.Fatal(stats.Error, server.URL)
			}
			return
		}
	}
	client := &http.Client{Transport: transport}
	res, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(res, t)
	if !first {
		t.Fatal("did not see first request")
	}
	if !second {
		t.Fatal("did not see second request")
	}
}

func TestSafeRetryAfterTimeout(t *testing.T) {
	t.Parallel()
	port, err := freeport.Get()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := httptest.NewUnstartedServer(sleepHandler(5 * time.Second))
	transport := &httpcontrol.Transport{
		MaxTries:          3,
		RequestTimeout:    5 * time.Millisecond,
		RetryAfterTimeout: true,
	}
	first := false
	second := false
	third := false
	transport.Stats = func(stats *httpcontrol.Stats) {
		if !first {
			first = true
			if stats.Error == nil {
				t.Fatal("was expecting error")
			}
			if !stats.Retry.Pending {
				t.Fatal("was expecting pending retry", stats.Error)
			}
			server.Listener, err = net.Listen("tcp", addr)
			if err != nil {
				t.Fatal(err)
			}
			server.Start()
			return
		}

		if !second {
			second = true
			if stats.Error == nil {
				t.Fatal("was expecting error")
			}
			if !stats.Retry.Pending {
				t.Fatal("was expecting pending retry", stats.Error)
			}
			return
		}

		if !third {
			third = true
			if stats.Error == nil {
				t.Fatal("was expecting error")
			}
			if !stats.Retry.Pending {
				t.Fatal("was expecting pending retry", stats.Error)
			}
		}
	}
	client := &http.Client{Transport: transport}
	_, err = client.Get(fmt.Sprintf("http://%s/", addr))

	// Expect this to fail
	if err == nil {
		t.Fatal(err)
	}

	if !first {
		t.Fatal("did not see first request")
	}
	if !second {
		t.Fatal("did not see second request")
	}

	if !third {
		t.Fatal("did not see third request")
	}
}

func TestRetryEOF(t *testing.T) {
	t.Parallel()
	startedWriting := make(chan struct{})
	finishWriting := make(chan struct{})
	server := httptest.NewUnstartedServer(partialWriteNotifyingHandler(startedWriting, finishWriting))
	go func() {
		for {
			<-startedWriting
			server.CloseClientConnections()
			finishWriting <- struct{}{}
		}
	}()
	transport := &httpcontrol.Transport{
		MaxTries: 1,
	}
	transport.Stats = func(stats *httpcontrol.Stats) {
		if stats.Error != io.EOF {
			t.Fatal("was expecting", io.EOF)
		}
		if stats.Retry.Count < transport.MaxTries {
			if !stats.Retry.Pending {
				t.Fatal("was expecting pending retry", stats.Error)
			}
		} else {
			if stats.Retry.Pending {
				t.Fatal("was not expecting pending retry", stats.Error)
			}
		}
	}
	server.Start()
	client := &http.Client{Transport: transport}
	_, err := client.Get(server.URL)
	if err != nil && !strings.HasSuffix(err.Error(), io.EOF.Error()) {
		t.Fatal("was expecting", io.EOF)
	}
}

var (
	flagCount int
	flagMutex sync.Mutex
)

func flagName() string {
	flagMutex.Lock()
	defer flagMutex.Unlock()
	flagCount++
	return fmt.Sprintf("testcontrol-%d", flagCount)
}

func TestFlag(t *testing.T) {
	c := httpcontrol.TransportFlag(flagName())
	if c == nil {
		t.Fatal("did not get an instance")
	}
}

func TestStatsString(t *testing.T) {
	s := httpcontrol.Stats{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/"},
		},
		Response: &http.Response{
			Status: "200 OK",
		},
	}
	ensure.DeepEqual(t, s.String(), "GET / got response with status 200 OK")
}

func TestCloseIdleConnections(t *testing.T) {
	(&httpcontrol.Transport{}).CloseIdleConnections()
}
