package gerrit

import (
	"context"
	"strings"

	"github.com/ovh/cds/sdk"
)

//Branches retrieves the branches
func (c *gerritClient) Branches(ctx context.Context, fullname string, _ sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	branches, _, err := c.client.Projects.ListBranches(fullname, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list branches")
	}
	if branches == nil {
		return []sdk.VCSBranch{}, nil
	}
	bs := make([]sdk.VCSBranch, 0, len(*branches))

	var defaultBranch string
	for _, b := range *branches {
		if b.Ref == "HEAD" {
			defaultBranch = b.Revision
			break
		}
	}

	for _, b := range *branches {
		if b.Ref == "HEAD" || strings.HasPrefix(b.Ref, "refs/meta/") {
			continue
		}
		newBranch := sdk.VCSBranch{
			ID:           b.Ref,
			LatestCommit: b.Revision,
			DisplayID:    strings.Replace(b.Ref, "refs/heads/", "", -1),
			Default:      false,
		}
		if newBranch.DisplayID == defaultBranch {
			newBranch.Default = true
		}
		bs = append(bs, newBranch)
	}
	return bs, nil
}

//Branch retrieves the branch
func (c *gerritClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	branch, _, err := c.client.Projects.GetBranch(fullname, filters.BranchName)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get branch")
	}

	if branch == nil {
		return nil, sdk.WithStack(sdk.ErrNoBranch)
	}

	newBranch := sdk.VCSBranch{
		ID:           branch.Ref,
		LatestCommit: branch.Revision,
		DisplayID:    strings.Replace(branch.Ref, "refs/heads/", "", -1),
		Default:      false,
	}

	head, _, err := c.client.Projects.GetBranch(fullname, "HEAD")
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get default branch")
	}
	if head != nil && head.Revision == newBranch.DisplayID {
		newBranch.Default = true
	}
	return &newBranch, nil
}
