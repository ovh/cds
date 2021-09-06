package gitlab

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Branches retrieves the branches
func (c *gitlabClient) Branches(ctx context.Context, fullname string, _ sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	p, _, err := c.client.Projects.GetProject(fullname, nil)
	if err != nil {
		return nil, err
	}

	branches, _, err := c.client.Branches.ListBranches(fullname, nil)
	if err != nil {
		return nil, err
	}

	brs := make([]sdk.VCSBranch, len(branches))
	for i, b := range branches {
		brs[i] = sdk.VCSBranch{
			ID:           b.Name,
			DisplayID:    b.Name,
			LatestCommit: b.Commit.ID,
			Default:      b.Name == p.DefaultBranch,
			Parents:      nil,
		}
	}

	return brs, nil
}

//Branch retrieves the branch
func (c *gitlabClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	if filters.Default {
		p, _, err := c.client.Projects.GetProject(fullname, nil)
		if err != nil {
			return nil, err
		}
		filters.BranchName = p.DefaultBranch
	}
	b, g, err := c.client.Branches.GetBranch(fullname, filters.BranchName)
	if err != nil {
		if g != nil && g.StatusCode == 404 {
			return nil, sdk.WithStack(sdk.ErrNoBranch)
		}
		return nil, err
	}

	br := &sdk.VCSBranch{
		ID:           b.Name,
		DisplayID:    b.Name,
		LatestCommit: b.Commit.ID,
		Default:      false,
		Parents:      nil,
	}

	return br, nil
}
