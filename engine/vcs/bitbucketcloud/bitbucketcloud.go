package bitbucketcloud

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const rootURL = "https://api.bitbucket.org/2.0"

// bitbucketcloudClient is a https://bitbucket.org wrapper for CDS vcs. interface
type bitbucketcloudClient struct {
	ClientID            string
	OAuthToken          string
	RefreshToken        string
	DisableStatus       bool
	DisableStatusDetail bool
	Cache               cache.Store
	apiURL              string
	uiURL               string
	proxyURL            string
}

//bitbucketcloudConsumer implements vcs.Server and it's used to instantiate a githubClient
type bitbucketcloudConsumer struct {
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
func New(ClientID, ClientSecret, apiURL, uiURL, proxyURL string, store cache.Store, disableStatus, disableStatusDetail bool) sdk.VCSServer {
	return &bitbucketcloudConsumer{
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

func (c *bitbucketcloudClient) GetAccessToken(_ context.Context) string {
	return c.OAuthToken
}
