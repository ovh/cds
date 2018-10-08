package cdsclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
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
	req.Header.Set("User-Agent", c.config.userAgent)
	req.Header.Set("Connection", "close")
	req.Header.Add(RequestedWithHeader, RequestedWithValue)
	if c.name != "" {
		req.Header.Add(RequestedNameHeader, c.name)
	}
	if c.isProvider {
		req.Header.Add("X-Provider-Name", c.config.User)
		req.Header.Add("X-Provider-Token", c.config.Token)
	}

	if c.config.Hash != "" {
		basedHash := base64.StdEncoding.EncodeToString([]byte(c.config.Hash))
		req.Header.Set(AuthHeader, basedHash)
	}
	if c.config.User != "" && c.config.Token != "" {
		req.Header.Add(SessionTokenHeader, c.config.Token)
		req.SetBasicAuth(c.config.User, c.config.Token)
	}

	resp, err := NoTimeout(c.HTTPClient).Do(req)
	if err != nil {
		return err
	}
	br := bufio.NewReader(resp.Body)
	defer resp.Body.Close() // nolint

	delim := []byte{':', ' '}

	var currEvent *SSEvent
	var EOF bool

	for !EOF {
		if ctx.Err() != nil {
			break
		}

		bs, err := br.ReadBytes('\n')

		if err != nil && err != io.EOF {
			return err
		}

		if len(bs) < 2 {
			continue
		}

		spl := bytes.Split(bs, delim)

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
		if err == io.EOF {
			EOF = true
		}
	}

	return nil

}
