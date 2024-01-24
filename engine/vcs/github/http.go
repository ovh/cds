package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Github http var
var (
	RateLimitLimit     = 5000
	RateLimitRemaining = 5000
	RateLimitReset     = int(time.Now().Unix())

	httpClient = cdsclient.NewHTTPClient(time.Second*30, false)
)

func (c *githubClient) setETag(ctx context.Context, path string, headers http.Header) {
	etag := headers.Get("ETag")

	r, _ := regexp.Compile(".*\"(.*)\".*")
	s := r.FindStringSubmatch(etag)
	if len(s) == 2 {
		etag = s[1]
	}

	if etag != "" {
		//Put etag for this path in cache for 15 minutes
		k := cache.Key("vcs", "github", "etag", sdk.Hash512(c.OAuthToken+c.username), strings.Replace(path, "https://", "", -1))
		if err := c.Cache.SetWithTTL(k, etag, 15*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
}

func (c *githubClient) getETag(ctx context.Context, path string) string {
	var s string
	k := cache.Key("vcs", "github", "etag", sdk.Hash512(c.OAuthToken+c.username), strings.Replace(path, "https://", "", -1))
	if _, err := c.Cache.Get(k, &s); err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	return s
}

func getNextPage(headers http.Header) string {
	linkHeader := headers.Get("Link")
	if linkHeader != "" {
		links := strings.Split(linkHeader, ",")
		for _, link := range links {
			if strings.Contains(link, "rel=\"next\"") {
				r, _ := regexp.Compile("<(.*)>.*")
				s := r.FindStringSubmatch(link)
				if len(s) == 2 {
					return s[1]
				}
				break
			}
		}
	}
	return ""
}

type getArgFunc func(ctx context.Context, c *githubClient, req *http.Request, path string)

func withETag(ctx context.Context, c *githubClient, req *http.Request, path string) {
	etag := c.getETag(ctx, path)
	if etag != "" {
		req.Header.Add("If-None-Match", fmt.Sprintf("W/\"%s\"", etag))
	}
}
func withoutETag(ctx context.Context, c *githubClient, req *http.Request, path string) {}

type postOptions struct {
	skipDefaultBaseURL bool
	asUser             bool
}

func (c *githubClient) post(ctx context.Context, path string, bodyType string, body io.Reader, headers map[string]string, opts *postOptions) (*http.Response, error) {
	if opts == nil {
		opts = new(postOptions)
	}
	if !opts.skipDefaultBaseURL && !strings.HasPrefix(path, c.GitHubAPIURL) {
		path = c.GitHubAPIURL + path
	}

	req, err := http.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")

	if err := c.setAuth(ctx, req, opts); err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// If body is not *bytes.Buffer, *bytes.Reader or *strings.Reader Content-Length is not set. (
	// Here we force Content-Length.
	// cf net/http/request.go  NewRequestWithContext
	if req.Header.Get("Content-Length") != "" {
		s, err := strconv.Atoi(req.Header.Get("Content-Length"))
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		req.ContentLength = int64(s)
	}

	return httpClient.Do(req)
}

func (c *githubClient) patch(ctx context.Context, path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
	if opts == nil {
		opts = new(postOptions)
	}
	if !opts.skipDefaultBaseURL && !strings.HasPrefix(path, c.GitHubAPIURL) {
		path = c.GitHubAPIURL + path
	}

	req, err := http.NewRequest(http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	if bodyType != "" {
		req.Header.Set("Content-Type", bodyType)
	}
	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")

	if err := c.setAuth(ctx, req, opts); err != nil {
		return nil, err
	}

	return httpClient.Do(req)
}

func (c *githubClient) setAuth(ctx context.Context, req *http.Request, opts *postOptions) error {
	if opts != nil && opts.asUser && c.username != "" && c.token != "" {
		req.SetBasicAuth(c.username, c.token)
		log.Debug(ctx, "Github API>> Request with basicAuth url:%s username:%v len:%d", req.URL.String(), c.username, len(c.token))
	} else if c.OAuthToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))
		log.Debug(ctx, "Github API>> Request with OAuthToken url:%s", req.URL.String())
	} else if c.token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.token))
		log.Debug(ctx, "Github API>> Request with token url:%s len:%d", req.URL.String(), len(c.token))
	} else {
		return sdk.NewError(sdk.ErrWrongRequest, errors.New("invalid configuration - github authentication"))
	}
	return nil
}

func (c *githubClient) get(ctx context.Context, path string, opts ...getArgFunc) (int, []byte, http.Header, error) {
	if isRateLimitReached() {
		return 0, nil, nil, ErrorRateLimit
	}

	if !strings.HasPrefix(path, c.GitHubAPIURL) {
		path = c.GitHubAPIURL + path
	}

	callURL, err := url.ParseRequestURI(path)
	if err != nil {
		return 0, nil, nil, err
	}

	req, err := http.NewRequest(http.MethodGet, callURL.String(), nil)
	if err != nil {
		return 0, nil, nil, err
	}

	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")

	if err := c.setAuth(ctx, req, nil); err != nil {
		return 0, nil, nil, err
	}

	if opts == nil {
		withETag(ctx, c, req, path)
	} else {
		for _, o := range opts {
			o(ctx, c, req, path)
		}
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
			log.Debug(ctx, "Github API>> Response Follow redirect :%s", location)
			return c.get(ctx, location)
		}
	case http.StatusUnauthorized:
		return res.StatusCode, nil, nil, ErrorUnauthorized
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, nil, err
	}

	c.setETag(ctx, path, res.Header)

	rateLimitLimit := res.Header.Get("X-RateLimit-Limit")
	rateLimitRemaining := res.Header.Get("X-RateLimit-Remaining")
	rateLimitReset := res.Header.Get("X-RateLimit-Reset")

	if rateLimitLimit != "" && rateLimitRemaining != "" && rateLimitReset != "" {
		RateLimitLimit, _ = strconv.Atoi(rateLimitLimit)
		RateLimitRemaining, _ = strconv.Atoi(rateLimitRemaining)
		RateLimitReset, _ = strconv.Atoi(rateLimitReset)
	}

	return res.StatusCode, resBody, res.Header, nil
}

func (c *githubClient) delete(ctx context.Context, path string) error {
	if isRateLimitReached() {
		return ErrorRateLimit
	}

	if !strings.HasPrefix(path, c.GitHubAPIURL) {
		path = c.GitHubAPIURL + path
	}

	req, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)

	if err := c.setAuth(ctx, req, nil); err != nil {
		return err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return sdk.WrapError(err, "Cannot do delete request")
	}

	rateLimitLimit := res.Header.Get("X-RateLimit-Limit")
	rateLimitRemaining := res.Header.Get("X-RateLimit-Remaining")
	rateLimitReset := res.Header.Get("X-RateLimit-Reset")

	if rateLimitLimit != "" && rateLimitRemaining != "" && rateLimitReset != "" {
		RateLimitRemaining, _ = strconv.Atoi(rateLimitRemaining)
		RateLimitReset, _ = strconv.Atoi(rateLimitReset)
	}

	if res.StatusCode != 204 {
		return fmt.Errorf("github>delete wrong status code %d on url %s", res.StatusCode, path)
	}
	return nil
}
