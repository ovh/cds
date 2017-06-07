package cdsclient

import (
	"context"
	"net/http"


	"github.com/ovh/cds/sdk"
)

// HTTPClient is a interface for HTTPClient mock
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Interface is the main interface for cdsclient package
type Interface interface {
	QueuePolling(c context.Context, jobs chan<- sdk.WorkflowNodeJobRun) error
}

type client struct {
	isWorker   bool
	isHatchery bool
	httpclient HTTPClient
	config     Config
}
