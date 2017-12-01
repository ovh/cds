package gitlab

import "github.com/ovh/cds/sdk"

//Branches retrieves the branches
func (c *gitlabClient) Branches(fullname string) ([]sdk.VCSBranch, error) {

	branches, _, err := c.client.Branches.ListBranches(fullname, nil)
	if err != nil {
		return nil, err
	}

	var brs []sdk.VCSBranch
	for _, b := range branches {
		br := sdk.VCSBranch{
			ID:           b.Name,
			DisplayID:    b.Name,
			LatestCommit: b.Commit.ID,
			Default:      false,
			Parents:      nil,
		}
		brs = append(brs, br)
	}

	return brs, nil
}

//Branch retrieves the branch
func (c *gitlabClient) Branch(fullname, branchName string) (*sdk.VCSBranch, error) {

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
