package bitbucketserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// bitbucketClient is a bitbucket wrapper for CDS vcs. interface
type bitbucketClient struct {
	username string
	token    string

	proxyURL          string
	consumer          bitbucketConsumer // DEPRECATED VCS
	accessToken       string            // DEPRECATED VCS
	accessTokenSecret string            // DEPRECATED VCS
}

// DEPRECATED VCS
//bitbucketConsumer implements vcs.Server and it's used to instantiate a bitbucketClient
type bitbucketConsumer struct {
	ConsumerKey      string `json:"consumer_key"`
	PrivateKey       []byte `json:"-"`
	URL              string `json:"url"`
	cache            cache.Store
	requestTokenURL  string
	authorizationURL string
	accessTokenURL   string
	callbackURL      string
	apiURL           string
	uiURL            string
	proxyURL         string
	disableStatus    bool
	username         string
	token            string
}

//New creates a new bitbucket Consumer
func New(URL, apiURL, uiURL, proxyURL string, store cache.Store, username, token string) sdk.VCSServer {
	return &bitbucketConsumer{
		URL:      URL,
		apiURL:   apiURL,
		uiURL:    uiURL,
		proxyURL: proxyURL,
		cache:    store,
		username: username,
		token:    token,
	}
}

// DEPRECATED VCS
func NewDeprecated(consumerKey string, privateKey []byte, URL, apiURL, uiURL, proxyURL, username, token string, store cache.Store, disableStatus bool) sdk.VCSServer {
	return &bitbucketConsumer{
		ConsumerKey:      consumerKey,
		PrivateKey:       privateKey,
		URL:              URL,
		apiURL:           apiURL,
		uiURL:            uiURL,
		proxyURL:         proxyURL,
		cache:            store,
		requestTokenURL:  URL + "/plugins/servlet/oauth/request-token",
		authorizationURL: URL + "/plugins/servlet/oauth/authorize",
		accessTokenURL:   URL + "/plugins/servlet/oauth/access-token",
		callbackURL:      oauth1OOB,
		disableStatus:    disableStatus,
		username:         username,
		token:            token,
	}
}

func getRepo(fullname string) (string, string, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return "", "", sdk.WithStack(fmt.Errorf("fullname %s must be <project>/<slug>", fullname))
	}
	project := t[0]
	slug := t[1]
	return project, slug, nil
}

func (c *bitbucketClient) GetAccessToken(_ context.Context) string {
	return c.accessToken
}
