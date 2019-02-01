package gerrit

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) PullRequest(ctx context.Context, repo string, id int) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, nil
}

// PullRequests fetch all the pull request for a repository
func (c *gerritClient) PullRequests(context.Context, string) ([]sdk.VCSPullRequest, error) {
	return []sdk.VCSPullRequest{}, nil
}

// PullRequestComment push a new comment on a pull request
func (c *gerritClient) PullRequestComment(context.Context, string, int, string) error {
	return nil
}

// PullRequestCreate create a new pullrequest
func (c *gerritClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, nil
}
