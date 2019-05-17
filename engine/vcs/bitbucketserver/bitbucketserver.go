package bitbucket

import (
	"fmt"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// bitbucketClient is a bitbucket wrapper for CDS vcs. interface
type bitbucketClient struct {
	consumer          bitbucketConsumer
	accessToken       string
	accessTokenSecret string
	proxyURL          string
	username          string
	token             string
}

//bitbucketConsumer implements vcs.Server and it's used to instanciate a bitbucketClient
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

//New creates a new bitbucketConsumer
func New(consumerKey string, privateKey []byte, URL, apiURL, uiURL, proxyURL, username, token string, store cache.Store, disableStatus bool) sdk.VCSServer {
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
		return "", "", fmt.Errorf("fullname %s must be <project>/<slug>", fullname)
	}
	project := t[0]
	slug := t[1]
	return project, slug, nil
}
