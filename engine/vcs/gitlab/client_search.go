package gitlab

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *gitlabClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
