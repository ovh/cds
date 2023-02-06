package gitlab

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rockbears/log"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) PullRequest(ctx context.Context, repo string, id string) (sdk.VCSPullRequest, error) {
	gitlabPRID, err := strconv.Atoi(id)
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(sdk.ErrWrongRequest, "invalid merge request identifier: %s", id)
	}
	mr, _, err := c.client.MergeRequests.GetMergeRequest(repo, gitlabPRID, nil)
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNotFound,
			"cannot found a merge request for repo %s with id %d", repo, gitlabPRID))
	}
	return toSDKPullRequest(repo, *mr), nil
}

// PullRequests fetch all the pull request for a repository
func (c *gitlabClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	var gitlabOpts gitlab.ListProjectMergeRequestsOptions

	switch opts.State {
	case sdk.VCSPullRequestStateOpen:
		gitlabOpts.State = gitlab.String("opened")
	case sdk.VCSPullRequestStateMerged:
		gitlabOpts.State = gitlab.String("merged")
	case sdk.VCSPullRequestStateClosed:
		gitlabOpts.State = gitlab.String("closed")
	}

	mrs, _, err := c.client.MergeRequests.ListProjectMergeRequests(repo, &gitlabOpts)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	res := make([]sdk.VCSPullRequest, 0, len(mrs))
	for i := range mrs {
		mr, err := c.PullRequest(ctx, repo, fmt.Sprintf("%d", mrs[i].IID))
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		res = append(res, mr)
	}
	return res, nil
}

// PullRequestComment push a new comment on a pull request
func (c *gitlabClient) PullRequestComment(ctx context.Context, repo string, prReq sdk.VCSPullRequestCommentRequest) error {
	if c.disableStatus {
		log.Warn(ctx, "gitlab.PullRequestComment>  ⚠ Gitlab statuses are disabled")
		return nil
	}
	_, _, err := c.client.Notes.CreateMergeRequestNote(repo, prReq.ID, &gitlab.CreateMergeRequestNoteOptions{
		Body: gitlab.String(prReq.Message),
	})
	if err != nil {
		return sdk.WithStack(err)
	}
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
	pr := sdk.VCSPullRequest{
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
			Slug:        mr.Author.Username,
		},
		Closed: mr.State == "closed",
		Merged: mr.State == "merged",
		MergeBy: sdk.VCSAuthor{
			DisplayName: mr.MergedBy.Username,
			Name:        mr.MergedBy.Name,
			Slug:        mr.MergedBy.Username,
		},
	}
	if mr.UpdatedAt != nil {
		pr.Updated = *mr.UpdatedAt
	}
	return pr
}
