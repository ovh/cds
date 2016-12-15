package repogithub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"

	"github.com/facebookgo/httpcontrol"
)

//Github http var
var (
	RateLimitRemaining = 5000
	RateLimitReset     int

	httpClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: time.Second * 30,
			MaxTries:       5,
		},
	}
)

func (g *GithubConsumer) postForm(path string, data url.Values, headers map[string][]string) (int, []byte, error) {
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
		ghErr := &Error{}
		if err := json.Unmarshal(resBody, ghErr); err == nil {
			return res.StatusCode, resBody, ghErr
		}
	}

	return res.StatusCode, resBody, nil
}

func (c *GithubClient) setETag(path string, headers http.Header) {

	etag := headers.Get("ETag")

	r, _ := regexp.Compile(".*\"(.*)\".*")
	s := r.FindStringSubmatch(etag)
	if len(s) == 2 {
		etag = s[1]
	}

	if etag != "" {
		//log.Debug("Github API>> Store ETag: %s : %s", path, etag)
		//Put etag for this path in cache for 59 minutes
		cache.SetWithTTL(cache.Key("reposmanager", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1)), etag, 59*60)
	}
}

func (c *GithubClient) getETag(path string) string {
	var s string
	cache.Get(cache.Key("reposmanager", "github", "etag", c.OAuthToken, strings.Replace(path, "https://", "", -1)), &s)
	//log.Debug("Github API>> Retrieve ETag: %s : %s", path, s)
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

type getArgFunc func(c *GithubClient, req *http.Request, path string)

func WithETag(c *GithubClient, req *http.Request, path string) {
	etag := c.getETag(path)
	if etag != "" {
		req.Header.Add("If-None-Match", fmt.Sprintf("W/\"%s\"", etag))
	}
}
func WithoutETag(c *GithubClient, req *http.Request, path string) {}

func (c *GithubClient) get(path string, opts ...getArgFunc) (int, []byte, http.Header, error) {
	if RateLimitRemaining < 100 {
		return 0, nil, nil, ErrorRateLimit
	}

	if !strings.HasPrefix(path, APIURL) {
		path = APIURL + path
	}

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return 0, nil, nil, err
	}

	req.Header.Set("User-Agent", "CDS-gh_client_id="+c.ClientID)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.OAuthToken))

	if opts == nil {
		WithETag(c, req, path)
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

	rateLimitLimit := res.Header.Get("X-RateLimit-Limit")
	rateLimitRemaining := res.Header.Get("X-RateLimit-Remaining")
	rateLimitReset := res.Header.Get("X-RateLimit-Reset")

	if rateLimitLimit != "" && rateLimitRemaining != "" && rateLimitReset != "" {
		RateLimitRemaining, _ = strconv.Atoi(rateLimitRemaining)
		RateLimitReset, _ = strconv.Atoi(rateLimitReset)
	}

	switch res.StatusCode {
	case http.StatusNotModified:
		return res.StatusCode, nil, res.Header, nil
	case http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusFound:
		location := res.Header.Get("Location")
		if location != "" {
			log.Info("Github API>> Reponse Follow redirect :%s", location)
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

	return res.StatusCode, resBody, res.Header, nil

}
