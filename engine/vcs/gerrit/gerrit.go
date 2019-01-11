package gerrit

import (
	g "github.com/andygrunwald/go-gerrit"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// gerritClient implements VCSAuthorizedClient interface
type gerritClient struct {
	client              *g.Client
	url                 string
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
	sshPort             int
}

// gerritConsumer implements vcs.Server and it's used to instanciate a gerritClient
type gerritConsumer struct {
	URL                 string `json:"url"`
	cache               cache.Store
	proxyURL            string
	disableStatus       bool
	disableStatusDetail bool
	sshPort             int
}

// New instanciate a new gerrit consumer
func New(URL, proxyURL string, store cache.Store, disableStatus bool, disableStatusDetail bool, sshPort int) sdk.VCSServer {
	return &gerritConsumer{
		URL:                 URL,
		cache:               store,
		proxyURL:            proxyURL,
		disableStatus:       disableStatus,
		disableStatusDetail: disableStatusDetail,
		sshPort:             sshPort,
	}
}
