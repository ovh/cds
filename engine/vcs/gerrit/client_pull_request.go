package gerrit

import (
	"context"

	"github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) PullRequest(ctx context.Context, repo string, id int) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, nil
}

// PullRequests fetch all the pull request for a repository
func (c *gerritClient) PullRequests(ctx context.Context, repo string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	return []sdk.VCSPullRequest{}, nil
}

// PullRequestComment push a new comment on a pull request
func (c *gerritClient) PullRequestComment(ctx context.Context, repo string, prRequest sdk.VCSPullRequestCommentRequest) error {

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
	return nil
}

// PullRequestCreate create a new pullrequest
func (c *gerritClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	return sdk.VCSPullRequest{}, nil
}
