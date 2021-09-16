// For more information about how to user cdsclient package have a look at https://ovh.github.io/cds/development/sdk/golang/.

package cdsclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
)

var _ Interface = new(client)

type client struct {
	httpClient          *http.Client
	httpNoTimeoutClient *http.Client
	httpWebsocketClient *websocket.Dialer
	config              *Config
	name                string
}

func NewWebsocketDialer(insecureSkipVerifyTLS bool) *websocket.Dialer {
	return &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: insecureSkipVerifyTLS},
	}
}

// NewHTTPClient returns a new HTTP Client.
func NewHTTPClient(timeout time.Duration, insecureSkipVerifyTLS bool) *http.Client {
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 0 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecureSkipVerifyTLS},
	}

	if timeout == 0 {
		transport.IdleConnTimeout = 0
		transport.ResponseHeaderTimeout = 0
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &transport,
	}
}

// New returns a client from a config struct
func New(c Config) Interface {
	cli := new(client)
	cli.config = &c
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Second*60, c.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, c.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(c.InsecureSkipVerifyTLS)
	cli.init()
	return cli
}

// NewWorker returns client for a worker
func NewWorker(endpoint string, name string, c *http.Client) WorkerInterface {
	conf := Config{
		Host:  endpoint,
		Retry: 10,
	}
	cli := new(client)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)

	if c == nil {
		cli.httpClient = NewHTTPClient(time.Second*360, false)
	} else {
		cli.httpClient = c
	}
	cli.httpNoTimeoutClient = NewHTTPClient(0, false)
	cli.httpWebsocketClient = NewWebsocketDialer(false)

	cli.name = name
	cli.init()
	return cli
}

// NewProviderClient returns an implementation for ProviderClient interface
func NewProviderClient(cfg ProviderConfig) ProviderClient {
	conf := Config{
		Host:                              cfg.Host,
		Retry:                             2,
		BuitinConsumerAuthenticationToken: cfg.Token,
		InsecureSkipVerifyTLS:             cfg.InsecureSkipVerifyTLS,
	}

	if cfg.RequestSecondsTimeout == 0 {
		cfg.RequestSecondsTimeout = 60
	}

	cli := new(client)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Duration(cfg.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(conf.InsecureSkipVerifyTLS)
	cli.init()
	return cli
}

// NewServiceClient returns client for a service
func NewServiceClient(ctx context.Context, cfg ServiceConfig) (Interface, []byte, error) {
	conf := Config{
		Host:                              cfg.Host,
		Retry:                             2,
		BuitinConsumerAuthenticationToken: cfg.Token,
		InsecureSkipVerifyTLS:             cfg.InsecureSkipVerifyTLS,
	}

	if cfg.RequestSecondsTimeout == 0 {
		cfg.RequestSecondsTimeout = 60
	}

	cli := new(client)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Duration(cfg.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(conf.InsecureSkipVerifyTLS)
	cli.config.Verbose = cfg.Verbose
	cli.init()

	if cfg.Hook != nil {
		if err := cfg.Hook(cli); err != nil {
			return nil, nil, newError(err)
		}
	}

	var nbError int
retry:
	var res sdk.AuthConsumerSigninResponse
	_, headers, code, err := cli.RequestJSON(ctx, "POST", "/auth/consumer/"+string(sdk.ConsumerBuiltin)+"/signin", sdk.AuthConsumerSigninRequest{"token": cfg.Token}, &res)
	if err != nil {
		if code == 401 {
			nbError++
			if nbError == 60 {
				time.Sleep(time.Minute)
				goto retry
			}
		}
		return nil, nil, err
	}
	cli.config.SessionToken = res.Token

	base64EncodedPubKey := headers.Get("X-Api-Pub-Signing-Key")
	pubKey, err := base64.StdEncoding.DecodeString(base64EncodedPubKey)

	return cli, pubKey, newError(err)
}

func (c *client) init() {
	if os.Getenv("CDS_VERBOSE") == "true" {
		c.config.Verbose = true
	}
}

func (c *client) APIURL() string {
	return c.config.Host
}

func (c *client) HTTPClient() *http.Client {
	return c.httpClient
}
func (c *client) HTTPNoTimeoutClient() *http.Client {
	return c.httpNoTimeoutClient
}
func (c *client) HTTPWebsocketClient() *websocket.Dialer {
	return c.httpWebsocketClient
}

var _ error = new(Error)

type Error struct {
	sdkError       error
	transportError error
	apiError       error
}

func (e *Error) Cause() error {
	if e == nil {
		return nil
	}
	if e.apiError != nil {
		return e.apiError
	}
	if e.transportError != nil {
		return e.transportError
	}
	if e.sdkError != nil {
		return e.sdkError
	}
	return nil
}

func (e *Error) Error() string {
	if e.apiError != nil {
		return "API Error: " + e.apiError.Error()
	}
	if e.transportError != nil {
		return "Transport Error: " + e.transportError.Error()
	}
	if e.sdkError != nil {
		return e.sdkError.Error()
	}
	panic("unknown error")
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func (e *Error) StackTrace() string {
	if err, ok := e.Cause().(stackTracer); ok {
		return fmt.Sprintf("%+v", err)
	}
	return ""
}

func newAPIError(e error) error {
	if e == nil {
		return nil
	}
	return &Error{apiError: errors.WithStack(e)}
}

func newTransportError(e error) error {
	if e == nil {
		return nil
	}
	return &Error{transportError: errors.WithStack(e)}
}

func newError(e error) error {
	if e == nil {
		return nil
	}
	return &Error{sdkError: errors.WithStack(e)}
}
