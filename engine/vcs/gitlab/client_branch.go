package gitlab

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Branches retrieves the branches
func (c *gitlabClient) Branches(ctx context.Context, fullname string) ([]sdk.VCSBranch, error) {

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
			Default:      false,
			Parents:      nil,
		}
	}

	return brs, nil
}

//Branch retrieves the branch
func (c *gitlabClient) Branch(ctx context.Context, fullname, branchName string) (*sdk.VCSBranch, error) {

	b, _, err := c.client.Branches.GetBranch(fullname, branchName)
	if err != nil {
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
