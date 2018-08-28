package bitbucket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
)

func TestRepos(t *testing.T) {
	client := getAuthorizedClient(t)
	repos, err := client.Repos(context.Background())
	test.NoError(t, err)
	assert.NotEmpty(t, repos)
}

func TestRepoByFullname(t *testing.T) {
	client := getAuthorizedClient(t)
	repo, err := client.RepoByFullname(context.Background(), "CDS/images")
	test.NoError(t, err)
	t.Logf("repo: %+v", repo)
}

func TestGrantReadPermission(t *testing.T) {
	client := getAuthorizedClient(t)
	test.NoError(t, client.GrantReadPermission(context.Background(), "CDS/demo"))
}
