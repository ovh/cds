package github

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	githubUsername := cfg["githubUsername"]
	githubToken := cfg["githubToken"]

	if githubUsername == "" && githubToken == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(context.TODO(), sdk.RedisConf{Host: redisHost, Password: redisPassword, DbIndex: 0}, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := New("", "", "http://localhost", "", "", cache)
	vcsAuth := sdk.VCSAuth{
		Type:     sdk.VCSTypeGithub,
		Token:    githubToken,
		Username: githubUsername,
	}
	cli, err := ghConsummer.GetAuthorizedClient(context.Background(), vcsAuth)
	if err != nil {
		t.Fatalf("Unable to init authorized client (%s): %v", redisHost, err)
	}

	return cli
}

func TestAuthorizedClient(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)
}

func TestRepos(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	repos, err := ghClient.Repos(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, repos)
}

func TestRepoByFullname(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	repo, err := ghClient.RepoByFullname(context.Background(), "ovh/cds")
	require.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestBranches(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branches, err := ghClient.Branches(context.Background(), "ovh/cds", sdk.VCSBranchesFilter{})
	require.NoError(t, err)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branch, err := ghClient.Branch(context.Background(), "ovh/cds", sdk.VCSBranchFilters{BranchName: "master"})
	require.NoError(t, err)
	assert.NotNil(t, branch)
}

func TestCommits(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	commits, err := ghClient.Commits(context.Background(), "ovh/cds", "master", "", "")
	require.NoError(t, err)
	assert.NotNil(t, commits)
}
