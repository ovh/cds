package cdsclient

import (
	"net/http"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
)

// NewLocalServiceClient creates a cdsclient.Interface that communicates with the
// API handler in-process, without network calls or authentication tokens.
// The serviceName and serviceType identify the calling service so the auth
// middleware can load the proper service identity from the database.
func NewLocalServiceClient(handler http.Handler, serviceName, serviceType string) Interface {
	transport := NewLocalRoundTripper(handler, serviceName, serviceType)

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   0, // no timeout for in-process calls
	}

	noTimeoutClient := &http.Client{
		Transport: transport,
		Timeout:   0,
	}

	cli := new(serviceClient)
	cli.config = &Config{
		Host:  "http://local",
		Mutex: &sync.Mutex{},
		// SessionToken deliberately empty — local transport uses context injection
		// BuiltinConsumerAuthenticationToken deliberately empty — no token refresh needed
	}
	cli.httpClient = httpClient
	cli.httpNoTimeoutClient = noTimeoutClient
	cli.consumerType = sdk.ConsumerBuiltin
	cli.isLocal = true

	return cli
}

// NewLocalHatcheryServiceClient creates a HatcheryServiceClient that communicates
// with the API handler in-process without network calls or tokens.
func NewLocalHatcheryServiceClient(handler http.Handler, serviceName string) (HatcheryServiceClient, error) {
	transport := NewLocalRoundTripper(handler, serviceName, sdk.TypeHatchery)

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Minute,
	}

	noTimeoutClient := &http.Client{
		Transport: transport,
		Timeout:   0,
	}

	cli := new(hatcheryClient)
	cli.config = &Config{
		Host:  "http://local",
		Mutex: &sync.Mutex{},
	}
	cli.httpClient = httpClient
	cli.httpNoTimeoutClient = noTimeoutClient
	cli.consumerType = sdk.ConsumerHatchery
	cli.isLocal = true

	return cli, nil
}
