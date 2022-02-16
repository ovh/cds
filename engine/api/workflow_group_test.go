package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_postWorkflowGroupHandler_RequireGroupOnProject(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, jwt := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	// User can add permission for g2 on workflow because it is on project
	uri := router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g2,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 2)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)

	// User cannot add permission for g3 on workflow because it is not on project
	uri = router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g3,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "Cannot add this permission group on your workflow because this group is not already in the project's permissions", sdkError.Message)
}

func Test_postWorkflowGroupHandler_ShouldBeRWXIfGroupIsRWXOnProject(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, jwt := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	// User cannot add permission<RWX for g2 on workflow
	uri := router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "Cannot add this permission group on your workflow because you can't have less rights than rights in your project when you are in RWX", sdkError.Message)

	// User can add RWX permission for g2 on workflow
	uri = router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 2)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)
}

func Test_postWorkflowGroupHandler_UserShouldBeGroupAdminForRWAndRWX(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin of g1 and g2
	u, jwt := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 and g3 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// User cannot add RX permission for g3 on workflow because not admin of g3
	uri := router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g3,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "User is not a group's admin", sdkError.Message)

	// User can add RX permission for g2 on workflow because admin of g2
	uri = router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 2)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)

	// User can add R permission for g3 on workflow
	uri = router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 3)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionRead, workflowResult.Groups.GetByGroupID(g3.ID).Permission)
}

func Test_postWorkflowGroupHandler_OnlyReadForDifferentOrganization(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := &proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Set organization for groups
	require.NoError(t, group.InsertOrganization(context.TODO(), db, &group.Organization{
		GroupID:      g1.ID,
		Organization: "one",
	}))
	require.NoError(t, group.InsertOrganization(context.TODO(), db, &group.Organization{
		GroupID:      g2.ID,
		Organization: "two",
	}))

	_, jwt := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Cannot add RX permission for g2 on workflow because organization is not the same as project's one
	uri := router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
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

	// Can add R permission for g2 on workflow
	uri = router.GetRoute(http.MethodPost, api.postWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 2)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionRead, workflowResult.Groups.GetByGroupID(g2.ID).Permission)
}

func Test_putWorkflowGroupHandler_ShouldBeRWXIfGroupIsRWXOnProject(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, jwt := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set new group on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	// Set new group on workflow
	require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, w, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g,
	}))

	// User cannot set permission<RWX for new group on workflow
	uri := router.GetRoute(http.MethodPut, api.putWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g,
	})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "Cannot add this permission group on your workflow because you can't have less rights than rights in your project when you are in RWX", sdkError.Message)
}

func Test_putWorkflowGroupHandler_UserShouldBeGroupAdminForRWAndRWX(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin of g1 and g2
	u, jwt := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 and g3 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Set g2 and g3 on workflow
	require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, w, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	}))
	require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, w, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g3,
	}))

	// User cannot set RWX permission for g3 on workflow because not admin of g3
	uri := router.GetRoute(http.MethodPut, api.putWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g3.Name,
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

	// User can set RX permission for g2 on workflow because admin of g2
	uri = router.GetRoute(http.MethodPut, api.putWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 3)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionReadExecute, workflowResult.Groups.GetByGroupID(g3.ID).Permission)

	// User can downgrade permission to R for g3 on workflow
	uri = router.GetRoute(http.MethodPut, api.putWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g3.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPut, uri, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 3)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g3.ID))
	require.Equal(t, sdk.PermissionRead, workflowResult.Groups.GetByGroupID(g3.ID).Permission)
}

func Test_putWorkflowGroupHandler_OnlyReadForDifferentOrganization(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := &proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Set organization for groups
	require.NoError(t, group.InsertOrganization(context.TODO(), db, &group.Organization{
		GroupID:      g1.ID,
		Organization: "one",
	}))
	require.NoError(t, group.InsertOrganization(context.TODO(), db, &group.Organization{
		GroupID:      g2.ID,
		Organization: "two",
	}))

	_, jwt := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Set g2 on workflow
	require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, w, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	}))

	// Cannot set RX permission for g2 on workflow because organization is not the same as project's one
	uri := router.GetRoute(http.MethodPut, api.putWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g2.Name,
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

func Test_deleteWorkflowGroupHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := &proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, pass := assets.InsertAdminUser(t, db)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	// Set g2 on workflow
	require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, w, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g2,
	}))

	// Can remove a workflow group
	uri := router.GetRoute(http.MethodDelete, api.deleteWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g1.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var workflowResult sdk.Workflow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workflowResult))
	require.Len(t, workflowResult.Groups, 1)
	require.NotNil(t, workflowResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, workflowResult.Groups.GetByGroupID(g2.ID).Permission)

	// Cannot remove last workflow group
	uri = router.GetRoute(http.MethodDelete, api.deleteWorkflowGroupHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        g2.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var sdkError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sdkError))
	require.Equal(t, "The last group must have the write permission", sdkError.Message)
}

// Test_UpdateProjectPermsWithWorkflow Useful to test permission propagation on project
func Test_UpdateProjectPermsWithWorkflow(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// Create a lambda user that is admin of g1 and g2
	u, jwt := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	// Create a pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	// Post a new workflow
	wf := sdk.Workflow{
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
	uri := router.GetRoute(http.MethodPost, api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, &wf)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))
	require.NotEqual(t, 0, wf.ID)

	// Add group g2 on the project
	uri = router.GetRoute(http.MethodPost, api.postGroupInProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, http.MethodPost, uri, sdk.GroupPermission{
		Group:      *g2,
		Permission: sdk.PermissionReadWriteExecute,
	})
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// Check project and workflow permissions
	projResult, err := project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithGroups,
	)
	require.NoError(t, err)
	require.Equal(t, 2, len(projResult.ProjectGroups))
	require.NotNil(t, projResult.ProjectGroups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, projResult.ProjectGroups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, projResult.ProjectGroups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, projResult.ProjectGroups.GetByGroupID(g2.ID).Permission)
	wfResult, err := workflow.Load(context.TODO(), db, api.Cache, *proj, wf.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, len(wfResult.Groups))
	require.NotNil(t, wfResult.Groups.GetByGroupID(g1.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, wfResult.Groups.GetByGroupID(g1.ID).Permission)
	require.NotNil(t, wfResult.Groups.GetByGroupID(g2.ID))
	require.Equal(t, sdk.PermissionReadWriteExecute, wfResult.Groups.GetByGroupID(g2.ID).Permission)
}

// Test_PermissionOnWorkflowInferiorOfProject Useful to test when permission on wf is superior than permission on project
func Test_PermissionOnWorkflowInferiorOfProject(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)
	assets.SetUserGroupAdmin(t, db, proj.ProjectGroups[0].Group.ID, u.ID)

	// Add a new group on project to let us update the previous group permission to READ (because we must have at least one RW permission on project)
	newGr := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   newGr.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            newGr.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	oldLink, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(context.TODO(), db, proj.ProjectGroups[0].Group.ID, proj.ID)
	require.NoError(t, err)
	newLink := *oldLink
	newLink.Role = sdk.PermissionRead
	require.NoError(t, group.UpdateLinkGroupProject(db, &newLink))

	// First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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

	//Prepare request to create workflow
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	require.NotEqual(t, 0, newWf.ID)

	// Update workflow group to change READ to RWX and get permission on project in READ and permission on workflow in RWX to test edition and run
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	require.NotEmpty(t, uri)

	newGp := sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: sdk.PermissionReadWriteExecute,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	require.NoError(t, group.DeleteLinkGroupUserForGroupIDAndUserID(db, proj.ProjectGroups[0].Group.ID, u.ID))

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	require.NoError(t, errP)
	wfLoaded, errL := workflow.Load(context.Background(), db, api.Cache, *proj2, newWf.Name, workflow.LoadOptions{DeepPipeline: true})
	require.NoError(t, errL)
	require.Equal(t, 2, len(wfLoaded.Groups))

	// Try to update workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	require.NotEmpty(t, uri)

	wfLoaded.HistoryLength = 300
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wfLoaded)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wfLoaded, errL = workflow.Load(context.Background(), db, api.Cache, *proj2, newWf.Name, workflow.LoadOptions{})
	require.NoError(t, errL)
	require.Equal(t, 2, len(wfLoaded.Groups))
	require.Equal(t, int64(300), wfLoaded.HistoryLength)

	// Try to run workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	require.NotEmpty(t, uri)

	opts := sdk.WorkflowRunPostHandlerOption{
		FromNodeIDs: []int64{wfLoaded.WorkflowData.Node.ID},
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Fullname: u.Fullname,
			Email:    u.GetEmail(),
		},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 202, w.Code)

	// Update permission group on workflow to switch RWX to RO
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	require.NotEmpty(t, uri)

	newGp = sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: sdk.PermissionRead,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// try to run the workflow with user in read only
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	require.NotEmpty(t, uri)

	// create user in read only
	userRo, passRo := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)
	req = assets.NewAuthentifiedRequest(t, userRo, passRo, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

// Test_PermissionOnWorkflowWithRestrictionOnNode Useful to test when we add permission on a workflow node
func Test_PermissionOnWorkflowWithRestrictionOnNode(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)
	assets.SetUserGroupAdmin(t, db, proj.ProjectGroups[0].Group.ID, u.ID)

	// Add a new group on project to let us update the previous group permission to READ (because we must have at least one RW permission on project)
	newGr := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   newGr.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            newGr.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	oldLink, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(context.TODO(), db, proj.ProjectGroups[0].Group.ID, proj.ID)
	require.NoError(t, err)
	newLink := *oldLink
	newLink.Role = sdk.PermissionRead
	require.NoError(t, group.UpdateLinkGroupProject(db, &newLink))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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

	//Prepare request to create workflow
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	require.NotEqual(t, 0, newWf.ID)

	// Update workflow group to change READ to RWX and get permission on project in READ and permission on workflow in RWX to test edition and run
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	require.NotEmpty(t, uri)

	newGp := sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: sdk.PermissionReadWriteExecute,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	require.NoError(t, group.DeleteLinkGroupUserForGroupIDAndUserID(db, proj.ProjectGroups[0].Group.ID, u.ID))

	proj2, err := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	require.NoError(t, err)
	wfLoaded, err := workflow.Load(context.Background(), db, api.Cache, *proj2, newWf.Name, workflow.LoadOptions{DeepPipeline: true})
	require.NoError(t, err)
	require.Equal(t, 2, len(wfLoaded.Groups))

	// Try to update workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	wfLoaded.HistoryLength = 300
	wfLoaded.WorkflowData.Node.Groups = []sdk.GroupPermission{
		{
			Group:      proj.ProjectGroups[0].Group,
			Permission: sdk.PermissionReadExecute,
		},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wfLoaded)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wfLoaded, err = workflow.Load(context.Background(), db, api.Cache, *proj2, newWf.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, len(wfLoaded.Groups))
	require.Equal(t, int64(300), wfLoaded.HistoryLength)

	// Try to run workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	require.NotEmpty(t, uri)

	opts := sdk.WorkflowRunPostHandlerOption{
		FromNodeIDs: []int64{wfLoaded.WorkflowData.Node.ID},
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Fullname: u.Fullname,
			Email:    u.GetEmail(),
		},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
	var wfError sdk.Error
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfError))
	require.Equal(t, "you don't have execution right", wfError.Message)
}
