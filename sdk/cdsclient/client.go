package cdsclient

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/ovh/cds/sdk"
)

type client struct {
	isWorker   bool
	isHatchery bool
	isService  bool
	isProvider bool
	HTTPClient HTTPClient
	config     Config
	name       string
}

// New returns a client from a config struct
func New(c Config) Interface {
	cli := new(client)
	cli.config = c
	cli.HTTPClient = &http.Client{
		Timeout: time.Second * 60,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerifyTLS},
		},
	}
	cli.init()
	return cli
}

// NewService returns client for a service
func NewService(endpoint string, timeout time.Duration) Interface {
	conf := Config{
		Host:  endpoint,
		Retry: 2,
	}
	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Timeout: timeout,
	}
	cli.isService = true
	cli.init()
	return cli
}

// NewWorker returns client for a worker
func NewWorker(endpoint string, name string, c HTTPClient) Interface {
	conf := Config{
		Host:  endpoint,
		Retry: 10,
	}
	cli := new(client)
	cli.config = conf

	if c == nil {
		cli.HTTPClient = &http.Client{Timeout: time.Second * 360}
	} else {
		cli.HTTPClient = c
	}

	cli.isWorker = true
	cli.name = name
	cli.init()
	return cli
}

// NewHatchery returns client for a hatchery
func NewHatchery(endpoint string, token string, requestSecondsTimeout int, insecureSkipVerifyTLS bool, name string) Interface {
	conf := Config{
		Host:  endpoint,
		Retry: 2,
		Token: token,
		InsecureSkipVerifyTLS: insecureSkipVerifyTLS,
	}
	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Duration(requestSecondsTimeout) * time.Second,
			MaxTries:        5,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.InsecureSkipVerifyTLS},
		},
	}

	// hatchery don't need to make a request without timeout on API
	cli.isHatchery = true
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
		Host:  cfg.Host,
		Retry: 2,
		Token: cfg.Token,
		User:  cfg.Name,
	}

	if cfg.RequestSecondsTimeout == 0 {
		cfg.RequestSecondsTimeout = 60
	}

	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Duration(cfg.RequestSecondsTimeout) * time.Second,
			MaxTries:        5,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerifyTLS},
		},
	}
	cli.isProvider = true
	cli.name = cfg.Name
	cli.init()
	return cli
}

func (c *client) init() {
	if c.isWorker {
		c.config.userAgent = sdk.WorkerAgent
	} else if c.isHatchery {
		c.config.userAgent = sdk.HatcheryAgent
	} else if c.isService {
		c.config.userAgent = sdk.ServiceAgent
	} else {
		c.config.userAgent = sdk.SDKAgent
	}

	if os.Getenv("CDS_VERBOSE") == "true" {
		c.config.Verbose = true
	}
}

func (c *client) APIURL() string {
	return c.config.Host
}
