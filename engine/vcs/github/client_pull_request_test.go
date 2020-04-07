package github

import (
	"context"
	"github.com/ovh/cds/sdk"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
)

func TestPullRequests(t *testing.T) {
	client := getNewAuthorizedClient(t)
	prs, err := client.PullRequests(context.Background(), "ovh/cds")
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
}

func TestPullRequestComment(t *testing.T) {
	client := getNewAuthorizedClient(t)
	prs, err := client.PullRequests(context.Background(), "ovh/cds")
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
	if len(prs) > 0 {
		r := sdk.VCSPullRequestCommentRequest{Message: "this is a test"}
		r.ID = prs[0].ID
		test.NoError(t, client.PullRequestComment(context.Background(), "ovh/cds", r))
	}
}
