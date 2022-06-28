package gitea

import (
	"context"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

// AuthorizeRedirect returns the request token, the Authorize URL
func (g *giteaConsumer) AuthorizeRedirect(_ context.Context) (string, string, error) {
	return "", "", sdk.WithStack(sdk.ErrNotImplemented)
}

// AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *giteaConsumer) AuthorizeToken(_ context.Context, token, verifier string) (string, string, error) {
	return "", "", sdk.WithStack(sdk.ErrNotImplemented)
}

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
