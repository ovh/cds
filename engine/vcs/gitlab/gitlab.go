package gitlab

import (
	"context"

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
	client              *gitlab.Client
	accessToken         string
	uiURL               string
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
}

// gitlabConsumer implements vcs.Server and it's used to instantiate a gitlabClient
type gitlabConsumer struct {
	URL                      string `json:"url"`
	appID                    string
	secret                   string
	cache                    cache.Store
	AuthorizationCallbackURL string
	uiURL                    string
	proxyURL                 string
	disableStatus            bool
	disableStatusDetail      bool
	username                 string
	personalAccessToken      string
}

// New instantiate a new gitlab consumer
func New(URL, uiURL, proxyURL string, store cache.Store, username, token string, disableStatusDetail bool) sdk.VCSServer {
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
		disableStatusDetail: disableStatusDetail,
	}
}

func NewDeprecated(appID, clientSecret, URL, callbackURL, uiURL, proxyURL string, store cache.Store, disableStatus bool, disableStatusDetail bool) sdk.VCSServer {
	return &gitlabConsumer{
		URL:                      URL,
		secret:                   clientSecret,
		cache:                    store,
		appID:                    appID,
		AuthorizationCallbackURL: callbackURL,
		uiURL:                    uiURL,
		proxyURL:                 proxyURL,
		disableStatus:            disableStatus,
		disableStatusDetail:      disableStatusDetail,
	}
}

func (c *gitlabClient) GetAccessToken(_ context.Context) string {
	return c.accessToken
}
