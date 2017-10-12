package github

import "github.com/ovh/cds/engine/api/cache"
import "github.com/ovh/cds/sdk"

// githubClient is a github.com wrapper for CDS vcs. interface
type githubClient struct {
	ClientID         string
	OAuthToken       string
	DisableSetStatus bool
	DisableStatusURL bool
	Cache            cache.Store
	apiURL           string
	uiURL            string
}

//GithubConsumer implements vcs.Server and it's used to instanciate a githubClient
type githubConsumer struct {
	ClientID     string `json:"client-id"`
	ClientSecret string `json:"-"`
	Cache        cache.Store
}

//New creates a new GithubConsumer
func New(ClientID, ClientSecret string, store cache.Store) sdk.VCSServer {
	return &githubConsumer{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Cache:        store,
	}
}
