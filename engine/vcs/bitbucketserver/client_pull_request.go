package bitbucketserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) PullRequest(ctx context.Context, repo string, id int) (sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.WithStack(err)
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d", project, slug, id)
	params := url.Values{}

	var response sdk.BitbucketServerPullRequest
	if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(err, "Unable to get pullrequest")
	}

	pr, err := b.ToVCSPullRequest(ctx, repo, response)
	if err != nil {
		return sdk.VCSPullRequest{}, err
	}

	return pr, nil
}

func (b *bitbucketClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	bbPR := []sdk.BitbucketServerPullRequest{}

	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests", project, slug)
	params := url.Values{}

	switch opts.State {
	case sdk.VCSPullRequestStateOpen:
		params.Set("state", "OPEN")
	case sdk.VCSPullRequestStateMerged:
		params.Set("state", "MERGED")
	case sdk.VCSPullRequestStateClosed:
		params.Set("state", "DECLINED")
	}

	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response PullRequestResponse
		if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}

		bbPR = append(bbPR, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	prs := make([]sdk.VCSPullRequest, len(bbPR))
	for i, r := range bbPR {
		pr, err := b.ToVCSPullRequest(ctx, repo, r)
		if err != nil {
			return nil, err
		}
		prs[i] = pr
	}
	return prs, nil
}

// PullRequestComment push a new comment on a pull request
func (b *bitbucketClient) PullRequestComment(ctx context.Context, repo string, prRequest sdk.VCSPullRequestCommentRequest) error {

	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WithStack(err)
	}
	payload := map[string]string{
		"text": prRequest.Message,
	}
	values, err := json.Marshal(payload)
	if err != nil {
		return sdk.WithStack(err)
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d/comments", project, slug, prRequest.ID)

	canWrite, err := b.UserHasWritePermission(ctx, repo)
	if err != nil {
		return err
	}
	if !canWrite {
		if err := b.GrantWritePermission(ctx, repo); err != nil {
			return err
		}
	}
	return b.do(ctx, "POST", "core", path, nil, values, nil, &options{asUser: true})
}

func (b *bitbucketClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return pr, sdk.WithStack(err)
	}

	canWrite, err := b.UserHasWritePermission(ctx, repo)
	if err != nil {
		return pr, err
	}
	if !canWrite {
		if err := b.GrantWritePermission(ctx, repo); err != nil {
			return pr, err
		}
	}

	request := sdk.BitbucketServerPullRequest{
		Title:  pr.Title,
		State:  "OPEN",
		Open:   true,
		Closed: false,
		FromRef: sdk.BitbucketServerRef{
			ID: fmt.Sprintf("refs/heads/%s", pr.Head.Branch.DisplayID),
			Repository: sdk.BitbucketServerRepository{
				Slug: slug,
				Project: sdk.BitbucketServerProject{
					Key: project,
				},
			},
		},
		ToRef: sdk.BitbucketServerRef{
			ID: fmt.Sprintf("refs/heads/%s", pr.Base.Branch.DisplayID),
			Repository: sdk.BitbucketServerRepository{
				Slug: slug,
				Project: sdk.BitbucketServerProject{
					Key: project,
				},
			},
		},
		Locked:       false,
		Author:       nil,
		Participants: nil,
	}

	values, _ := json.Marshal(request)
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests", project, slug)

	if err := b.do(ctx, "POST", "core", path, nil, values, &request, &options{asUser: true}); err != nil {
		return pr, sdk.WithStack(err)
	}

	return b.ToVCSPullRequest(ctx, repo, request)
}

func (b *bitbucketClient) ToVCSPullRequest(ctx context.Context, repo string, pullRequest sdk.BitbucketServerPullRequest) (sdk.VCSPullRequest, error) {
	pr := sdk.VCSPullRequest{
		ID:     pullRequest.ID,
		Closed: pullRequest.Closed,
		Merged: pullRequest.State == "MERGED",
		Base: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				ID:           strings.Replace(pullRequest.ToRef.ID, "refs/heads/", "", 1),
				DisplayID:    pullRequest.ToRef.DisplayID,
				LatestCommit: pullRequest.ToRef.LatestCommit,
			},
		},
		Head: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				ID:           strings.Replace(pullRequest.FromRef.ID, "refs/heads/", "", 1),
				DisplayID:    pullRequest.FromRef.DisplayID,
				LatestCommit: pullRequest.FromRef.LatestCommit,
			},
		},
	}
	if len(pullRequest.Links.Self) > 0 {
		pr.URL = pullRequest.Links.Self[0].Href
	}
	if pullRequest.Author != nil {
		pr.User = sdk.VCSAuthor{
			Name:        pullRequest.Author.User.Name,
			DisplayName: pullRequest.Author.User.DisplayName,
			Email:       pullRequest.Author.User.EmailAddress,
		}
	}

	return pr, nil
}
