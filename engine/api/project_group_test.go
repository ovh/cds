package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_putGroupRoleOnProjectHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user 1 that is admin of g1 and g2
	u1, jwtLambda1 := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u1.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u1.ID)
	// Create a lambda user 2 that is member of
	_, jwtLambda2 := assets.InsertLambdaUser(t, db, &g1)

	// User 1 can add g2 on project because admin of it
	uri := api.Router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtLambda1, http.MethodPost, uri, sdk.GroupPermission{Group: *g2, Permission: sdk.PermissionReadExecute})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// User 2 cannot add more permission on group g2 because not admin of it
	uri = api.Router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda2, http.MethodPut, uri, sdk.GroupPermission{Group: *g2, Permission: sdk.PermissionReadWriteExecute})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// User 2 can downgrade permission on group g2
	uri = api.Router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda2, http.MethodPut, uri, sdk.GroupPermission{Group: *g2, Permission: sdk.PermissionRead})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// User 2 can remove permission on group g2
	uri = api.Router.GetRoute(http.MethodDelete, api.deleteGroupFromProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda2, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func Test_postGroupInProjectHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g4 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin on g1 and g2
	u, jwtLambda := assets.InsertLambdaUser(t, db, &g1, g2, g3)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// User is admin of g2
	uri := api.Router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{Group: *g2, Permission: sdk.PermissionRead})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// User is not admin of g3
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{Group: *g3, Permission: sdk.PermissionRead})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// User is not member of g4
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{Group: *g4, Permission: sdk.PermissionRead})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

// Test_ProjectPerms Useful to test permission on project
func Test_ProjectPerms(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	newWf := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},

		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	assert.NotEqual(t, 0, newWf.ID)
	newEnv := sdk.Environment{
		Name: "env-" + sdk.RandomString(5),
	}
	uri = router.GetRoute("POST", api.addEnvironmentHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newEnv)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	newApp := sdk.Application{
		Name: "app-" + sdk.RandomString(5),
	}
	uri = router.GetRoute("POST", api.addApplicationHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newApp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	newPip := sdk.Pipeline{
		Name: "pip-" + sdk.RandomString(5),
	}
	uri = router.GetRoute("POST", api.addPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newPip)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
