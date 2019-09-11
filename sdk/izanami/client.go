package izanami

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Client represents the izanami client
type Client struct {
	apiURL       string
	clientID     string
	clientSecret string
	HTTPClient   *http.Client
}

// FeatureClient represents a client for feature management
type FeatureClient struct {
	client *Client
}

// SwaggerClient represents a client for swagger endpoints
type SwaggerClient struct {
	client *Client
}

// New creates a new izanami client
func New(apiURL, clientID, secret string) (*Client, error) {
	client := &Client{
		apiURL:       apiURL,
		clientID:     clientID,
		clientSecret: secret,
		HTTPClient:   &http.Client{},
	}
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
		TLSClientConfig:       &tls.Config{},
	}
	client.HTTPClient.Transport = &transport

	_, err := client.Swagger().Get()
	return client, err
}

// Feature creates a specific client for feature management
func (c *Client) Feature() *FeatureClient {
	return &FeatureClient{
		c,
	}
}

// Swagger creates a specific client for getting swagger.json
func (c *Client) Swagger() *SwaggerClient {
	return &SwaggerClient{
		c,
	}
}
