package bitbucketcloud

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const rootURL = "https://api.bitbucket.org/2.0"

// bitbucketcloudClient is a https://bitbucket.org wrapper for CDS vcs. interface
type bitbucketcloudClient struct {
	appPassword          string
	username             string
	ClientID             string // DEPRECATED
	OAuthToken           string // DEPRECATED
	RefreshToken         string // DEPRECATED
	DisableStatus        bool
	DisableStatusDetails bool
	Cache                cache.Store
	apiURL               string
	uiURL                string
	proxyURL             string
}

//bitbucketcloudConsumer implements vcs.Server and it's used to instantiate a githubClient
type bitbucketcloudConsumer struct {
	ClientID             string `json:"client-id"`
	ClientSecret         string `json:"-"`
	Cache                cache.Store
	uiURL                string
	apiURL               string
	proxyURL             string
	disableStatus        bool
	disableStatusDetails bool
}

//New creates a new GithubConsumer
func New(apiURL, uiURL, proxyURL string, store cache.Store) sdk.VCSServer {
	return &bitbucketcloudConsumer{
		Cache:    store,
		apiURL:   apiURL,
		uiURL:    uiURL,
		proxyURL: proxyURL,
	}
}

// DEPRECATED VCS
func NewDeprecated(ClientID, ClientSecret, apiURL, uiURL, proxyURL string, store cache.Store, disableStatus, disableStatusDetails bool) sdk.VCSServer {
	return &bitbucketcloudConsumer{
		ClientID:             ClientID,
		ClientSecret:         ClientSecret,
		Cache:                store,
		apiURL:               apiURL,
		uiURL:                uiURL,
		proxyURL:             proxyURL,
		disableStatus:        disableStatus,
		disableStatusDetails: disableStatusDetails,
	}
}

func (client *bitbucketcloudClient) GetAccessToken(_ context.Context) string {
	return client.OAuthToken
}
