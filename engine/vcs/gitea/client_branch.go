package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Branches(ctx context.Context, fullname string, filters sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) GetDefaultBranch(ctx context.Context, fullname string) (*sdk.VCSBranch, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
