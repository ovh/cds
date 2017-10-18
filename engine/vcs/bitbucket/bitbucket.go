package bitbucket

import (
	"fmt"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// bitbucketClient is a github.com wrapper for CDS vcs. interface
type bitbucketClient struct {
	consumer          bitbucketConsumer
	apiURL            string
	uiURL             string
	accessToken       string
	accessTokenSecret string
}

//bitbucketConsumer implements vcs.Server and it's used to instanciate a githubClient
type bitbucketConsumer struct {
	ConsumerKey      string `json:"consumer_key"`
	PrivateKey       []byte `json:"-"`
	URL              string `json:"url"`
	cache            cache.Store
	requestTokenURL  string
	authorizationURL string
	accessTokenURL   string
	callbackURL      string
}

//New creates a new bitbucketConsumer
func New(consumerKey string, privateKey []byte, URL string, store cache.Store) sdk.VCSServer {
	return &bitbucketConsumer{
		ConsumerKey:      consumerKey,
		PrivateKey:       privateKey,
		URL:              URL,
		cache:            store,
		requestTokenURL:  URL + "/plugins/servlet/oauth/request-token",
		authorizationURL: URL + "/plugins/servlet/oauth/authorize",
		accessTokenURL:   URL + "/plugins/servlet/oauth/access-token",
		callbackURL:      oauth1OOB,
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
