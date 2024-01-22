package gitlab

import (
	"context"
	"net/http"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	_ sdk.VCSAuthorizedClient = &gitlabClient{}
	_ sdk.VCSServer           = &gitlabConsumer{}
)

// gitlabClient implements VCSAuthorizedClient interface
type gitlabClient struct {
	client               *gitlab.Client
	uiURL                string
	proxyURL             string
	disableStatus        bool
	disableStatusDetails bool
}

// gitlabConsumer implements vcs.Server and it's used to instantiate a gitlabClient
type gitlabConsumer struct {
	URL                      string `json:"url"`
	cache                    cache.Store
	AuthorizationCallbackURL string
	uiURL                    string
	proxyURL                 string
	username                 string
	personalAccessToken      string
}

// New instantiate a new gitlab consumer
func New(URL, uiURL, proxyURL string, store cache.Store, username, token string) sdk.VCSServer {
	var url = URL
	if url == "" {
		url = "https://gitlab.com"
	}
	return &gitlabConsumer{
		URL:                 url,
		cache:               store,
		uiURL:               uiURL,
		proxyURL:            proxyURL,
		username:            username,
		personalAccessToken: token,
	}
}

// GetAuthorized returns an authorized client
func (g *gitlabConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	gclient := gitlab.NewClient(httpClient, vcsAuth.Token)
	c := &gitlabClient{
		client:   gclient,
		uiURL:    g.uiURL,
		proxyURL: g.proxyURL,
	}
	c.client.SetBaseURL(g.URL + "/api/v4")
	return c, nil
}
