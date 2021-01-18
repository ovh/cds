package github

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/browser"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// TestNew needs githubClientID and githubClientSecret
func TestNewClient(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	assert.NotNil(t, ghConsummer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	clientID := cfg["githubClientID"]
	clientSecret := cfg["githubClientSecret"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if clientID == "" && clientSecret == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := New(clientID, clientSecret, "", "", "http://localhost", "", "", "", "", cache, true, true)
	return ghConsummer
}

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	clientID := cfg["githubClientID"]
	clientSecret := cfg["githubClientSecret"]
	accessToken := cfg["githubAccessToken"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if clientID == "" && clientSecret == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := New(clientID, clientSecret, "", "", "http://localhost", "", "", "", "", cache, true, true)
	cli, err := ghConsummer.GetAuthorizedClient(context.Background(), accessToken, "", 0)
	if err != nil {
		t.Fatalf("Unable to init authorized client (%s): %v", redisHost, err)
	}

	return cli
}

func TestClientAuthorizeToken(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	token, url, err := ghConsummer.AuthorizeRedirect(context.Background())
	t.Logf("token: %s", token)
	t.Logf("url: %s", url)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, url)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out := make(chan http.Request, 1)

	go callbackServer(ctx, t, out)

	err = browser.OpenURL(url)
	require.NoError(t, err)

	r, ok := <-out
	t.Logf("Chan request closed? %v", !ok)
	t.Logf("OAuth request 2: %+v", r)
	assert.NotNil(t, r)

	cberr := r.FormValue("error")
	errDescription := r.FormValue("error_description")
	errURI := r.FormValue("error_uri")

	assert.Empty(t, cberr)
	assert.Empty(t, errDescription)
	assert.Empty(t, errURI)

	code := r.FormValue("code")
	state := r.FormValue("state")

	assert.NotEmpty(t, code)
	assert.NotEmpty(t, state)

	accessToken, accessTokenSecret, err := ghConsummer.AuthorizeToken(context.Background(), state, code)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, accessTokenSecret)
	require.NoError(t, err)

	t.Logf("Token is %s", accessToken)

	ghClient, err := ghConsummer.GetAuthorizedClient(context.Background(), accessToken, accessTokenSecret, 0)
	require.NoError(t, err)
	assert.NotNil(t, ghClient)
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

	branches, err := ghClient.Branches(context.Background(), "ovh/cds")
	require.NoError(t, err)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branch, err := ghClient.Branch(context.Background(), "ovh/cds", "master")
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
