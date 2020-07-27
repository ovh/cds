package vcs

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getAllVCSServersHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     "client_id",
			ClientSecret: "client_secret",
		},
	})
	require.NoError(t, err)

	err = s.addServerConfiguration("gitlab", ServerConfiguration{
		URL: "https://gitlab.com",
		Gitlab: &GitlabServerConfiguration{
			Secret: "mysecret",
		},
	})
	require.NoError(t, err)

	err = s.addServerConfiguration("bitbucket", ServerConfiguration{
		URL: "https://bitbucket.com",
		Bitbucket: &BitbucketServerConfiguration{
			ConsumerKey: "cds",
			PrivateKey:  "private key",
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{}
	uri := s.Router.GetRoute("GET", s.getAllVCSServersHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)

	var servers = map[string]ServerConfiguration{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &servers))

	assert.Len(t, servers, 3)

	log.Debug("Body: %s", rec.Body.String())

	//Prepare request
	vars = map[string]string{
		"name": "github",
	}
	uri = s.Router.GetRoute("GET", s.getVCSServersHandler, vars)
	require.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)

	var srv = ServerConfiguration{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srv))

	t.Logf("%s", rec.Body.String())

	//Prepare request
	vars = map[string]string{
		"name": "github",
	}
	uri = s.Router.GetRoute("GET", s.getVCSServersHooksHandler, vars)
	require.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)

	//Prepare request
	vars = map[string]string{
		"name": "github",
	}
	uri = s.Router.GetRoute("GET", s.getVCSServersPollingHandler, vars)
	require.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_accessTokenAuth(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)
	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     "client_id",
			ClientSecret: "client_secret",
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	//Without any header, this should return 401
	req.Header.Set(sdk.HeaderXAccessToken, "")
	req.Header.Set(sdk.HeaderXAccessTokenSecret, "")

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 401, rec.Code)
}

func Test_getReposHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)
	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getRepoHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": cfg["githubOwner"],
		"repo":  cfg["githubRepo"],
	}
	uri := s.Router.GetRoute("GET", s.getRepoHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getBranchesHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": cfg["githubOwner"],
		"repo":  cfg["githubRepo"],
	}
	uri := s.Router.GetRoute("GET", s.getBranchesHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getBranchHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": cfg["githubOwner"],
		"repo":  cfg["githubRepo"],
	}
	uri := s.Router.GetRoute("GET", s.getBranchHandler, vars)
	require.NotEmpty(t, uri)
	f := func(req *http.Request) {
		q := req.URL.Query()
		q.Set("branch", cfg["githubBranch"])
		req.URL.RawQuery = q.Encode()
	}
	req := newRequest(t, s, "GET", uri, nil, f)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	btes, _ := httputil.DumpRequest(req, false)
	t.Log(string(btes))

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getCommitsHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": cfg["githubCommitOwner"],
		"repo":  cfg["githubCommitRepo"],
	}

	uri := s.Router.GetRoute("GET", s.getCommitsHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil, func(req *http.Request) {
		q := req.URL.Query()
		q.Set("since", cfg["githubCommitSince"])
		q.Set("branch", cfg["githubCommitBranch"])
		req.URL.RawQuery = q.Encode()
	})

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getCommitHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":   "github",
		"owner":  cfg["githubOwner"],
		"repo":   cfg["githubRepo"],
		"commit": cfg["githubCommit"],
	}
	uri := s.Router.GetRoute("GET", s.getCommitHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getCommitStatusHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":   "github",
		"owner":  cfg["githubOwner"],
		"repo":   cfg["githubRepo"],
		"commit": cfg["githubCommit"],
	}
	uri := s.Router.GetRoute("GET", s.getCommitStatusHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	t.Logf("Status: %v", rec.Body.String())

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func checkConfigGithub(cfg map[string]string, t *testing.T) {
	if cfg["githubURL"] == "" {
		cfg["githubURL"] = "https://github.com"
	}
	if cfg["githubAPIURL"] == "" {
		cfg["githubAPIURL"] = "https://api.github.com"
	}
	if cfg["githubRepo"] == "" {
		cfg["githubRepo"] = "cds"
	}
	if cfg["githubOwner"] == "" {
		cfg["githubOwner"] = "ovh"
	}
	if cfg["githubBranch"] == "" {
		cfg["githubBranch"] = "gh-pages"
	}
	if cfg["githubCommit"] == "" {
		cfg["githubCommit"] = "a38dfc7cc835aadf6a112e8a540dd52cca79cc29"
	}
	if cfg["githubCommitOwner"] == "" {
		cfg["githubCommitOwner"] = "fsamin"
	}
	if cfg["githubCommitRepo"] == "" {
		cfg["githubCommitRepo"] = "go-dump"
	}
	if cfg["githubCommitSince"] == "" {
		cfg["githubCommitSince"] = "6e06b7fed23aeed808b4d60e8a085f9b9c4b45af"
	}
	if cfg["githubCommitBranch"] == "" {
		cfg["githubCommitBranch"] = "master"
	}

	if cfg["githubClientID"] == "" || cfg["githubClientSecret"] == "" {
		log.Debug("Skip Github Test - no configuration")
		t.SkipNow()
	}
}

func Test_postRepoGrantHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)

	//Bootstrap the service
	s, err := newTestService(t)
	require.NoError(t, err)

	checkConfigGithub(cfg, t)

	t.Logf("Testing grant with %s", cfg["githubUsername"])

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: cfg["githubURL"],
		Github: &GithubServerConfiguration{
			APIURL:       cfg["githubAPIURL"],
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
			Username:     cfg["githubUsername"],
			Token:        cfg["githubPassword"],
		},
	})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": cfg["githubCommitOwner"],
		"repo":  cfg["githubCommitRepo"],
	}
	uri := s.Router.GetRoute("POST", s.postRepoGrantHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "POST", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(sdk.HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 204, rec.Code)
}
