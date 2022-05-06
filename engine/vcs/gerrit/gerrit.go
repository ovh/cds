package gerrit

import (
	"context"

	g "github.com/andygrunwald/go-gerrit"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// gerritClient implements VCSAuthorizedClient interface
type gerritClient struct {
	client              *g.Client
	url                 string
	disableStatus       bool
	disableStatusDetail bool
	sshPort             int
	username            string
	reviewerName        string
	reviewerToken       string
}

// gerritConsumer implements vcs.Server and it's used to instantiate a gerritClient
type gerritConsumer struct {
	URL                 string `json:"url"`
	cache               cache.Store
	disableStatus       bool
	disableStatusDetail bool
	sshPort             int
	reviewerName        string
	reviewerToken       string
}

// New instantiate a new gerrit consumer
func New(URL string, store cache.Store, disableStatus bool, disableStatusDetail bool, sshPort int, reviewerName, reviewerToken string) sdk.VCSServer {
	return &gerritConsumer{
		URL:                 URL,
		cache:               store,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
		sshPort:             sshPort,
		reviewerName:        reviewerName,
		reviewerToken:       reviewerToken,
	}
}

func (c *gerritClient) GetAccessToken(_ context.Context) string {
	return c.username
}
