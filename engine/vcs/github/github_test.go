package github

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/pkg/browser"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk/log"
)

// TestNew needs githubClientID and githubClientSecret
func TestNewClient(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	assert.NotNil(t, ghConsummer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
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

	ghConsummer := New(clientID, clientSecret, "http://localhost", "", "", cache, true, true)
	return ghConsummer
}

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
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

	ghConsummer := New(clientID, clientSecret, "http://localhost", "", "", cache, true, true)
	cli, err := ghConsummer.GetAuthorizedClient(accessToken, "")
	if err != nil {
		t.Fatalf("Unable to init authorized client (%s): %v", redisHost, err)
	}

	return cli
}

func TestClientAuthorizeToken(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	token, url, err := ghConsummer.AuthorizeRedirect()
	t.Logf("token: %s", token)
	t.Logf("url: %s", url)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, url)
	test.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out := make(chan http.Request, 1)

	go callbackServer(ctx, t, out)

	err = browser.OpenURL(url)
	test.NoError(t, err)

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

	accessToken, accessTokenSecret, err := ghConsummer.AuthorizeToken(state, code)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, accessTokenSecret)
	test.NoError(t, err)

	t.Logf("Token is %s", accessToken)

	ghClient, err := ghConsummer.GetAuthorizedClient(accessToken, accessTokenSecret)
	test.NoError(t, err)
	assert.NotNil(t, ghClient)
}

func TestAuthorizedClient(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)
}

func TestRepos(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	repos, err := ghClient.Repos()
	test.NoError(t, err)
	assert.NotEmpty(t, repos)
}

func TestRepoByFullname(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	repo, err := ghClient.RepoByFullname("ovh/cds")
	test.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestBranches(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branches, err := ghClient.Branches("ovh/cds")
	test.NoError(t, err)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	branch, err := ghClient.Branch("ovh/cds", "master")
	test.NoError(t, err)
	assert.NotNil(t, branch)
}

func TestCommits(t *testing.T) {
	ghClient := getNewAuthorizedClient(t)
	assert.NotNil(t, ghClient)

	commits, err := ghClient.Commits("ovh/cds", "master", "", "")
	test.NoError(t, err)
	assert.NotNil(t, commits)
}
