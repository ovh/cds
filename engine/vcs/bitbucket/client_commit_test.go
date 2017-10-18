package bitbucket

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/assert"
)

func TestCommits(t *testing.T) {
	client := getAuthorizedClient(t)
	commits, err := client.Commits("CDS/images", "master", "", "")
	test.NoError(t, err)
	assert.NotEmpty(t, commits)
	t.Logf("%+v", commits)
}

func TestCommit(t *testing.T) {
	client := getAuthorizedClient(t)
	commit, err := client.Commit("CDS/images", "1244a1ccf125a80abeb191fce98d3cdcad13b8c2")
	test.NoError(t, err)
	t.Logf("%+v", commit)
}
