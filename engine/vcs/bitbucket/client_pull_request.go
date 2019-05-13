package bitbucket

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

	var response PullRequest
	if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(err, "Unable to get pullrequest")
	}

	pr, err := b.ToVCSPullRequest(ctx, repo, response)
	if err != nil {
		return sdk.VCSPullRequest{}, err
	}

	return pr, nil
}

func (b *bitbucketClient) PullRequests(ctx context.Context, repo string) ([]sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	bbPR := []PullRequest{}

	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests", project, slug)
	params := url.Values{}

	nextPage := 0
	for {
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
func (b *bitbucketClient) PullRequestComment(ctx context.Context, repo string, prID int, text string) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WithStack(err)
	}
	payload := map[string]string{
		"text": text,
	}
	values, _ := json.Marshal(payload)
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d/comments", project, slug, prID)

	return b.do(ctx, "POST", "core", path, nil, values, nil, &options{asUser: true})
}

func (b *bitbucketClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return pr, sdk.WithStack(err)
	}

	if err := b.GrantWritePermission(ctx, repo); err != nil {
		return pr, err
	}

	request := PullRequest{
		Title:  pr.Title,
		State:  "OPEN",
		Open:   true,
		Closed: false,
		FromRef: PullRequestRef{
			ID: fmt.Sprintf("refs/heads/%s", pr.Head.Branch.DisplayID),
			Repository: PullRequestRefRepository{
				Slug: slug,
				Project: Project{
					Key: project,
				},
			},
		},
		ToRef: PullRequestRef{
			ID: fmt.Sprintf("refs/heads/%s", pr.Base.Branch.DisplayID),
			Repository: PullRequestRefRepository{
				Slug: slug,
				Project: Project{
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

func (b *bitbucketClient) ToVCSPullRequest(ctx context.Context, repo string, pullRequest PullRequest) (sdk.VCSPullRequest, error) {
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
			Name:        pullRequest.Author.User.Username,
			DisplayName: pullRequest.Author.User.DisplayName,
			Email:       pullRequest.Author.User.EmailAddress,
		}
	}

	return pr, nil
}
