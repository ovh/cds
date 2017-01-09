package stash

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-stash/go-stash/oauth1"
)

var (
	// Returned if the specified resource does not exist.
	ErrNotFound = errors.New("Not Found")

	// Returned if the caller attempts to make a call or modify a resource
	// for which the caller is not authorized.
	//
	// The request was a valid request, the caller's authentication credentials
	// succeeded but those credentials do not grant the caller permission to
	// access the resource.
	ErrForbidden = errors.New("Forbidden")

	// Returned if the call requires authentication and either the credentials
	// provided failed or no credentials were provided.
	ErrNotAuthorized = errors.New("Unauthorized")

	// Returned if the caller submits a badly formed request. For example,
	// the caller can receive this return if you forget a required parameter.
	ErrBadRequest = errors.New("Bad Request")
)

// DefaultClient uses DefaultTransport, and is used internall to execute
// all http.Requests. This may be overriden for unit testing purposes.
//
// IMPORTANT: this is not thread safe and should not be touched with
// the exception overriding for mock unit testing.
var DefaultClient = http.DefaultClient

func (c *Client) do(method, api, path string, params url.Values, values []byte, v interface{}) error {
	// Sad hack to get username
	var username = false
	if path == "username" {
		username = true
		path = "/repos"
	}

	// if this is the guest client then we don't need
	// to sign the request ... we will execute just
	// a simple http request.
	if c == Guest {
		return c.guest(method, path, params, values, v)
	}

	// create the client
	var client = oauth1.Consumer{
		ConsumerKey:           c.ConsumerKey,
		ConsumerSecret:        c.ConsumerSecret,
		ConsumerPrivateKeyPem: c.ConsumerPrivateKeyPem,
	}

	// create the URI
	apiUrl := c.GetFullApiUrl(api)
	uri, err := url.Parse(apiUrl + path)
	if err != nil {
		return err
	}

	if params != nil && len(params) > 0 {
		uri.RawQuery = params.Encode()
	}

	// create the access token
	token := oauth1.NewAccessToken(c.AccessToken, c.TokenSecret, nil)

	// create the request
	req := &http.Request{
		URL:        uri,
		Method:     method,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
		Header:     http.Header{},
	}

	if values != nil && len(values) > 0 {
		buf := bytes.NewBuffer(values)
		req.Body = ioutil.NopCloser(buf)
		req.ContentLength = int64(buf.Len())
	}

	// sign the request
	if err := client.Sign(req, token); err != nil {
		return err
	}

	// make the request using the default http client
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	}

	// Unmarshall the JSON response
	if v != nil {
		// If looking for username then pull that from header
		if username {
			body, err = json.Marshal(map[string]string{"name": resp.Header["X-Ausername"][0]})
			if err != nil {
				return nil
			}
		}
		return json.Unmarshal(body, v)
	}

	return nil
}

func (c *Client) guest(method string, path string, params url.Values, values []byte, v interface{}) error {

	// create the URI
	uri, err := url.Parse(c.ApiUrl + path)
	if err != nil {
		return err
	}

	if params != nil && len(params) > 0 {
		uri.RawQuery = params.Encode()
	}

	// create the request
	req := &http.Request{
		URL:        uri,
		Method:     method,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
	}

	// add the Form values to the body
	if values != nil && len(values) > 0 {
		buf := bytes.NewBuffer(values)
		req.Body = ioutil.NopCloser(buf)
	}

	// make the request using the default http client
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	}

	// Unmarshall the JSON response
	if v != nil {
		return json.Unmarshal(body, v)
	}

	return nil
}
