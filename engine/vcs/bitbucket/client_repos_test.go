package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
)

func TestRepos(t *testing.T) {
	client := getAuthorizedClient(t)
	repos, err := client.Repos()
	test.NoError(t, err)
	assert.NotEmpty(t, repos)
}

func TestRepoByFullname(t *testing.T) {
	client := getAuthorizedClient(t)
	repo, err := client.RepoByFullname("CDS/images")
	test.NoError(t, err)
	t.Logf("repo: %+v", repo)
}
