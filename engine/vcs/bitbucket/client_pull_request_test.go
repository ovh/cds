package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
)

func TestPullRequests(t *testing.T) {
	client := getAuthorizedClient(t)
	prs, err := client.PullRequests("CDS/images")
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
}

func TestPullRequestComment(t *testing.T) {
	client := getAuthorizedClient(t)
	prs, err := client.PullRequests("CDS/images")
	test.NoError(t, err)
	assert.NotEmpty(t, prs)
	t.Logf("%v", prs)
	if len(prs) > 0 {
		test.NoError(t, client.PullRequestComment("CDS/images", prs[0].ID, "this is a test"))
	}
}
