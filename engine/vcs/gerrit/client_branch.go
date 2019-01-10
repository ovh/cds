package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Branches retrieves the branches
func (c *gerritClient) Branches(ctx context.Context, fullname string) ([]sdk.VCSBranch, error) {
	return nil, nil
}

//Branch retrieves the branch
func (c *gerritClient) Branch(ctx context.Context, fullname, branchName string) (*sdk.VCSBranch, error) {
	return nil, nil
}
