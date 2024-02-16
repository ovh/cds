package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/hatchery"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestGetWorkerModelV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        sdk.VCSTypeGithub,
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

	myscret := "my-registry-password"
	encodedSecret, err := project.EncryptWithBuiltinKey(context.TODO(), db, p.ID, "mysecret", myscret)
	require.NoError(t, err)

	e := sdk.Entity{
		Name:                "model1",
		Commit:              "123456",
		Ref:                 "refs/heads/master",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: fmt.Sprintf(`name: docker-unix
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

	// CREATE HATCHERY
	h := sdk.Hatchery{Name: sdk.RandomString(10)}
	uri := api.Router.GetRouteV2("POST", api.postHatcheryHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &h)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	var hatcheryCreated sdk.Hatchery
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hatcheryCreated))

	// GET CONSUMER AND CREATE SESSION
	consumer, err := authentication.LoadHatcheryConsumerByName(context.TODO(), db, hatcheryCreated.Name)
	require.NoError(t, err)
	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, hatchery.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Get Worker model with secret
	vars := map[string]string{
		"projectKey":           p.Key,
		"vcsIdentifier":        vcsProject.ID,
		"repositoryIdentifier": repo.Name,
		"workerModelName":      e.Name,
	}
	uriGetModel := api.Router.GetRouteV2("GET", api.getWorkerModelV2Handler, vars)
	test.NotEmpty(t, uriGetModel)
	reqGetModel := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uriGetModel+"?withSecrets=true&branch=master", nil)
	wGetModel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGetModel, reqGetModel)
	require.Equal(t, 200, wGetModel.Code)
	var modelDbWithClearSecret sdk.V2WorkerModel
	require.NoError(t, json.Unmarshal(wGetModel.Body.Bytes(), &modelDbWithClearSecret))

	require.Equal(t, modelDbWithClearSecret.Type, "docker")
	var dockerSpec sdk.V2WorkerModelDockerSpec
	require.NoError(t, json.Unmarshal(modelDbWithClearSecret.Spec, &dockerSpec))
	require.Equal(t, myscret, dockerSpec.Password)

}

func TestGetV2WorkerModelsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        sdk.VCSTypeGithub,
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
		Name:                "tmpl1",
		Commit:              "123456",
		Ref:                 "refs/heads/master",
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
		Ref:                 "refs/heads/openstack",
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
