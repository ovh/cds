package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

var (
	httpClient = cdsclient.NewHTTPClient(time.Second*30, false)
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

type options struct {
	asUser bool
}

func (c *bitbucketClient) do(ctx context.Context, method, api, path string, params url.Values, values []byte, v interface{}, opts *options) error {
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
		return sdk.WithStack(err)
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
	if opts != nil && opts.asUser && c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	} else {
		if err := c.consumer.Sign(req, token); err != nil {
			return err
		}
	}

	// ensure the appropriate content-type is set for POST,
	// assuming the field is not populated
	if (req.Method == "POST" || req.Method == "PUT") && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json")
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
		return sdk.WrapError(err, "HTTP Error")
	}

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sdk.WithStack(err)
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return sdk.ErrNotFound
	case 403:
		return sdk.ErrForbidden
	case 401:
		return sdk.ErrUnauthorized
	case 400:
		log.Warning("bitbucketClient.do> %s", string(body))
		return sdk.ErrWrongRequest
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
