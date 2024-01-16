package gitea

import (
	"context"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

// GetAuthorizedClient returns an authorized client
func (g *giteaConsumer) GetAuthorizedClient(_ context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	client, err := gg.NewClient(g.URL, gg.SetBasicAuth(g.username, g.token))
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return &giteaClient{
		consumer: *g,
		proxyURL: g.proxyURL,
		username: g.username,
		token:    g.token,
		client:   client,
	}, nil
}
