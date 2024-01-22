package gitea

import (
	"context"
	"fmt"
	"strings"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// giteaClient is a gitea wrapper for CDS vcs. interface
type giteaClient struct {
	username string
	token    string
	proxyURL string
	consumer giteaConsumer
	client   *gg.Client
}

// giteaConsumer implements vcs.Server and it's used to instantiate a giteaClient
type giteaConsumer struct {
	URL      string `json:"url"`
	cache    cache.Store
	apiURL   string
	uiURL    string
	proxyURL string
	username string
	token    string
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

// New creates a new bitbucket Consumer
func New(URL, apiURL, uiURL, proxyURL string, store cache.Store, username, token string) sdk.VCSServer {
	return &giteaConsumer{
		URL:      URL,
		apiURL:   apiURL,
		uiURL:    uiURL,
		proxyURL: proxyURL,
		cache:    store,
		username: username,
		token:    token,
	}
}

// GetAuthorizedClient returns an authorized client
func (g *giteaConsumer) GetAuthorizedClient(_ context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	client, err := gg.NewClient(g.URL, gg.SetBasicAuth(g.username, g.token))
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return &giteaClient{
		consumer: *g,
		proxyURL: g.proxyURL,
		username: g.username,
		token:    g.token,
		client:   client,
	}, nil
}
