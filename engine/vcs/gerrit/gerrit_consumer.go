package gerrit

import (
	"context"

	ger "github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
)

func (g *gerritConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	// Not implemented for gerrit
	return "", "", nil
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *gerritConsumer) AuthorizeToken(ctx context.Context, state, code string) (string, string, error) {
	// Not implemented for gerrit
	return "", "", nil
}

//GetAuthorized returns an authorized client
func (g *gerritConsumer) GetAuthorizedClient(ctx context.Context, username, password string) (sdk.VCSAuthorizedClient, error) {
	client, err := ger.NewClient(g.URL, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create gerrit client")
	}
	client.Authentication.SetBasicAuth(username, password)

	c := &gerritClient{
		client:              client,
		url:                 g.URL,
		disableStatus:       g.disableStatus,
		disableStatusDetail: g.disableStatusDetail,
		proxyURL:            g.proxyURL,
		sshPort:             g.sshPort,
	}
	return c, nil
}
