package api

import (
	"context"
	"encoding/json"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestGetV2WorkerModelsHandler(t *testing.T) {
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
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	e := sdk.Entity{
		Name:                "tmpl1",
		Commit:              "123456",
		Branch:              "master",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: docker-unix
type: docker
spec:
  image: monimage
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  shell: sh -c
  envs:
    MYVAR: toto`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	e2 := sdk.Entity{
		Name:                "tmpl2",
		Commit:              "123456",
		Branch:              "openstack",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: openstack-debian
type: openstack
spec:
  image: monimage
  flavor: maflavor
  pre_cmd: apt-get install docker-ce.
  cmd: ./worker
  post_cmd: sudo shutdown -h now`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e2))

	vars := map[string]string{
		"projectKey":           p.Key,
		"vcsIdentifier":        vcsProject.ID,
		"repositoryIdentifier": repo.Name,
	}
	uri := api.Router.GetRouteV2("GET", api.getWorkerModelsV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	body := w.Body.Bytes()
	var wms []sdk.V2WorkerModel
	require.NoError(t, json.Unmarshal(body, &wms))

	t.Logf("%+v", wms)
	require.Equal(t, 2, len(wms))

	varsGetOne := map[string]string{
		"projectKey":           p.Key,
		"vcsIdentifier":        vcsProject.ID,
		"repositoryIdentifier": repo.Name,
	}
	uriOne := api.Router.GetRouteV2("GET", api.getWorkerModelsV2Handler, varsGetOne)
	test.NotEmpty(t, uri)
	reqOne := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriOne+"?branch=master", nil)

	wOne := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wOne, reqOne)
	require.Equal(t, 200, wOne.Code)

	bodyOne := wOne.Body.Bytes()
	var wm []sdk.V2WorkerModel
	require.NoError(t, json.Unmarshal(bodyOne, &wm))

	require.Equal(t, 1, len(wm))
	require.Equal(t, "docker", wm[0].Type)
}
