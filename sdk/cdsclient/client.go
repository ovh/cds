package cdsclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/sdk"
)

type client struct {
	httpClient    *http.Client
	httpSSEClient *http.Client
	config        Config
	name          string
}

// NewHTTPClient returns a new HTTP Client
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
	cli.config = c
	cli.httpClient = NewHTTPClient(time.Second*60, c.InsecureSkipVerifyTLS)
	cli.httpSSEClient = NewHTTPClient(0, c.InsecureSkipVerifyTLS)
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
	cli.config = conf

	if c == nil {
		cli.httpClient = NewHTTPClient(time.Second*360, false)
	} else {
		cli.httpClient = c
	}
	cli.httpSSEClient = NewHTTPClient(0, false)

	cli.name = name
	cli.init()
	return cli
}

// NewClientFromConfig returns a client from the config file
func NewClientFromConfig(r io.Reader) (Interface, error) {
	return nil, nil
}

// NewClientFromEnv returns a client from the environment variables
func NewClientFromEnv() (Interface, error) {
	return nil, nil
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
	cli.config = conf
	cli.httpClient = NewHTTPClient(time.Duration(cfg.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpSSEClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.init()
	return cli
}

// NewServiceClient returns client for a service
func NewServiceClient(cfg ServiceConfig) (Interface, []byte, error) {
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
	cli.config = conf
	cli.httpClient = NewHTTPClient(time.Duration(cfg.RequestSecondsTimeout)*time.Second, conf.InsecureSkipVerifyTLS)
	cli.httpSSEClient = NewHTTPClient(0, conf.InsecureSkipVerifyTLS)
	cli.config.Verbose = cfg.Verbose
	cli.init()

	if cfg.Hook != nil {
		if err := cfg.Hook(cli); err != nil {
			return nil, nil, sdk.WithStack(err)
		}
	}

	var res sdk.AuthConsumerSigninResponse
	_, headers, _, err := cli.RequestJSON(context.Background(), "POST", "/auth/consumer/"+string(sdk.ConsumerBuiltin)+"/signin", sdk.AuthConsumerSigninRequest{"token": cfg.Token}, &res)
	if err != nil {
		return nil, nil, err
	}
	cli.config.SessionToken = res.Token

	base64EncodedPubKey := headers.Get("X-Api-Pub-Signing-Key")
	pubKey, err := base64.StdEncoding.DecodeString(base64EncodedPubKey)

	return cli, pubKey, err
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
func (c *client) HTTPSSEClient() *http.Client {
	return c.httpSSEClient
}
