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
	return toSDKPullRequest(repo, *mr), nil
}

// PullRequests fetch all the pull request for a repository
func (c *gitlabClient) PullRequests(ctx context.Context, repo string) ([]sdk.VCSPullRequest, error) {
	mrs, _, err := c.client.MergeRequests.ListProjectMergeRequests(repo, nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	res := make([]sdk.VCSPullRequest, 0, len(mrs))
	for i := range mrs {
		res = append(res, toSDKPullRequest(repo, *mrs[i]))
	}
	return res, nil
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
	return toSDKPullRequest(repo, *mr), nil
}

func toSDKPullRequest(repo string, mr gitlab.MergeRequest) sdk.VCSPullRequest {
	return sdk.VCSPullRequest{
		ID: mr.IID,
		Base: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				DisplayID:    mr.TargetBranch,
				LatestCommit: mr.DiffRefs.BaseSha,
			},
		},
		Head: sdk.VCSPushEvent{
			Repo: repo,
			Branch: sdk.VCSBranch{
				DisplayID:    mr.SourceBranch,
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
	}
}
