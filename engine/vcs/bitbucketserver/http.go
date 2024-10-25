package bitbucketserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	httpClient = cdsclient.NewHTTPClient(time.Second*30, false)
)

type Options struct {
	DisableCache bool
}

func (b *bitbucketClient) getFullAPIURL(api string) string {
	var url string
	switch api {
	case "keys":
		url = fmt.Sprintf("%s/rest/keys/1.0", b.consumer.URL)
	case "ssh":
		url = fmt.Sprintf("%s/rest/ssh/1.0", b.consumer.URL)
	case "core":
		url = fmt.Sprintf("%s/rest/api/1.0", b.consumer.URL)
	case "build-status":
		url = fmt.Sprintf("%s/rest/build-status/1.0", b.consumer.URL)
	case "insights":
		url = fmt.Sprintf("%s/rest/insights/1.0", b.consumer.URL)
	}

	return url
}

func (b *bitbucketClient) do(ctx context.Context, method, api, path string, params url.Values, values []byte, v interface{}, opts Options) error {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.do_http")
	defer end()

	// Sad hack to get username
	var username = false
	if path == "username" {
		username = true
		path = "/repos"
	}

	// create the URI
	apiURL := b.getFullAPIURL(api)
	uri, err := url.Parse(apiURL + path)
	if err != nil {
		return sdk.WithStack(err)
	}

	if len(params) > 0 {
		uri.RawQuery = params.Encode()
	}

	// create the request
	req := &http.Request{
		URL:        uri,
		Method:     method,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
		Header:     http.Header{},
	}

	log.Info(ctx, "%s %s", req.Method, req.URL.String())

	if len(values) > 0 {
		buf := bytes.NewBuffer(values)
		req.Body = io.NopCloser(buf)
		req.ContentLength = int64(buf.Len())
	}

	var cacheKey string
	if b.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))
		cacheKey = cache.Key("vcs", "bitbucket", "request", req.URL.String(), b.username)
	}

	// ensure the appropriate content-type is set for POST,
	// assuming the field is not populated
	if (req.Method == "POST" || req.Method == "PUT") && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	if v != nil && method == "GET" && !opts.DisableCache {
		find, err := b.consumer.cache.Get(cacheKey, v)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
		}
		if find {
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return sdk.WithStack(err)
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return sdk.WithStack(sdk.ErrNotFound)
	case 403:
		return sdk.WithStack(sdk.ErrForbidden)
	case 401:
		return sdk.WithStack(sdk.ErrUnauthorized)
	case 400:
		log.Warn(ctx, "bitbucketClient.do> %s", string(body))
		return sdk.WithStack(sdk.ErrWrongRequest)
	}

	if method != "GET" {
		if err := b.consumer.cache.Delete(cacheKey); err != nil {
			log.Error(ctx, "bitbucketClient.do> unable to delete cache key %v: %v", cacheKey, err)
		}
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
			if err := sdk.JSONUnmarshal(body, v); err != nil {
				return err
			}
		}
		if method == "GET" {
			if err := b.consumer.cache.Set(cacheKey, v); err != nil {
				log.Error(ctx, "unable to cache set %v: %v", cacheKey, err)
			}
		}
	}

	return nil
}

func (b *bitbucketClient) stream(ctx context.Context, method, api, path string, params url.Values, values []byte) (io.Reader, http.Header, error) {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.stream")
	defer end()

	// create the URI
	apiURL := b.getFullAPIURL(api)
	uri, err := url.Parse(apiURL + path)
	if err != nil {
		return nil, nil, sdk.WithStack(err)
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
		Header:     http.Header{},
	}

	log.Info(ctx, "%s %s", req.Method, req.URL.String())

	if values != nil && len(values) > 0 {
		buf := bytes.NewBuffer(values)
		req.Body = io.NopCloser(buf)
		req.ContentLength = int64(buf.Len())
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))

	// ensure the appropriate content-type is set for POST,
	// assuming the field is not populated
	if (req.Method == "POST" || req.Method == "PUT") && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// make the request using the default http client
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "HTTP Error")
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return nil, nil, sdk.WithStack(sdk.ErrNotFound)
	case 403:
		return nil, nil, sdk.WithStack(sdk.ErrForbidden)
	case 401:
		return nil, nil, sdk.WithStack(sdk.ErrUnauthorized)
	case 400:
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Warn(ctx, "bitbucketClient.do> %s", string(body))
		return nil, nil, sdk.WithStack(sdk.ErrWrongRequest)
	}
	return resp.Body, resp.Header, nil
}
