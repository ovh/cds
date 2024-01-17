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
	disableStatusDetails bool
	Cache                cache.Store
	apiURL               string
	uiURL                string
	proxyURL             string
	username             string
	token                string
}

// GithubConsumer implements vcs.Server and it's used to instantiate a githubClient
type githubConsumer struct {
	Cache        cache.Store
	GitHubURL    string
	GitHubAPIURL string
	uiURL        string
	apiURL       string
	proxyURL     string
	username     string
	token        string
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

// GetAuthorized returns an authorized client
func (g *githubConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	c := &githubClient{
		GitHubURL:    g.GitHubURL,    // default value of this field is computed in github.New() func
		GitHubAPIURL: g.GitHubAPIURL, // default value of this field is computed in github.New() func
		Cache:        g.Cache,
		uiURL:        g.uiURL,
		apiURL:       g.apiURL,
		proxyURL:     g.proxyURL,
		username:     vcsAuth.Username,
		token:        vcsAuth.Token,
	}

	return c, c.RateLimit(ctx)
}
