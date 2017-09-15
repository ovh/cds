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
	isWorker                 bool
	isHatchery               bool
	HTTPClient               HTTPClient
	HTTPClientWithoutTimeout HTTPClient
	config                   Config
}

// New returns a client from a config struct
func New(c Config) Interface {
	cli := new(client)
	cli.config = c
	cli.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}
	cli.HTTPClientWithoutTimeout = &http.Client{}
	cli.init()
	return cli
}

// NewWorker returns client for a worker
func NewWorker(endpoint string) Interface {
	conf := Config{
		Host:  endpoint,
		Retry: 2,
	}
	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}
	cli.HTTPClientWithoutTimeout = &http.Client{}
	cli.isWorker = true
	cli.init()
	return cli
}

// NewHatchery returns client for a hatchery
func NewHatchery(endpoint string, token string, requestSecondsTimeout int, insecureSkipVerifyTLS bool) Interface {
	conf := Config{
		Host:  endpoint,
		Retry: 2,
		Token: token,
	}
	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Duration(requestSecondsTimeout) * time.Second,
			MaxTries:        5,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerifyTLS},
		},
	}

	// hatchery don't need to make a request without timeout on API
	cli.HTTPClientWithoutTimeout = nil
	cli.isHatchery = true
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

func (c *client) init() {
	if c.isWorker {
		c.config.userAgent = sdk.WorkerAgent
	} else if c.isHatchery {
		c.config.userAgent = sdk.HatcheryAgent
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
