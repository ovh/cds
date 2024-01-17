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
	username             string
	token                string
	proxyURL             string
	disableStatusDetails bool
	consumer             bitbucketConsumer
}

// bitbucketConsumer implements vcs.Server and it's used to instantiate a bitbucketClient
type bitbucketConsumer struct {
	ConsumerKey string `json:"consumer_key"`
	PrivateKey  []byte `json:"-"`
	URL         string `json:"url"`
	cache       cache.Store
	apiURL      string
	uiURL       string
	proxyURL    string
}

// New creates a new bitbucket Consumer
func New(URL, apiURL, uiURL, proxyURL string, store cache.Store, username, token string) sdk.VCSServer {
	return &bitbucketConsumer{
		URL:      URL,
		apiURL:   apiURL,
		uiURL:    uiURL,
		proxyURL: proxyURL,
		cache:    store,
	}
}

// GetAuthorized returns an authorized client
func (g *bitbucketConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	return &bitbucketClient{
		consumer: *g,
		proxyURL: g.proxyURL,
		username: vcsAuth.Username,
		token:    vcsAuth.Token,
	}, nil
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
