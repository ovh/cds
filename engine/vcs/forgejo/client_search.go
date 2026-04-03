package forgejo

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
