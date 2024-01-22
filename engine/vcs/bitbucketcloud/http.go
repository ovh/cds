package bitbucketcloud

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Github http var
var httpClient = cdsclient.NewHTTPClient(time.Second*30, false)

type postOptions struct {
	skipDefaultBaseURL bool
	asUser             bool
}

func (client *bitbucketcloudClient) setAuth(ctx context.Context, req *http.Request) error {
	if client.appPassword != "" && client.username != "" {
		req.SetBasicAuth(client.username, client.appPassword)
		log.Debug(ctx, "Bitbucketcloud API>> Request with basicAuth url:%s username:%v len:%d", req.URL.String(), client.username, len(client.appPassword))
		return nil
	}
	return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid configuration - bitbucketcloud authentication username:%v lenPassword:%d", client.username, len(client.appPassword)))
}

func (client *bitbucketcloudClient) post(ctx context.Context, path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, rootURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)
	req.Header.Add("Accept", "application/json")
	if err := client.setAuth(ctx, req); err != nil {
		return nil, err
	}

	log.Debug(ctx, "Bitbucket Cloud API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
}

func (client *bitbucketcloudClient) put(ctx context.Context, path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, rootURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)
	req.Header.Add("Accept", "application/json")
	if err := client.setAuth(ctx, req); err != nil {
		return nil, err
	}

	log.Debug(context.TODO(), "Bitbucket Cloud API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
}

func (client *bitbucketcloudClient) get(ctx context.Context, path string) (int, []byte, http.Header, error) {
	callURL, err := url.ParseRequestURI(rootURL + path)
	if err != nil {
		return 0, nil, nil, sdk.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodGet, callURL.String(), nil)
	if err != nil {
		return 0, nil, nil, sdk.WithStack(err)
	}

	req.Header.Add("Accept", "application/json")
	if err := client.setAuth(ctx, req); err != nil {
		return 0, nil, nil, sdk.WithStack(err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotModified:
		return res.StatusCode, nil, res.Header, nil
	case http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusFound:
		location := res.Header.Get("Location")
		if location != "" {
			log.Debug(ctx, "Bitbucket Cloud API>> Response Follow redirect :%s", location)
			return client.get(ctx, location)
		}
	case http.StatusUnauthorized:
		return res.StatusCode, nil, nil, sdk.WithStack(ErrorUnauthorized)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, nil, sdk.WithStack(err)
	}

	return res.StatusCode, resBody, res.Header, nil
}

func (client *bitbucketcloudClient) delete(ctx context.Context, path string) error {
	req, err := http.NewRequest(http.MethodDelete, rootURL+path, nil)
	if err != nil {
		return sdk.WithStack(err)
	}

	if err := client.setAuth(ctx, req); err != nil {
		return err
	}
	log.Debug(ctx, "Bitbucket Cloud API>> Request URL %s", req.URL.String())

	res, err := httpClient.Do(req)
	if err != nil {
		return sdk.WrapError(err, "Cannot do delete request")
	}

	if res.StatusCode != 204 {
		return fmt.Errorf("bitbucketcloud>delete wrong status code %d on url %s", res.StatusCode, path)
	}
	return nil
}

func (client *bitbucketcloudClient) do(ctx context.Context, method, api, path string, params url.Values, values []byte, v interface{}) error {
	// create the URI
	uri, err := url.Parse(rootURL + path)
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

	if len(values) > 0 {
		buf := bytes.NewBuffer(values)
		req.Body = io.NopCloser(buf)
		req.ContentLength = int64(buf.Len())
	}
	if err := client.setAuth(ctx, req); err != nil {
		return err
	}

	// ensure the appropriate content-type is set for POST,
	// assuming the field is not populated
	if (req.Method == "POST" || req.Method == "PUT") && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json")
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

	return sdk.WithStack(sdk.JSONUnmarshal(body, v))
}
