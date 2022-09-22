package api

import (
	"context"
	"encoding/json"
	"github.com/ovh/cds/engine/api/organization"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_postGroupInProjectHandler_UserShouldBeGroupAdminForRWAndRWX(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin on g1 and g2
	u, jwtLambda := assets.InsertLambdaUser(t, db, &g1, g2, g3)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// User cannot add RX permission for g3 on project because not admin of g3
	uri := router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g3,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "User is not a group's admin", sdkError.Message)

	// User can add RX permission for g2 on project because admin of g2
	uri = router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 2)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, proj.ProjectGroups.GetByGroupID(g2.ID).Permission)

	// User can add R permission for g3 on project
	uri = router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 3)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, proj.ProjectGroups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionRead, proj.ProjectGroups.GetByGroupID(g3.ID).Permission)
}

func Test_postGroupInProjectHandler_OnlyReadForDifferentOrganization(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := &proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	orgOne := sdk.Organization{Name: "one"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgOne))

	orgTwo := sdk.Organization{Name: "two"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgTwo))

	// Set organization for groups
	require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, &group.GroupOrganization{
		GroupID:        g1.ID,
		OrganizationID: orgOne.ID,
	}))
	require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, &group.GroupOrganization{
		GroupID:        g2.ID,
		OrganizationID: orgTwo.ID,
	}))

	_, jwt := assets.InsertAdminUser(t, db)

	// Cannot add RX permission for g2 on project because organization is not the same as project's one
	uri := router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "given group with organization \"two\" don't match project organization \"one\"", sdkError.From)

	// Can add R permission for g2 on project
	uri = router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 2)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionRead, proj.ProjectGroups.GetByGroupID(g2.ID).Permission)
}

func Test_putGroupRoleOnProjectHandler_UserShouldBeGroupAdminForRWAndRWX(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin of g1 and g2
	u, jwt := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// Set g2 and g3 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadExecute,
	}))

	// User cannot set RWX permission for g3 on project because not admin of g3
	uri := router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g3.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g3,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "User is not a group's admin", sdkError.Message)

	// User can set RX permission for g2 on project because admin of g2
	uri = router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 3)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, proj.ProjectGroups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionReadExecute, proj.ProjectGroups.GetByGroupID(g3.ID).Permission)

	// User can downgrade permission to R for g3 on workflow
	uri = router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g3.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 3)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, proj.ProjectGroups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionRead, proj.ProjectGroups.GetByGroupID(g3.ID).Permission)
}

func Test_putGroupRoleOnProjectHandler_OnlyReadForDifferentOrganization(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := &proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	orgOne := sdk.Organization{Name: "one"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgOne))

	orgTwo := sdk.Organization{Name: "two"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgTwo))

	// Set organization for groups
	require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, &group.GroupOrganization{
		GroupID:        g1.ID,
		OrganizationID: orgOne.ID,
	}))
	require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, &group.GroupOrganization{
		GroupID:        g2.ID,
		OrganizationID: orgTwo.ID,
	}))

	_, jwt := assets.InsertAdminUser(t, db)

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Cannot set RX permission for g2 on project because organization is not the same as project's one
	uri := router.GetRoute(http.MethodPut, api.putGroupRoleOnProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "given group with organization \"two\" don't match project organization \"one\"", sdkError.From)
}

func Test_deleteGroupFromProjectHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is member of g1
	_, jwt := assets.InsertLambdaUser(t, db, &g1)

	// Add group g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// User can remove permission on group g2
	uri := api.Router.GetRoute(http.MethodDelete, api.deleteGroupFromProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
		"groupName":      g2.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, group.LoadGroupsIntoProject(context.TODO(), db, proj))
	require.Len(t, proj.ProjectGroups, 1)
	require.NotNil(t, proj.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, proj.ProjectGroups.GetByGroupID(g1.ID).Permission)
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
