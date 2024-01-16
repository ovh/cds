package bitbucketcloud

import (
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const rootURL = "https://api.bitbucket.org/2.0"

// bitbucketcloudClient is a https://bitbucket.org wrapper for CDS vcs. interface
type bitbucketcloudClient struct {
	appPassword          string
	username             string
	DisableStatus        bool
	DisableStatusDetails bool
	Cache                cache.Store
	apiURL               string
	uiURL                string
	proxyURL             string
}

// bitbucketcloudConsumer implements vcs.Server and it's used to instantiate a githubClient
type bitbucketcloudConsumer struct {
	Cache    cache.Store
	uiURL    string
	apiURL   string
	proxyURL string
}

// New creates a new GithubConsumer
func New(apiURL, uiURL, proxyURL string, store cache.Store) sdk.VCSServer {
	return &bitbucketcloudConsumer{
		Cache:    store,
		apiURL:   apiURL,
		uiURL:    uiURL,
		proxyURL: proxyURL,
	}
}
