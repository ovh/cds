package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
)

func Test_getProjectsV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_ = assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{}
	uri := api.Router.GetRouteV2("GET", api.getProjectsV2Handler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	projs := []sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projs))
	require.Greater(t, len(projs), 0)
}

func Test_updateGetDeleteProjectV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	proj.Name = "updated_" + sdk.RandomString(10)

	// UPDATE
	vars := map[string]string{
		"projectKey": proj.Key,
	}

	bts, _ := json.Marshal(proj)
	reader := bytes.NewReader(bts)

	uri := api.Router.GetRouteV2("PUT", api.updateProjectV2Handler, vars)
	req, err := http.NewRequest("PUT", uri, reader)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// GET
	uriGet := api.Router.GetRouteV2("GET", api.getProjectV2Handler, vars)
	reqGet, err := http.NewRequest("GET", uriGet, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, reqGet, u, pass)
	// Do the request
	wGET := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGET, reqGet)
	require.Equal(t, 200, wGET.Code)

	projGET := sdk.Project{}
	test.NoError(t, json.Unmarshal(wGET.Body.Bytes(), &projGET))
	require.Equal(t, proj.Name, projGET.Name)

	// DELETE
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectV2Handler, vars)
	reqDel, err := http.NewRequest("DELETE", uriDelete, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, reqDel, u, pass)
	// Do the request
	wDel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDel, reqDel)
	require.Equal(t, 204, wDel.Code)
}

func Test_deleteProjectV2Handler_CleanSchedulerHooks(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProj.ID, "org/myrepo")

	// Create an entity for the workflow
	e := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		Name:                "my-workflow",
		Ref:                 "refs/heads/master",
		Commit:              "123456",
		Head:                true,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	// Insert scheduler hooks for this project
	wh1 := sdk.V2WorkflowHook{
		ProjectKey:     proj.Key,
		VCSName:        vcsProj.Name,
		RepositoryName: repo.Name,
		EntityID:       e.ID,
		WorkflowName:   "my-workflow",
		Ref:            "refs/heads/master",
		Commit:         "123456",
		Type:           sdk.WorkflowHookTypeScheduler,
		Data:           sdk.V2WorkflowHookData{},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	// Verify the hook exists
	hooks, err := workflow_v2.LoadDistinctSchedulerWorkflowKeysByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Len(t, hooks, 1)

	// Setup hooks service mock
	sHook, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, sHook)
		services.NewClient = services.NewDefaultClient
	}()

	// Expect the DELETE call to hooks service for the scheduler
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), http.MethodDelete, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ ...interface{}) (http.Header, int, error) {
			require.True(t, strings.HasPrefix(path, "/v2/workflow/scheduler/"))
			require.Contains(t, path, "vcs")
			require.Contains(t, path, "my-workflow")
			return nil, 200, nil
		}).Times(1)

	// DELETE the project
	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectV2Handler, vars)
	reqDel, err := http.NewRequest("DELETE", uriDelete, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, reqDel, u, pass)

	wDel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDel, reqDel)
	require.Equal(t, 204, wDel.Code)
}
