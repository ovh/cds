package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) PullRequest(ctx context.Context, repo string, id string) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

// PullRequestComment push a new comment on a pull request
func (g *giteaClient) PullRequestComment(ctx context.Context, repo string, prRequest sdk.VCSPullRequestCommentRequest) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, sdk.WithStack(sdk.ErrNotImplemented)
}
