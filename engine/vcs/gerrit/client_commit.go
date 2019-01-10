package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Commits returns commit data from a given starting commit, between two commits
//The commits may be identified by branch or tag name or by hash.
func (c *gerritClient) Commits(ctx context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	return nil, nil
}

//Commit retrieves a specific according to a hash
func (c *gerritClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	return sdk.VCSCommit{}, nil
}

func (c *gerritClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	return nil, nil
}
