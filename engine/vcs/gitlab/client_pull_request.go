package gitlab

import (
	"context"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) PullRequest(ctx context.Context, repo string, id int) (sdk.VCSPullRequest, error) {
	mr, _, err := c.client.MergeRequests.GetMergeRequest(repo, id, nil)
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNotFound,
			"cannot found a merge request for repo %s with id %d", repo, id))
	}
	return sdk.VCSPullRequest{
		ID: mr.IID,
		Base: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				LatestCommit: mr.DiffRefs.BaseSha,
			},
		},
		Head: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				LatestCommit: mr.DiffRefs.HeadSha,
			},
		},
		URL: mr.WebURL,
		User: sdk.VCSAuthor{
			DisplayName: mr.Author.Username,
			Name:        mr.Author.Name,
		},
		Closed: mr.State == "closed",
		Merged: mr.State == "merged",
	}, nil
}

// PullRequests fetch all the pull request for a repository
func (c *gitlabClient) PullRequests(context.Context, string) ([]sdk.VCSPullRequest, error) {
	return []sdk.VCSPullRequest{}, nil
}

// PullRequestComment push a new comment on a pull request
func (c *gitlabClient) PullRequestComment(context.Context, string, sdk.VCSPullRequestCommentRequest) error {
	return nil
}

// PullRequestCreate create a new pullrequest
func (c *gitlabClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	mr, _, err := c.client.MergeRequests.CreateMergeRequest(repo, &gitlab.CreateMergeRequestOptions{
		Title:        &pr.Title,
		SourceBranch: &pr.Head.Branch.DisplayID,
		TargetBranch: &pr.Base.Branch.DisplayID,
	})
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.WithStack(err)
	}

	return sdk.VCSPullRequest{
		ID: mr.IID,
		Base: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				ID:           pr.Base.Branch.DisplayID,
				DisplayID:    pr.Base.Branch.DisplayID,
				LatestCommit: mr.DiffRefs.BaseSha,
			},
		},
		Head: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				ID:           pr.Head.Branch.DisplayID,
				DisplayID:    pr.Head.Branch.DisplayID,
				LatestCommit: mr.DiffRefs.HeadSha,
			},
		},
		URL: mr.WebURL,
		User: sdk.VCSAuthor{
			DisplayName: mr.Author.Username,
			Name:        mr.Author.Name,
		},
		Closed: mr.State == "closed",
		Merged: mr.State == "merged",
	}, nil
}
