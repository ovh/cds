package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/facebookgo/httpcontrol"
)

//HTTPClient represent a http client
type HTTPClient struct {
	api      string
	user     string
	password string
	token    string
	http     *http.Client
}

//NewHTTPClient creates a new http client
func NewHTTPClient(api, user, password, token string) *HTTPClient {
	c := &HTTPClient{
		api:      api,
		user:     user,
		password: password,
		token:    token,
	}

	client := &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: 10 * time.Second,
			MaxTries:       5,
		},
	}

	c.http = client
	return c
}

// CDSRequest executes an authentificated HTTP request on $path given $method and $args
func (c *HTTPClient) CDSRequest(method string, path string, args []byte) ([]byte, int, error) {
	mods := []sdk.RequestModifier{
		func(req *http.Request) {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "CDS/"+sdk.VERSION)
			req.Header.Set("Connection", "close")
			req.Header.Add(sdk.RequestedWithHeader, sdk.RequestedWithValue)
		},
		func(req *http.Request) {
			if c.user != "" && c.password != "" {
				req.SetBasicAuth(c.user, c.password)
			}
		},
		func(req *http.Request) {
			if c.user != "" && c.token != "" {
				req.Header.Add(sdk.SessionTokenHeader, c.token)
				req.SetBasicAuth(c.user, c.token)
			}
		},
	}

	return c.Request(method, c.api+path, args, mods...)
}

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *HTTPClient) Request(method string, url string, args []byte, mods ...sdk.RequestModifier) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(args))

	for i := range mods {
		mods[i](req)
	}

	if err != nil {
		return nil, 0, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}

	var nbRetry = 0
	for resp.StatusCode > 500 && nbRetry < 10 {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		resp, err = c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}
		// Avoid the infinite loop
		nbRetry = nbRetry +1
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode == 500 {
		cdserr := sdk.DecodeError(body)
		if cdserr != nil {
			return nil, resp.StatusCode, cdserr
		}
	}

	// if everything is fine, return body
	return body, resp.StatusCode, nil
}

//Do is a wrapper to the hatchery http client
func (c *HTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	return c.http.Do(req)
}
