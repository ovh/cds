package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
)

func TestGetV2ActionHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        "github",
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   p.ID,
	}

	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
		ProjectKey:   p.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	e := sdk.Entity{
		Name:                "test-action",
		Commit:              "123456",
		Branch:              "master",
		Type:                sdk.EntityTypeAction,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: test-action
author: cds_team
description: simple test action as code
inputs:
  name:
    description: name of the person to greet
    required: true
    default: Steven
outputs:
  name:
    description: name of the person who was greeted
    value: ${{ steps.hello.outputs.person }}
runs:
  steps:
    - run: echo Welcome in nthis new action
    - id: hello
      run: echo Hello ${{ inputs.name }}
    - run: echo "name=${{ inputs.name }}" >> $CDS_OUTPUT`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	vars := map[string]string{
		"projectKey":           p.Key,
		"vcsIdentifier":        vcsProject.ID,
		"repositoryIdentifier": repo.Name,
		"actionName":           "test-action",
	}
	uri := api.Router.GetRouteV2("GET", api.getActionV2Handler, vars) + "?branch=master"
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	body := w.Body.Bytes()
	var wms sdk.V2Action
	require.NoError(t, json.Unmarshal(body, &wms))

	t.Logf("%+v", wms)
}
