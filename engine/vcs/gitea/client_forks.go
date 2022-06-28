package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
