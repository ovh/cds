package cdsclient

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/sdk"
)

type client struct {
	isWorker   bool
	isHatchery bool
	HTTPClient HTTPClient
	config     Config
}

// New returns a client from a config struct
func New(c Config) Interface {
	cli := new(client)
	cli.config = c
	cli.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}
	cli.init()
	return cli
}

// NewWorker returns client for a worker
func NewWorker(endpoind string) Interface {
	conf := Config{
		Host:  endpoind,
		Retry: 2,
	}
	cli := new(client)
	cli.config = conf
	cli.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}
	cli.isWorker = true
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

func (c *client) Requirements() ([]sdk.Requirement, error) {
	var req []sdk.Requirement
	if _, err := c.GetJSON("/action/requirement", &req); err != nil {
		return nil, err
	}
	return req, nil
}
