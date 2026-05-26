package forgejo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (f *forgejoClient) PullRequest(ctx context.Context, repo string, id string) (sdk.VCSPullRequest, error) {
	ret := sdk.VCSPullRequest{}
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return ret, err
	}

	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return ret, fmt.Errorf("invalid pull-request id %q: %w", id, err)
	}

	var pr PullRequest
	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repoName, i)
	if _, err := f.client.get(ctx, apiPath, &pr); err != nil {
		return ret, sdk.WrapError(err, "unable to get forgejo pull-request repo:%v id:%v", repo, id)
	}

	return toVCSPullRequest(&pr), nil
}

func (f *forgejoClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return nil, err
	}

	const maxResults = 100
	const pageSize = 50
	basePath := fmt.Sprintf("/repos/%s/%s/pulls", owner, repoName)

	// Map CDS state to Forgejo API state values: open, closed, all.
	// Forgejo has no "merged" state; merged PRs are a subset of "closed".
	switch opts.State {
	case sdk.VCSPullRequestStateOpen:
		basePath += "?state=open"
	case sdk.VCSPullRequestStateClosed, sdk.VCSPullRequestStateMerged:
		basePath += "?state=closed"
	case sdk.VCSPullRequestStateAll:
		basePath += "?state=all"
	}

	var allPRs []*PullRequest
	for page := 1; ; page++ {
		var pagePRs []*PullRequest
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err := f.client.get(ctx, apiPath, &pagePRs); err != nil {
			return nil, sdk.WrapError(err, "unable to get forgejo pull-requests from repo:%v", repo)
		}
		allPRs = append(allPRs, pagePRs...)
		if len(pagePRs) < pageSize || len(allPRs) >= maxResults {
			break
		}
	}
	if len(allPRs) > maxResults {
		allPRs = allPRs[:maxResults]
	}

	ret := make([]sdk.VCSPullRequest, 0, len(allPRs))
	for _, pr := range allPRs {
		ret = append(ret, toVCSPullRequest(pr))
	}
	return ret, nil
}

func toVCSPullRequest(pr *PullRequest) sdk.VCSPullRequest {
	vcs := sdk.VCSPullRequest{
		ID:    int(pr.Index),
		State: string(pr.State),
	}
	if pr.Head != nil {
		vcs.Head = sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				DisplayID:    pr.Head.Ref,
				LatestCommit: pr.Head.Sha,
			},
		}
	}
	return vcs
}

// PullRequestComment push a new comment on a pull request
func (f *forgejoClient) PullRequestComment(ctx context.Context, repo string, prRequest sdk.VCSPullRequestCommentRequest) error {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return err
	}

	log.Debug(ctx, "PullRequestComment> trying post comment %s", prRequest.Message)

	opt := CreatePullReviewOptions{
		Body: prRequest.Message,
	}

	var s PullReview
	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repoName, prRequest.ID)
	if _, err := f.client.post(ctx, apiPath, opt, &s); err != nil {
		return sdk.WrapError(err, "unable to create pull-request comment on repo:%v id:%v", repo, prRequest.ID)
	}

	log.Debug(ctx, "PullRequestComment> comment %d %s", s.ID, s.Body)

	return nil
}

func (f *forgejoClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, sdk.WithStack(sdk.ErrNotImplemented)
}
