package bitbucketserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestPullRequests(t *testing.T) {
	client := getAuthorizedClient(t)
	prs, err := client.PullRequests(context.Background(), "CDS/tests", sdk.VCSPullRequestOptions{})
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
}

func TestPullRequestComment(t *testing.T) {
	client := getAuthorizedClient(t)
	prs, err := client.PullRequests(context.Background(), "CDS/tests", sdk.VCSPullRequestOptions{})
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
	if len(prs) > 0 {
		r := sdk.VCSPullRequestCommentRequest{Message: "this is a test from unit test TestPullRequestComment"}
		r.ID = prs[0].ID
		test.NoError(t, client.PullRequestComment(context.Background(), "CDS/tests", r))
	}
}
