package gitea

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (g *giteaClient) PullRequest(ctx context.Context, repo string, id string) (sdk.VCSPullRequest, error) {
	ret := sdk.VCSPullRequest{}
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return ret, fmt.Errorf("invalid repo gitea: %s", repo)
	}

	i, _ := strconv.ParseInt(id, 10, 64)

	pr, _, err := g.client.GetPullRequest(t[0], t[1], i)
	if err != nil {
		return ret, sdk.WrapError(err, "unable to get gitea pull-request repo:%v id:%v", repo, id)
	}

	ret = sdk.VCSPullRequest{
		ID:    int(pr.Index),
		State: string(pr.State),
		Head: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				DisplayID:    pr.Head.Ref,
				LatestCommit: pr.Head.Sha,
			},
		},
	}
	return ret, nil
}

func (g *giteaClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return nil, fmt.Errorf("invalid repo gitea: %s", repo)
	}

	prs, _, err := g.client.ListRepoPullRequests(t[0], t[1], gitea.ListPullRequestsOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get gitea pull-requests from repo:%v", repo)
	}

	ret := make([]sdk.VCSPullRequest, 0)
	for _, pr := range prs {
		ret = append(ret, sdk.VCSPullRequest{
			ID:    int(pr.Index),
			State: string(pr.State),
			Head: sdk.VCSPushEvent{
				Branch: sdk.VCSBranch{
					DisplayID:    pr.Head.Ref,
					LatestCommit: pr.Head.Sha,
				},
			},
		})
	}
	return ret, nil
}

// PullRequestComment push a new comment on a pull request
func (g *giteaClient) PullRequestComment(ctx context.Context, repo string, prRequest sdk.VCSPullRequestCommentRequest) error {
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return fmt.Errorf("invalid repo gitea: %s", repo)
	}

	log.Debug(ctx, "PullRequestComment> trying post comment %s", prRequest.Message)

	opt := gitea.CreatePullReviewOptions{
		Body: prRequest.Message,
	}
	s, _, err := g.client.CreatePullReview(t[0], t[1], int64(prRequest.ID), opt)
	if err != nil {
		return sdk.WrapError(err, "unable to create pull-request comment on repo:%v id:%v", repo, prRequest.ID)
	}

	log.Debug(ctx, "PullRequestComment> comment %d %s", s.ID, s.Body)

	return nil
}

func (g *giteaClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, sdk.WithStack(sdk.ErrNotImplemented)
}
