package bitbucket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/facebookgo/httpcontrol"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

var (
	httpClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: time.Second * 30,
			MaxTries:       5,
		},
	}
)

func requestString(method string, uri string, params map[string]string) string {
	// loop through params, add keys to map
	var keys []string
	for key := range params {
		keys = append(keys, key)
	}

	// sort the array of header keys
	sort.StringSlice(keys).Sort()

	// create the signed string
	result := method + "&" + escape(uri)

	// loop through sorted params and append to the string
	for pos, key := range keys {
		if pos == 0 {
			result += "&"
		} else {
			result += escape("&")
		}

		result += escape(fmt.Sprintf("%s=%s", key, escape(params[key])))
	}

	return result
}

var (
	// ErrNotFound is returned if the specified resource does not exist.
	ErrNotFound = errors.New("Not Found")

	// ErrForbidden is returned if the caller attempts to make a call or modify a resource
	// for which the caller is not authorized.
	//
	// The request was a valid request, the caller's authentication credentials
	// succeeded but those credentials do not grant the caller permission to
	// access the resource.
	ErrForbidden = errors.New("Forbidden")

	// ErrNotAuthorized is returned if the call requires authentication and either the credentials
	// provided failed or no credentials were provided.
	ErrNotAuthorized = errors.New("Unauthorized")

	// ErrBadRequest is returned if the caller submits a badly formed request. For example,
	// the caller can receive this return if you forget a required parameter.
	ErrBadRequest = errors.New("Bad Request")
)

func (c *bitbucketClient) getFullAPIURL(api string) string {
	var url string
	switch api {
	case "keys":
		url = fmt.Sprintf("%s/rest/keys/1.0", c.consumer.URL)
	case "ssh":
		url = fmt.Sprintf("%s/rest/ssh/1.0", c.consumer.URL)
	case "core":
		url = fmt.Sprintf("%s/rest/api/1.0", c.consumer.URL)
	case "build-status":
		url = fmt.Sprintf("%s/rest/build-status/1.0", c.consumer.URL)
	}

	return url
}

func (c *bitbucketClient) do(method, api, path string, params url.Values, values []byte, v interface{}) error {
	// Sad hack to get username
	var username = false
	if path == "username" {
		username = true
		path = "/repos"
	}

	// create the URI
	apiURL := c.getFullAPIURL(api)
	uri, err := url.Parse(apiURL + path)
	if err != nil {
		return err
	}

	if params != nil && len(params) > 0 {
		uri.RawQuery = params.Encode()
	}

	// create the access token
	token := NewAccessToken(c.accessToken, c.accessTokenSecret, nil)

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
	if err := c.consumer.Sign(req, token); err != nil {
		return err
	}

	cacheKey := cache.Key("vcs", "bitbucket", "request", req.URL.String(), token.Token())
	if v != nil && method == "GET" {
		if c.consumer.cache.Get(cacheKey, v) {
			return nil
		}
	}

	// make the request using the default http client
	resp, err := httpClient.Do(req)
	if err != nil {
		return sdk.WrapError(err, "VCS> Bitbucket> HTTP Error")
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

		// bitbucket can return 204 with no-content
		if resp.StatusCode != 204 || strings.TrimSpace(string(body)) != "" {
			if err := json.Unmarshal(body, v); err != nil {
				return err
			}
		}

		c.consumer.cache.Set(cacheKey, v)
	}

	return nil
}
