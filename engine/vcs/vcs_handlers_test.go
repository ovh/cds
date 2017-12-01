package vcs

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
)

func Test_getAllVCSServersHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     "client_id",
			ClientSecret: "client_secret",
		},
	})
	test.NoError(t, err)

	err = s.addServerConfiguration("gitlab", ServerConfiguration{
		URL: "https://gitlab.com",
		Gitlab: &GitlabServerConfiguration{
			Secret: "mysecret",
		},
	})
	test.NoError(t, err)

	err = s.addServerConfiguration("bitbucket", ServerConfiguration{
		URL: "https://bitbucket.com",
		Bitbucket: &BitbucketServerConfiguration{
			ConsumerKey: "cds",
			PrivateKey:  "private key",
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{}
	uri := s.Router.GetRoute("GET", s.getAllVCSServersHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)

	var servers = map[string]ServerConfiguration{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &servers))

	assert.Len(t, servers, 3)

	log.Debug("Body: %s", rec.Body.String())

	//Prepare request
	vars = map[string]string{
		"name": "github",
	}
	uri = s.Router.GetRoute("GET", s.getVCSServersHandler, vars)
	test.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)

	var srv = ServerConfiguration{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srv))

	t.Logf("%s", rec.Body.String())

	//Prepare request
	vars = map[string]string{
		"name": "github",
	}
	uri = s.Router.GetRoute("GET", s.getVCSServersHooksHandler, vars)
	test.NotEmpty(t, uri)
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
	test.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_accessTokenAuth(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     "client_id",
			ClientSecret: "client_secret",
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	//Without any header, this should return 401
	req.Header.Set(HeaderXAccessToken, "")
	req.Header.Set(HeaderXAccessTokenSecret, "")

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 401, rec.Code)
}

func Test_getReposHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)
	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name": "github",
	}
	uri := s.Router.GetRoute("GET", s.getReposHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getRepoHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": "ovh",
		"repo":  "cds",
	}
	uri := s.Router.GetRoute("GET", s.getRepoHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getBranchesHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": "ovh",
		"repo":  "cds",
	}
	uri := s.Router.GetRoute("GET", s.getBranchesHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getBranchHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": "ovh",
		"repo":  "cds",
	}
	uri := s.Router.GetRoute("GET", s.getBranchHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)
	q := req.URL.Query()
	q.Set("branch", "vcs/old-code")
	req.URL.RawQuery = q.Encode()

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getCommitsHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":  "github",
		"owner": "fsamin",
		"repo":  "go-dump",
	}
	uri := s.Router.GetRoute("GET", s.getCommitsHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)
	q := req.URL.Query()
	q.Set("since", "61dc819dc47bc6b556b2c4e1293f919d492f1f5a")
	q.Set("branch", "fsamin/fix-2017-07-28-10-07-56")
	req.URL.RawQuery = q.Encode()

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func Test_getCommitHandler(t *testing.T) {
	cfg := test.LoadTestingConf(t)

	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	checkConfigGithub(cfg, t)

	err = s.addServerConfiguration("github", ServerConfiguration{
		URL: "https://github.com",
		Github: &GithubServerConfiguration{
			ClientID:     cfg["githubClientID"],
			ClientSecret: cfg["githubClientSecret"],
		},
	})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"name":   "github",
		"owner":  "ovh",
		"repo":   "cds",
		"commit": "a38dfc7cc835aadf6a112e8a540dd52cca79cc29",
	}
	uri := s.Router.GetRoute("GET", s.getCommitHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "GET", uri, nil)

	token := base64.StdEncoding.EncodeToString([]byte(cfg["githubAccessToken"]))
	req.Header.Set(HeaderXAccessToken, token)
	//accessTokenSecret is useless for github, let's give the same token
	req.Header.Set(HeaderXAccessTokenSecret, token)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
}

func checkConfigGithub(cfg map[string]string, t *testing.T) {
	if cfg["githubClientID"] == "" || cfg["githubClientSecret"] == "" {
		log.Debug("Skip Github Test - no configuration")
		t.SkipNow()
	}
}
