package github

import (
	"context"
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
		test.NoError(t, client.PullRequestComment(context.Background(), "ovh/cds", prs[0].ID, "this is a test"))
	}
}
