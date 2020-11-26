package gerrit

import (
	"context"
	"net/http"

	"github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
)

// ⚠ For gerrit, PullRequest means Change

// PullRequest Get a gerrit change
func (c *gerritClient) PullRequest(_ context.Context, _ string, id string) (sdk.VCSPullRequest, error) {
	change, resp, err := c.client.Changes.GetChange(id, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return sdk.VCSPullRequest{}, sdk.WrapError(sdk.ErrNotFound, "unable to find change %s", id)
		}
		return sdk.VCSPullRequest{}, sdk.WithStack(err)
	}
	if change == nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(sdk.ErrNotFound, "unable to find change %s", id)
	}
	return c.toVCSPullRequest(*change), nil
}

// PullRequests fetch all the pull request for a repository
func (c *gerritClient) PullRequests(_ context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	queryOpts := &gerrit.QueryChangeOptions{
		QueryOptions: gerrit.QueryOptions{
			Query: []string{
				"project:" + repo,
			},
		},
	}
	switch opts.State {
	case sdk.VCSPullRequestStateOpen:
		queryOpts.Query = append(queryOpts.Query, "open")
	case sdk.VCSPullRequestStateMerged:
		queryOpts.Query = append(queryOpts.Query, "merged")
	case sdk.VCSPullRequestStateClosed:
		queryOpts.Query = append(queryOpts.Query, "abandoned")
	}

	changes, _, err := c.client.Changes.QueryChanges(queryOpts)

	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if changes == nil {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to list changes")
	}
	prs := make([]sdk.VCSPullRequest, 0, len(*changes))
	for _, ch := range *changes {
		prs = append(prs, c.toVCSPullRequest(ch))
	}
	return prs, nil
}

// PullRequestComment push a new comment on a change
func (c *gerritClient) PullRequestComment(_ context.Context, _ string, prRequest sdk.VCSPullRequestCommentRequest) error {

	// Use reviewer account to post the review
	c.client.Authentication.SetBasicAuth(c.reviewerName, c.reviewerToken)

	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#review-input
	ri := gerrit.ReviewInput{
		Message: prRequest.Message,
		Tag:     "CDS",
		Labels:  nil,
		Notify:  "OWNER", // Send notification to the owner
	}

	if _, _, err := c.client.Changes.SetReview(prRequest.ChangeID, prRequest.Revision, &ri); err != nil {
		return sdk.WrapError(err, "unable to set gerrit review")
	}

	return nil
}

// PullRequestCreate create a new pullrequest
func (c *gerritClient) PullRequestCreate(_ context.Context, _ string, _ sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, nil
}

func (c *gerritClient) toVCSPullRequest(change gerrit.ChangeInfo) sdk.VCSPullRequest {
	pr := sdk.VCSPullRequest{
		ChangeID: change.ID,
		Closed:   change.Status == "ABANDONED",
		Merged:   change.Status == "MERGED",
		Base: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				ID:        change.Branch,
				DisplayID: change.Branch,
			},
		},
		Head: sdk.VCSPushEvent{
			Commit: sdk.VCSCommit{
				Hash: change.CurrentRevision,
			},
			Branch: sdk.VCSBranch{
				ID:        change.Branch,
				DisplayID: change.Branch,
			},
		},
		Revision: change.CurrentRevision,
		User: sdk.VCSAuthor{
			Name:        change.Owner.Name,
			DisplayName: change.Owner.Username,
			Email:       change.Owner.Email,
		},
		Title:   change.Subject,
		Updated: change.Updated.Time,
	}
	return pr
}
