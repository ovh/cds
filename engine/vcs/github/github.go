package github

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// githubClient is a github.com wrapper for CDS vcs. interface
type githubClient struct {
	ClientID            string
	OAuthToken          string
	DisableStatus       bool
	DisableStatusDetail bool
	Cache               cache.Store
	apiURL              string
	uiURL               string
	proxyURL            string
}

//GithubConsumer implements vcs.Server and it's used to instanciate a githubClient
type githubConsumer struct {
	ClientID            string `json:"client-id"`
	ClientSecret        string `json:"-"`
	Cache               cache.Store
	uiURL               string
	apiURL              string
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
}

//New creates a new GithubConsumer
func New(ClientID, ClientSecret string, apiURL, uiURL, proxyURL string, store cache.Store, disableStatus, disableStatusDetail bool) sdk.VCSServer {
	return &githubConsumer{
		ClientID:            ClientID,
		ClientSecret:        ClientSecret,
		Cache:               store,
		apiURL:              apiURL,
		uiURL:               uiURL,
		proxyURL:            proxyURL,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
	}
}
