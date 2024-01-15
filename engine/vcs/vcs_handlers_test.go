package vcs

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func Test_accessTokenAuth(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	//Without any header, this should return 401
	req.Header.Set(sdk.HeaderXVCSType, "")

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

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

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
	req.Header.Set(sdk.HeaderXVCSToken, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	t.Logf("Status: %v", rec.Body.String())

	//Asserts
	assert.Equal(t, 200, rec.Code)
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
	req.Header.Set(sdk.HeaderXVCSToken, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 204, rec.Code)
}
