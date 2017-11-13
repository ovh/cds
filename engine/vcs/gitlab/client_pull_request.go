package gitlab

import "github.com/ovh/cds/sdk"

// PullRequests fetch all the pull request for a repository
func (c *gitlabClient) PullRequests(string) ([]sdk.VCSPullRequest, error) {
	return []sdk.VCSPullRequest{}, nil
}
