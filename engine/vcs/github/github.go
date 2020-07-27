package github

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// githubClient is a github.com wrapper for CDS vcs. interface
type githubClient struct {
	GitHubURL           string
	GitHubAPIURL        string
	ClientID            string
	OAuthToken          string
	DisableStatus       bool
	DisableStatusDetail bool
	Cache               cache.Store
	apiURL              string
	uiURL               string
	proxyURL            string
	username            string
	token               string
}

//GithubConsumer implements vcs.Server and it's used to instantiate a githubClient
type githubConsumer struct {
	ClientID            string `json:"client-id"`
	ClientSecret        string `json:"-"`
	Cache               cache.Store
	GitHubURL           string
	GitHubAPIURL        string
	uiURL               string
	apiURL              string
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
	username            string
	token               string
}

//New creates a new GithubConsumer
func New(ClientID, ClientSecret, githubURL, githubAPIURL, apiURL, uiURL, proxyURL, username, token string, store cache.Store, disableStatus, disableStatusDetail bool) sdk.VCSServer {
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
		ClientID:            ClientID,
		ClientSecret:        ClientSecret,
		GitHubURL:           githubURL,
		GitHubAPIURL:        githubAPIURL,
		Cache:               store,
		apiURL:              apiURL,
		uiURL:               uiURL,
		proxyURL:            proxyURL,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
		username:            username,
		token:               token,
	}
}

func (c *githubClient) GetAccessToken(_ context.Context) string {
	return c.OAuthToken
}
