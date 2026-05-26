package forgejo

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
)

// forgejoHTTPClient is a simple HTTP client for the Forgejo API.
type forgejoHTTPClient struct {
	baseURL    string
	username   string
	token      string
	httpClient *http.Client
}

// newForgejoHTTPClient creates a new Forgejo HTTP client with basic auth.
func newForgejoHTTPClient(baseURL, username, token string) *forgejoHTTPClient {
	return &forgejoHTTPClient{
		baseURL:    strings.TrimRight(baseURL, "/") + "/api/v1",
		username:   username,
		token:      token,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// doRequest performs an HTTP request and decodes the JSON response into result.
func (c *forgejoHTTPClient) doRequest(ctx context.Context, method, path string, body io.Reader, result interface{}) (*http.Response, error) {
	fullURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("forgejo HTTP client: failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forgejo HTTP client: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("forgejo HTTP client: failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return resp, fmt.Errorf("forgejo HTTP client: %s %s returned status %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return resp, fmt.Errorf("forgejo HTTP client: failed to decode response: %w", err)
		}
	}

	return resp, nil
}

// get performs a GET request.
func (c *forgejoHTTPClient) get(ctx context.Context, path string, result interface{}) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

// post performs a POST request with a JSON body.
func (c *forgejoHTTPClient) post(ctx context.Context, path string, body interface{}, result interface{}) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("forgejo HTTP client: failed to marshal body: %w", err)
	}
	return c.doRequest(ctx, http.MethodPost, path, bytes.NewReader(b), result)
}

// buildPaginatedPath appends pagination query parameters to a path.
func buildPaginatedPath(basePath string, opts ListOptions) string {
	params := url.Values{}
	if opts.Page != 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.PageSize != 0 {
		params.Set("limit", fmt.Sprintf("%d", opts.PageSize))
	}
	if len(params) == 0 {
		return basePath
	}
	if strings.Contains(basePath, "?") {
		return basePath + "&" + params.Encode()
	}
	return basePath + "?" + params.Encode()
}
