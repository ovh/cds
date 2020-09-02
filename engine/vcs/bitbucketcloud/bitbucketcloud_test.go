package bitbucketcloud

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/pkg/browser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk/log"
)

var currentAccessToken string
var currentRefreshToken string

// TestNew needs bitbucketCloudClientID and bitbucketCloudClientSecret
func TestNewClient(t *testing.T) {
	bbConsumer := getNewConsumer(t)
	assert.NotNil(t, bbConsumer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	clientID := cfg["bitbucketCloudClientID"]
	clientSecret := cfg["bitbucketCloudClientSecret"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if clientID == "" && clientSecret == "" {
		t.Logf("Unable to read bitbucket cloud configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	bbConsumer := New(clientID, clientSecret, "http://localhost", "", "", cache, true, true)
	return bbConsumer
}

func getNewAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	clientID := cfg["bitbucketCloudClientID"]
	clientSecret := cfg["bitbucketCloudClientSecret"]
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

	bbConsumer := New(clientID, clientSecret, "http://localhost", "", "", cache, true, true)
	cli, err := bbConsumer.GetAuthorizedClient(context.Background(), currentAccessToken, currentRefreshToken, time.Now().Unix())
	if err != nil {
		t.Fatalf("Unable to init authorized client (%s): %v", redisHost, err)
	}

	return cli
}

func TestClientAuthorizeToken(t *testing.T) {
	bbConsumer := getNewConsumer(t)
	token, url, err := bbConsumer.AuthorizeRedirect(context.Background())
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

	assert.NotEmpty(t, code)

	accessToken, refreshToken, err := bbConsumer.AuthorizeToken(context.Background(), "", code)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	require.NoError(t, err)

	currentAccessToken = accessToken
	currentRefreshToken = refreshToken
	t.Logf("Token is %s", accessToken)

	bbClient, err := bbConsumer.GetAuthorizedClient(context.Background(), accessToken, refreshToken, time.Now().Unix())
	require.NoError(t, err)
	assert.NotNil(t, bbClient)
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

	branches, err := bbClient.Branches(context.Background(), "bnjjj/test")
	require.NoError(t, err)
	assert.NotEmpty(t, branches)
}

func TestBranch(t *testing.T) {
	bbClient := getNewAuthorizedClient(t)
	assert.NotNil(t, bbClient)

	branch, err := bbClient.Branch(context.Background(), "bnjjj/test", "master")
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
