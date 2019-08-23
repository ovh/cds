package cdsclient

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"runtime/pprof"
	"strings"

	"github.com/ovh/cds/sdk"
)

const (
	sseEvent = "event"
	sseData  = "data"
)

//SSEvent is a go representation of an http server-sent event
type SSEvent struct {
	URI  string
	Type string
	Data io.Reader
}

// RequestSSEGet takes the uri of an SSE stream and channel, and will send an Event
// down the channel when received, until the stream is closed. It will then
// close the stream. This is blocking, and so you will likely want to call this
// in a new goroutine (via `go c.RequestSSEGet(..)`)
func (c *client) RequestSSEGet(ctx context.Context, path string, evCh chan<- SSEvent, mods ...RequestModifier) error {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	if !c.config.HasValidSessionToken() && c.config.BuitinConsumerAuthenticationToken != "" {
		resp, err := c.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{"token": c.config.BuitinConsumerAuthenticationToken})
		if err != nil {
			return err
		}
		c.config.SessionToken = resp.Token
	}

	labels := pprof.Labels("path", path, "method", "GET")
	ctx = pprof.WithLabels(ctx, labels)
	pprof.SetGoroutineLabels(ctx)

	uri := c.config.Host + path
	if strings.HasPrefix(path, "http") {
		uri = path
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "close")

	auth := "Bearer " + c.config.SessionToken
	req.Header.Add("Authorization", auth)

	resp, err := c.httpSSEClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == 401 {
		c.config.SessionToken = ""
	}

	br := bufio.NewReader(resp.Body)
	defer resp.Body.Close() // nolint

	delim := []byte{':', ' '}

	var currEvent *SSEvent
	var EOF bool

	for !EOF {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		bs, err := br.ReadBytes('\n')

		if err != nil {
			switch err {
			case io.EOF, io.ErrUnexpectedEOF, io.ErrClosedPipe:
				EOF = true
			default:
				return err
			}
		}

		if len(bs) < 2 {
			continue
		}

		spl := bytes.SplitN(bs, delim, 2)

		if len(spl) < 2 {
			continue
		}

		currEvent = &SSEvent{URI: uri}
		switch string(spl[0]) {
		case sseEvent:
			currEvent.Type = string(bytes.TrimSpace(spl[1]))
		case sseData:
			currEvent.Data = bytes.NewBuffer(bytes.TrimSpace(spl[1]))
			evCh <- *currEvent
		}

	}

	return nil

}
