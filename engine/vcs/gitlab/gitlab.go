package gitlab

import (
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// GitlabClient implements RepositoriesManagerClient interface
type gitlabClient struct {
	client              *gitlab.Client
	uiURL               string
	disableStatus       bool
	disableStatusDetail bool
}

//gitlabConsumer implements vcs.Server and it's used to instanciate a githubClient
type gitlabConsumer struct {
	URL                      string `json:"url"`
	appID                    string
	secret                   string
	cache                    cache.Store
	AuthorizationCallbackURL string
	uiURL                    string
	disableStatus            bool
	disableStatusDetail      bool
}

func New(appID string, clientSecret string, URL string, callbackURL string, uiURL string, store cache.Store, disableStatus bool, disableStatusDetail bool) sdk.VCSServer {
	return &gitlabConsumer{
		URL:    URL,
		secret: clientSecret,
		cache:  store,
		appID:  appID,
		AuthorizationCallbackURL: callbackURL,
		uiURL:               uiURL,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
	}
}
