package github

import (
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

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"

	"github.com/facebookgo/httpcontrol"
	"github.com/ovh/cds/sdk"
)

//Github http var
var (
	RateLimitLimit     int
	RateLimitRemaining = 5000
	RateLimitReset     int

	httpClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: time.Second * 30,
			MaxTries:       5,
		},
	}
)

func (g *githubConsumer) postForm(path string, data url.Values, headers map[string][]string) (int, []byte, error) {
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest(http.MethodPost, URL+path, body)
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

func (c *githubClient) setETag(path string, headers http.Header) {
	etag := headers.Get("ETag")

	r, _ := regexp.Compile(".*\"(.*)\".*")
	s := r.FindStringSubmatch(etag)
	if len(s) == 2 {
		etag = s[1]
	}

	if etag != "" {
		//Put etag for this path in cache for 15 minutes
		c.Cache.SetWithTTL(cache.Key("vcs", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1)), etag, 15*60)
	}
}

func (c *githubClient) getETag(path string) string {
	var s string
	c.Cache.Get(cache.Key("vcs", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1)), &s)
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

type getArgFunc func(c *githubClient, req *http.Request, path string)

func withETag(c *githubClient, req *http.Request, path string) {
	etag := c.getETag(path)
	if etag != "" {
		req.Header.Add("If-None-Match", fmt.Sprintf("W/\"%s\"", etag))
	}
}
func withoutETag(c *githubClient, req *http.Request, path string) {}

func (c *githubClient) post(path string, bodyType string, body io.Reader, skipDefaultBaseURL bool) (*http.Response, error) {
	if !skipDefaultBaseURL && !strings.HasPrefix(path, APIURL) {
		path = APIURL + path
	}

	req, err := http.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)
	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))

	log.Debug("Github API>> Request URL %s", req.URL.String())

	return httpClient.Do(req)
}

func (c *githubClient) get(path string, opts ...getArgFunc) (int, []byte, http.Header, error) {
	if isRateLimitReached() {
		return 0, nil, nil, ErrorRateLimit
	}

	if !strings.HasPrefix(path, APIURL) {
		path = APIURL + path
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
		withETag(c, req, path)
	} else {
		for _, o := range opts {
			o(c, req, path)
		}
	}

	log.Debug("Github API>> Request URL %s", req.URL.String())

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
			return c.get(location)
		}
	case http.StatusUnauthorized:
		return res.StatusCode, nil, nil, ErrorUnauthorized
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, nil, err
	}

	c.setETag(path, res.Header)

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

	if !strings.HasPrefix(path, APIURL) {
		path = APIURL + path
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
		return sdk.WrapError(err, "githubClient.delete > Cannot do delete request")
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
