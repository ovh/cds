package gerrit

import (
	"context"

	ger "github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
)

// GetAuthorized returns an authorized client
func (g *gerritConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	client, err := ger.NewClient(g.URL, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create gerrit client on url %q", g.URL)
	}

	client.Authentication.SetBasicAuth(vcsAuth.Username, vcsAuth.Token)

	c := &gerritClient{
		client:               client,
		url:                  g.URL,
		disableStatusDetails: g.disableStatusDetails,
		sshPort:              g.sshPort,
		sshUsername:          g.sshUsername,
		reviewerToken:        g.reviewerToken,
		reviewerName:         g.reviewerName,
	}
	return c, nil
}
