package bitbucketserver

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/assert"
)

func TestCommits(t *testing.T) {
	client := getAuthorizedClient(t)
	commits, err := client.Commits(context.Background(), "CDS/tests", "master", "", "")
	test.NoError(t, err)
	assert.NotEmpty(t, commits)
	t.Logf("%+v", commits)
}

func TestCommit(t *testing.T) {
	client := getAuthorizedClient(t)
	commit, err := client.Commit(context.Background(), "CDS/tests", "0b6d50472e9b2c03d72a422ea11bf3faa570d0bd")
	test.NoError(t, err)
	t.Logf("%+v", commit)
	assert.Contains(t, commit.Author.Email, "yvonnick.esnault")

}
