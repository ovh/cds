package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ovh/cds/sdk"
)

func (c *client) RequestWebsocket(ctx context.Context, goRoutines *sdk.GoRoutines, path string, msgToSend <-chan json.RawMessage, msgReceived chan<- json.RawMessage, errorReceived chan<- error) error {
	wsContext, wsContextCancel := context.WithCancel(ctx)
	defer wsContextCancel()

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
	wsContext = pprof.WithLabels(wsContext, labels)
	pprof.SetGoroutineLabels(wsContext)

	host := c.config.Host
	if strings.Contains(path, "http") {
		host = path
	} else {
		host += path
	}
	uHost, err := url.Parse(host)
	if err != nil {
		return sdk.WrapError(err, "wrong Host configuration")
	}
	urlWebsocket := url.URL{
		Scheme:   strings.Replace(uHost.Scheme, "http", "ws", -1),
		Host:     uHost.Host,
		Path:     uHost.Path,
		RawQuery: uHost.RawQuery,
	}

	headers := make(map[string][]string)
	date := sdk.FormatDateRFC5322(time.Now())
	headers["Date"] = []string{date}
	headers["X-CDS-RemoteTime"] = []string{date}
	auth := "Bearer " + c.config.SessionToken
	headers["Authorization"] = []string{auth}
	con, _, err := c.httpWebsocketClient.Dial(urlWebsocket.String(), headers)
	if err != nil {
		return sdk.WithStack(err)
	}
	defer con.Close() // nolint

	// Message to send
	goRoutines.Exec(wsContext, fmt.Sprintf("RequestWebsocket-%s-%s", c.config.User, sdk.UUID()), func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-msgToSend:
				if err := con.WriteJSON(m); err != nil {
					errorReceived <- sdk.WrapError(err, "unable to send message")
				}
			}
		}
	})

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, message, err := con.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return err
			}
			errorReceived <- sdk.WrapError(err, "unable to send message")
			continue
		}
		msgReceived <- message
	}
}
