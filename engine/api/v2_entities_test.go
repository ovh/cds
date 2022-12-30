package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetEntitiesHandler(t *testing.T) {
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
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	myscret := "my-registry-password"
	encodedSecret, err := project.EncryptWithBuiltinKey(context.TODO(), db, p.ID, "mysecret", myscret)
	require.NoError(t, err)

	e := sdk.Entity{
		Name:                "model1",
		Commit:              "123456",
		Branch:              "master",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: fmt.Sprintf(`name: model1
type: docker
spec:
  image: monimage
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  password: %s
  shell: sh -c
  envs:
    MYVAR: toto`, encodedSecret),
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	vars := map[string]string{
		"projectKey":           p.Key,
		"vcsIdentifier":        vcsProject.ID,
		"repositoryIdentifier": repo.Name,
		"entityType":           sdk.EntityTypeWorkerModel,
		"entityName":           "model1",
	}
	uri := api.Router.GetRouteV2("GET", api.getEntityHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?branch=master", nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	body := w.Body.Bytes()
	var wm sdk.Entity
	require.NoError(t, json.Unmarshal(body, &wm))

	t.Logf("%+v", wm)
}
