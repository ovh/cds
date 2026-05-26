package forgejo

import (
	"context"
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"
)

// forgejoClient is a forgejo wrapper for CDS vcs. interface
type forgejoClient struct {
	client *forgejoHTTPClient
}

// forgejoConsumer implements vcs.Server and it's used to instantiate a forgejoClient
type forgejoConsumer struct {
	URL      string `json:"url"`
	username string
	token    string
}

func getRepo(fullname string) (string, string, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return "", "", sdk.WithStack(fmt.Errorf("fullname %s must be <project>/<slug>", fullname))
	}
	project := t[0]
	slug := t[1]
	return project, slug, nil
}

// New creates a new forgejo Consumer
func New(URL, username, token string) sdk.VCSServer {
	return &forgejoConsumer{
		URL:      URL,
		username: username,
		token:    token,
	}
}

// GetAuthorizedClient returns an authorized client
func (g *forgejoConsumer) GetAuthorizedClient(_ context.Context, _ sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	return &forgejoClient{
		client: newForgejoHTTPClient(g.URL, g.username, g.token),
	}, nil
}
