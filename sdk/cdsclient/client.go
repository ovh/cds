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

var _ Interface = new(serviceClient)
var _ HatcheryServiceClient = new(hatcheryClient)

type serviceClient struct {
	client
}

type client struct {
	httpClient          *http.Client
	httpNoTimeoutClient *http.Client
	httpWebsocketClient *websocket.Dialer
	config              *Config
	name                string
	consumerType        sdk.AuthConsumerType
	signinRequest       interface{}
}

type hatcheryClient struct {
	client
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
		ResponseHeaderTimeout: timeout,
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
	cli := new(serviceClient)
	cli.config = &c
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Second*60, c.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, c.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(c.InsecureSkipVerifyTLS)
	cli.consumerType = sdk.ConsumerBuiltin
	cli.init()
	return cli
}

func NewWorkerV2(endpoint string, name string, c *http.Client) V2WorkerInterface {
	conf := Config{
		Host:  endpoint,
		Retry: 10,
	}
	cli := new(serviceClient)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.consumerType = sdk.ConsumerBuiltin

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

// NewWorker returns client for a worker
func NewWorker(endpoint string, name string, c *http.Client) WorkerInterface {
	conf := Config{
		Host:  endpoint,
		Retry: 10,
	}
	cli := new(serviceClient)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.consumerType = sdk.ConsumerBuiltin

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
		Host:                               cfg.Host,
		Retry:                              2,
		BuiltinConsumerAuthenticationToken: cfg.Token,
		InsecureSkipVerifyTLS:              cfg.InsecureSkipVerifyTLS,
	}

	if cfg.RequestSecondsTimeout == 0 {
		cfg.RequestSecondsTimeout = 60
	}

	cli := new(serviceClient)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Duration(cfg.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(conf.InsecureSkipVerifyTLS)
	cli.consumerType = sdk.ConsumerBuiltin
	cli.init()
	return cli
}

func NewHatcheryServiceClient(ctx context.Context, clientConfig ServiceConfig, requestSign interface{}) (HatcheryServiceClient, []byte, error) {
	conf := Config{
		Host:                               clientConfig.Host,
		Retry:                              2,
		BuiltinConsumerAuthenticationToken: clientConfig.TokenV2,
		InsecureSkipVerifyTLS:              clientConfig.InsecureSkipVerifyTLS,
	}

	if clientConfig.RequestSecondsTimeout == 0 {
		clientConfig.RequestSecondsTimeout = 60
	}

	cli := new(hatcheryClient)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Duration(clientConfig.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(conf.InsecureSkipVerifyTLS)
	cli.config.Verbose = clientConfig.Verbose
	cli.consumerType = sdk.ConsumerHatchery
	cli.init()

	cli.signinRequest = requestSign

	var nbError int
retry:
	var res sdk.AuthConsumerSigninResponse
	_, headers, code, err := cli.RequestJSON(ctx, "POST", "/v2/auth/consumer/"+string(sdk.ConsumerHatchery)+"/signin",
		cli.signinRequest, &res)
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

// NewServiceClient returns client for a service
func NewServiceClient(ctx context.Context, clientConfig ServiceConfig, registerPayload interface{}) (Interface, *sdk.Service, []byte, error) {
	conf := Config{
		Host:                               clientConfig.Host,
		Retry:                              2,
		BuiltinConsumerAuthenticationToken: clientConfig.Token,
		InsecureSkipVerifyTLS:              clientConfig.InsecureSkipVerifyTLS,
	}

	if clientConfig.RequestSecondsTimeout == 0 {
		clientConfig.RequestSecondsTimeout = 60
	}

	cli := new(serviceClient)
	cli.config = &conf
	cli.config.Mutex = new(sync.Mutex)
	cli.httpClient = NewHTTPClient(time.Duration(clientConfig.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpNoTimeoutClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.httpWebsocketClient = NewWebsocketDialer(conf.InsecureSkipVerifyTLS)
	cli.config.Verbose = clientConfig.Verbose
	cli.consumerType = sdk.ConsumerBuiltin
	cli.init()

	if clientConfig.Hook != nil {
		if err := clientConfig.Hook(cli); err != nil {
			return nil, nil, nil, newError(err)
		}
	}

	cli.signinRequest = &sdk.AuthConsumerSigninRequest{
		"token":   clientConfig.Token,
		"service": registerPayload,
	}

	var nbError int
retry:
	var res sdk.AuthConsumerSigninResponse
	_, headers, code, err := cli.RequestJSON(ctx, "POST", "/auth/consumer/"+string(sdk.ConsumerBuiltin)+"/signin",
		cli.signinRequest, &res)
	if err != nil {
		if code == 401 {
			nbError++
			if nbError == 60 {
				time.Sleep(time.Minute)
				goto retry
			}
		}
		return nil, nil, nil, err
	}
	cli.config.SessionToken = res.Token

	base64EncodedPubKey := headers.Get("X-Api-Pub-Signing-Key")
	pubKey, err := base64.StdEncoding.DecodeString(base64EncodedPubKey)

	return cli, res.Service, pubKey, newError(err)
}

func (c *client) init() {
	if os.Getenv("CDS_VERBOSE") == "true" {
		c.config.Verbose = true
	}
}

func (c *client) APIURL() string {
	return c.config.Host
}

func (c *client) CDNURL() (string, error) {
	if c.config.CDNHost == "" {
		confCDN, err := c.ConfigCDN()
		if err != nil {
			return "", err
		}
		c.config.CDNHost = confCDN.HTTPURL
	}
	return c.config.CDNHost, nil
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

func (c *client) GetConsumerType() sdk.AuthConsumerType {
	if c.consumerType == "" {
		return sdk.ConsumerBuiltin
	}
	return c.consumerType
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
