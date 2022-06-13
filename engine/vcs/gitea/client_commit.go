package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Commits(ctx context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	return sdk.VCSCommit{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
