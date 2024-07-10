package gitlab

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

// TestNew needs githubClientID and githubClientSecret

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	gitlabUsername := cfg["gitlabUsername"]
	gitlabToken := cfg["gitlabToken"]

	if gitlabUsername == "" && gitlabToken == "" {
		t.Logf("Unable to read gitlab configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(sdk.RedisConf{Host: redisHost, Password: redisPassword, DbIndex: 0}, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	glConsummer := New("https://gitlab.com", "http://localhost:8081", "", cache, gitlabUsername, gitlabToken)

	vcsAuth := sdk.VCSAuth{
		Type:     sdk.VCSTypeGitlab,
		Username: gitlabUsername,
		Token:    gitlabToken,
	}
	cli, err := glConsummer.GetAuthorizedClient(context.Background(), vcsAuth)
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

	repo, err := ghClient.RepoByFullname(context.Background(), "yvonnick.esnault/demo")

	require.NoError(t, err)
	t.Logf("%+v", repo)
	assert.NotNil(t, repo)
}

func TestBranches(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branches, err := ghClient.Branches(context.Background(), "yvonnick.esnault/demo", sdk.VCSBranchesFilter{})
	require.NoError(t, err)
	t.Logf("%+v", branches)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branch, err := ghClient.Branch(context.Background(), "yvonnick.esnault/demo", sdk.VCSBranchFilters{BranchName: "master"})
	require.NoError(t, err)
	t.Logf("%+v", branch)
	assert.NotNil(t, branch)
}

func TestCommits(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	commits, err := ghClient.Commits(context.Background(), "yvonnick.esnault/demo", "master", "", "")
	require.NoError(t, err)
	t.Logf("%+v", commits)
	assert.NotNil(t, commits)
}
