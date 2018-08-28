package gitlab

import (
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

var (
	_ sdk.VCSAuthorizedClient = &gitlabClient{}
	_ sdk.VCSServer           = &gitlabConsumer{}
)

// gitlabClient implements VCSAuthorizedClient interface
type gitlabClient struct {
	client              *gitlab.Client
	uiURL               string
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
}

// gitlabConsumer implements vcs.Server and it's used to instanciate a gitlabClient
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
}

// New instanciate a new gitlab consumer
func New(appID, clientSecret, URL, callbackURL, uiURL, proxyURL string, store cache.Store, disableStatus bool, disableStatusDetail bool) sdk.VCSServer {
	return &gitlabConsumer{
		URL:    URL,
		secret: clientSecret,
		cache:  store,
		appID:  appID,
		AuthorizationCallbackURL: callbackURL,
		uiURL:               uiURL,
		proxyURL:            proxyURL,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
	}
}
