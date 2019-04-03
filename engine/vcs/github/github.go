package github

import (
	"fmt"
	"github.com/ovh/cds/engine/api/cache"
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

//GithubConsumer implements vcs.Server and it's used to instanciate a githubClient
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
		URL    = "https://github.com"
		APIURL = "https://api.github.com"
	)
	// if the github GitHubURL is passed as an empty string default it to public GitHub
	if githubURL == "" {
		githubURL = URL
	}
	// if the githubAPIURL is empty first check if githubURL was passed in, if not set to default
	if githubAPIURL == "" {
		if githubURL == "" {
			githubAPIURL = APIURL
		} else {
			githubAPIURL = fmt.Sprintf("%s/api/v3", githubURL)
		}
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
