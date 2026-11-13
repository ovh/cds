package bitbucketserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	repo, err := client.RepoByFullname(context.Background(), "CDS/tests")
	test.NoError(t, err)
	t.Logf("repo: %+v", repo)
}

func TestToVCSRepoParent(t *testing.T) {
	var fork, origin Repo
	require.NoError(t, json.Unmarshal([]byte(`{"slug":"cds","project":{"key":"~KHOPA"},"origin":{"slug":"cds","project":{"key":"CDS"}}}`), &fork))
	require.NoError(t, json.Unmarshal([]byte(`{"slug":"cds","project":{"key":"CDS"}}`), &origin))

	client := &bitbucketClient{}
	require.Equal(t, "CDS/cds", client.ToVCSRepo(fork).Parent)
	require.Empty(t, client.ToVCSRepo(origin).Parent)
}
