package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) PullRequests(ctx context.Context, repo string) ([]sdk.VCSPullRequest, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WrapError(err, "vcs> bitbucket> PullRequests>")
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
			return nil, sdk.WrapError(err, "vcs> bitbucket> PullRequests> Unable to get repos")
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
		pr := sdk.VCSPullRequest{
			ID: r.ID,
		}
		if len(r.Links.Self) > 0 {
			pr.URL = r.Links.Self[0].Href
		}
		//
		pr.Base = sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				ID: strings.Replace(r.ToRef.ID, "refs/heads/", "", 1),
			},
		}
		pr.Head = sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				ID: strings.Replace(r.FromRef.ID, "refs/heads/", "", 1),
			},
		}
		pr.User = sdk.VCSAuthor{
			Name:        r.Author.User.Name,
			DisplayName: r.Author.User.DisplayName,
			Email:       r.Author.User.EmailAddress,
		}

		baseBranch, err := b.Branch(ctx, repo, pr.Base.Branch.ID)
		if err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> PullRequests> unable to get branch %s", baseBranch)
		}
		pr.Base.Branch = *baseBranch
		pr.Base.Commit = sdk.VCSCommit{
			Hash: baseBranch.LatestCommit,
		}

		headBranch, err := b.Branch(ctx, r.FromRef.Repository.Project.Key+"/"+r.FromRef.Repository.Slug, pr.Head.Branch.ID)
		if err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> PullRequests> unable to get branch %s", headBranch)
		}
		pr.Head.Branch = *headBranch

		prs[i] = pr
	}

	return prs, nil
}

// PullRequestComment push a new comment on a pull request
func (b *bitbucketClient) PullRequestComment(ctx context.Context, repo string, prID int, text string) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> PullRequestComment>")
	}
	payload := map[string]string{
		"text": text,
	}
	values, _ := json.Marshal(payload)
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d/comments", project, slug, prID)

	return b.do(ctx, "POST", "core", path, nil, values, nil, &options{asUser: true})
}
