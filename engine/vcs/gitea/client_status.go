package gitea

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (client *giteaClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (client *giteaClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
