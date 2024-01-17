package gerrit

import (
	"context"

	ger "github.com/andygrunwald/go-gerrit"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// gerritClient implements VCSAuthorizedClient interface
type gerritClient struct {
	client               *ger.Client
	url                  string
	disableStatusDetails bool
	sshUsername          string
	sshPort              int
	reviewerName         string
	reviewerToken        string
}

// gerritConsumer implements vcs.Server and it's used to instantiate a gerritClient
type gerritConsumer struct {
	URL           string `json:"url"`
	cache         cache.Store
	sshUsername   string
	sshPort       int
	reviewerName  string
	reviewerToken string
}

// New instantiate a new gerrit consumer
func New(URL string, store cache.Store, sshUsername string, sshPort int, reviewerName, reviewerToken string) sdk.VCSServer {
	return &gerritConsumer{
		URL:           URL,
		cache:         store,
		sshUsername:   sshUsername,
		sshPort:       sshPort,
		reviewerName:  reviewerName,
		reviewerToken: reviewerToken,
	}
}

// GetAuthorized returns an authorized client
func (g *gerritConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	client, err := ger.NewClient(g.URL, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create gerrit client on url %q", g.URL)
	}

	client.Authentication.SetBasicAuth(vcsAuth.Username, vcsAuth.Token)

	c := &gerritClient{
		client:        client,
		url:           g.URL,
		sshPort:       g.sshPort,
		sshUsername:   g.sshUsername,
		reviewerToken: g.reviewerToken,
		reviewerName:  g.reviewerName,
	}
	return c, nil
}
