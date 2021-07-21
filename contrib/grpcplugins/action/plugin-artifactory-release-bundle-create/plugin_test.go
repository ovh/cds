package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/stretchr/testify/require"
)

func runHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/distribution/api/v1/release_bundle", r.RequestURI)
		content, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close() // nolint
		require.Equal(t, string(`{"name":"the_name","version":"1.0.0","dry_run":false,"sign_immediately":true,"description":"the_description","spec":{"queries":[{"aql":"items.find({\"$and\":[{\"$and\":[{\"$eq\":\"my-repository\",\"repo\":null}]},{\"$and\":[{\"@cds_version\":\"${cds_version}\"},{\"@cds_application\":\"my-application\"}]}]}).include(\"name\",\"repo\",\"path\",\"actual_md5\",\"actual_sha1\",\"size\",\"type\",\"modified\",\"created\")"}]}}`), string(content))
		w.WriteHeader(http.StatusCreated)
	}))
}

func TestRun(t *testing.T) {
	ts := runHTTPServer(t)
	t.Cleanup(func() {
		ts.Close()
	})

	var p = artifactoryReleaseBundleCreatePlugin{}
	var q = actionplugin.ActionQuery{
		Options: map[string]string{
			"name":          "the_name",
			"version":       "1.0.0",
			"description":   "the_description",
			"url":           ts.URL + "/artifactory/",
			"token":         "1234567890",
			"release_notes": "",
			"specification": `
files:
- aql:
    items.find:
      "$and":
      - "$and":
        - repo:
          "$eq": my-repository
      - "$and":
        - "@cds_version": "${cds_version}"
        - "@cds_application": my-application`,
		},
	}
	res, err := p.Run(context.TODO(), &q)
	require.NoError(t, err)
	require.NotNil(t, res)
}
