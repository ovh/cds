package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk"
)

//Github http var
var (
	RateLimitLimit     = 5000
	RateLimitRemaining = 5000
	RateLimitReset     = int(time.Now().Unix())

	httpClient = cdsclient.NewHTTPClient(time.Second*30, false)
)

func (g *githubConsumer) postForm(path string, data url.Values, headers map[string][]string) (int, []byte, error) {
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest(http.MethodPost, g.GitHubURL+path, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CDS-gh_client_id="+g.ClientID)
	for k, h := range headers {
		for i := range h {
			req.Header.Add(k, h[i])
		}
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	if res.StatusCode > 400 {
		ghErr := &ghError{}
		if err := json.Unmarshal(resBody, ghErr); err == nil {
			return res.StatusCode, resBody, ghErr
		}
	}

	return res.StatusCode, resBody, nil
}

func (c *githubClient) setETag(ctx context.Context, path string, headers http.Header) {
	etag := headers.Get("ETag")

	r, _ := regexp.Compile(".*\"(.*)\".*")
	s := r.FindStringSubmatch(etag)
	if len(s) == 2 {
		etag = s[1]
	}

	if etag != "" {
		//Put etag for this path in cache for 15 minutes
		k := cache.Key("vcs", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1))
		if err := c.Cache.SetWithTTL(k, etag, 15*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
}

func (c *githubClient) getETag(ctx context.Context, path string) string {
	var s string
	k := cache.Key("vcs", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1))
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

func (c *githubClient) post(path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
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
	if opts.asUser && c.token != "" {
		req.SetBasicAuth(c.username, c.token)
	} else {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))
	}

	log.Debug("Github API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
}

func (c *githubClient) patch(path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
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
	if opts.asUser && c.token != "" {
		req.SetBasicAuth(c.username, c.token)
	} else {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))
	}

	log.Debug("Github API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
}

func (c *githubClient) put(path string, bodyType string, body io.Reader, opts *postOptions) (*http.Response, error) {
	if opts == nil {
		opts = new(postOptions)
	}
	if !opts.skipDefaultBaseURL && !strings.HasPrefix(path, c.GitHubAPIURL) {
		path = c.GitHubAPIURL + path
	}

	req, err := http.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")
	if opts.asUser && c.token != "" {
		req.SetBasicAuth(c.username, c.token)
	} else {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))
	}

	log.Debug("Github API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
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
	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))

	if opts == nil {
		withETag(ctx, c, req, path)
	} else {
		for _, o := range opts {
			o(ctx, c, req, path)
		}
	}

	log.Debug("Github API>> Request GitHubURL %s", req.URL.String())

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
			log.Debug("Github API>> Response Follow redirect :%s", location)
			return c.get(ctx, location)
		}
	case http.StatusUnauthorized:
		return res.StatusCode, nil, nil, ErrorUnauthorized
	}

	resBody, err := ioutil.ReadAll(res.Body)
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

func (c *githubClient) delete(path string) error {
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
	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))
	log.Debug("Github API>> Request URL %s", req.URL.String())

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
