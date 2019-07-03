package izanami

import (
	"net/http"
	"time"

	"github.com/facebookgo/httpcontrol"
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
		HTTPClient: &http.Client{
			Transport: &httpcontrol.Transport{
				RequestTimeout: time.Second * 30,
				MaxTries:       5,
			},
		},
	}
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
