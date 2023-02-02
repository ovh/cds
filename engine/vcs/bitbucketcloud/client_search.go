package bitbucketcloud

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (client *bitbucketcloudClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
