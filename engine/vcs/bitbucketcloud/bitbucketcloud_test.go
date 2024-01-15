package bitbucketcloud

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
)

// TestNew needs bitbucketCloudClientID and bitbucketCloudClientSecret
func TestNewClient(t *testing.T) {
	bbConsumer := getNewConsumer(t)
	assert.NotNil(t, bbConsumer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	cache, err := cache.New(redisHost, redisPassword, 0, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	bbConsumer := New("http://localhost", "", "", cache)
	return bbConsumer
}

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	cache, err := cache.New(redisHost, redisPassword, 0, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	bbConsumer := New("http://localhost", "", "", cache)
	vcsAuth := sdk.VCSAuth{
		Type: sdk.VCSTypeBitbucketCloud,
	}
	cli, err := bbConsumer.GetAuthorizedClient(context.Background(), vcsAuth)
	if err != nil {
		t.Fatalf("Unable to init authorized client (%s): %v", redisHost, err)
	}

	return cli
}

func TestAuthorizedClient(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)
}

func TestRepos(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	repos, err := bbClient.Repos(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, repos)
}

func TestRepoByFullname(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	repo, err := bbClient.RepoByFullname(context.Background(), "bnjjj/test")
	require.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestBranches(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	branches, err := bbClient.Branches(context.Background(), "bnjjj/test", sdk.VCSBranchesFilter{})
	require.NoError(t, err)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	branch, err := bbClient.Branch(context.Background(), "bnjjj/test", sdk.VCSBranchFilters{BranchName: "master"})
	require.NoError(t, err)
	assert.NotNil(t, branch)
}

func TestCommits(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	commits, err := bbClient.Commits(context.Background(), "bnjjj/test", "master", "HEAD", "HEAD")
	require.NoError(t, err)
	assert.NotNil(t, commits)
}
