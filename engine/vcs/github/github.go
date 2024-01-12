package github

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// githubClient is a github.com wrapper for CDS vcs. interface
type githubClient struct {
	GitHubURL            string
	GitHubAPIURL         string
	ClientID             string
	OAuthToken           string
	DisableStatus        bool
	DisableStatusDetails bool
	Cache                cache.Store
	apiURL               string
	uiURL                string
	proxyURL             string
	username             string
	token                string
}

// GithubConsumer implements vcs.Server and it's used to instantiate a githubClient
type githubConsumer struct {
	ClientID             string `json:"client-id"`
	ClientSecret         string `json:"-"`
	Cache                cache.Store
	GitHubURL            string
	GitHubAPIURL         string
	uiURL                string
	apiURL               string
	proxyURL             string
	disableStatus        bool
	disableStatusDetails bool
	username             string
	token                string
}

// New creates a new GithubConsumer
func New(githubURL, githubAPIURL, apiURL, uiURL, proxyURL string, store cache.Store) sdk.VCSServer {
	//Github const
	const (
		publicURL    = "https://github.com"
		publicAPIURL = "https://api.github.com"
	)
	// if the githubURL is passed as an empty string default it to public GitHub
	if githubURL == "" {
		githubURL = publicURL
	}
	// if the githubAPIURL is passed as an empty string default it to public GitHub
	if githubAPIURL == "" {
		githubAPIURL = publicAPIURL
	}
	return &githubConsumer{
		GitHubURL:    githubURL,
		GitHubAPIURL: githubAPIURL,
		Cache:        store,
		apiURL:       apiURL,
		uiURL:        uiURL,
		proxyURL:     proxyURL,
	}
}

func (c *githubClient) GetAccessToken(_ context.Context) string {
	return c.OAuthToken
}
