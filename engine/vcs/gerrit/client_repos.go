package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Repos returns the list of accessible repositories
func (c *gerritClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	return nil, nil
}

//RepoByFullname returns the repo from its fullname
func (c *gerritClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	return sdk.VCSRepo{}, nil
}

func (c *gerritClient) GrantReadPermission(ctx context.Context, repo string) error {
	return nil
}
