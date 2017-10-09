package vcs

import (
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

	err = s.addServerConfiguration(&ServerConfiguration{
		Name: "github",
		URL:  "https://github.com",
		Github: &GithubServerConfiguration{
			Secret: "mysecret",
		},
	})
	test.NoError(t, err)

	err = s.addServerConfiguration(&ServerConfiguration{
		Name: "gitlab",
		URL:  "https://gitlab.com",
		Gitlab: &GitlabServerConfiguration{
			Secret: "mysecret",
		},
	})
	test.NoError(t, err)

	err = s.addServerConfiguration(&ServerConfiguration{
		Name: "bitbucket",
		URL:  "https://bitbucket.com",
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

	var servers = []ServerConfiguration{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &servers))

	assert.Len(t, servers, 3)

	log.Debug("Body: %s", rec.Body.String())

}
