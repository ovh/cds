package bitbucketserver

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
