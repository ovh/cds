package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_postWorkflowGroupHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))

	t.Logf("%+v\n", proj)

	newGrp := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, newGrp.ID, permission.PermissionRead))
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	reqG := sdk.GroupPermission{
		Permission: 7,
		Group:      *newGrp,
	}

	uri := router.GetRoute("POST", api.postWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, reqG)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var wFromAPI sdk.Workflow
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wFromAPI))

	assert.Equal(t, len(wFromAPI.Groups), 2)
	assert.Equal(t, wFromAPI.Groups[1].Group.Name, reqG.Group.Name)
}

func Test_postWorkflowGroupWithLessThanRWXProjectHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))

	t.Logf("%+v\n", proj)

	newGrp := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, newGrp.ID, permission.PermissionReadWriteExecute))
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	reqG := sdk.GroupPermission{
		Permission: 4,
		Group:      *newGrp,
	}

	uri := router.GetRoute("POST", api.postWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, reqG)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
}

func Test_putWorkflowGroupHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))

	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	_, _, errG := group.AddGroup(api.mustDB(), &gr)
	test.NoError(t, errG)

	tmpGr := assets.InsertTestGroup(t, db, sdk.RandomString(5))
	test.NoError(t, group.InsertGroupInProject(db, proj2.ID, tmpGr.ID, permission.PermissionRead))

	test.NoError(t, group.InsertGroupInProject(db, proj2.ID, gr.ID, permission.PermissionRead))
	test.NoError(t, group.AddWorkflowGroup(db, &w, sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID:   tmpGr.ID,
			Name: tmpGr.Name,
		},
	}))
	test.NoError(t, group.AddWorkflowGroup(db, &w, sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID:   gr.ID,
			Name: gr.Name,
		},
	}))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        gr.Name,
	}
	reqG := sdk.GroupPermission{
		Permission: 4,
		Group:      gr,
	}

	uri := router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, reqG)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var wFromAPI sdk.Workflow
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wFromAPI))

	assert.Equal(t, 3, len(wFromAPI.Groups))
	checked := false
	for _, grp := range wFromAPI.Groups {
		if grp.Group.Name == reqG.Group.Name {
			checked = true
			assert.Equal(t, 4, grp.Permission)
		}
	}
	assert.True(t, checked)
}

func Test_deleteWorkflowGroupHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))

	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	_, _, errG := group.AddGroup(api.mustDB(), &gr)
	test.NoError(t, errG)

	test.NoError(t, group.InsertGroupInProject(db, proj2.ID, gr.ID, 7))
	test.NoError(t, group.AddWorkflowGroup(api.mustDB(), &w, sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID:   gr.ID,
			Name: gr.Name,
		},
	}))
	test.NoError(t, group.DeleteWorkflowGroup(db, &w, proj.ProjectGroups[0].Group.ID, 0))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"groupName":        gr.Name,
	}
	reqG := sdk.GroupPermission{
		Permission: 4,
		Group: sdk.Group{
			ID:   1,
			Name: gr.Name,
		},
	}

	uri := router.GetRoute("DELETE", api.deleteWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, reqG)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)
}

// Test_UpdateProjectPermsWithWorkflow Useful to test permission propagation on project
func Test_UpdateProjectPermsWithWorkflow(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertLambdaUser(api.mustDB(), &proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	newWf := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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

	newGr := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	newGp := sdk.GroupPermission{
		Group:      *newGr,
		Permission: permission.PermissionReadWriteExecute,
	}

	uri = router.GetRoute("POST", api.addGroupInProjectHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)
	wfLoaded, errL := workflow.Load(context.Background(), db, api.Cache, proj2, newWf.Name, u, workflow.LoadOptions{})
	test.NoError(t, errL)

	assert.Equal(t, 2, len(wfLoaded.Groups))
	checked := 0
	for _, grProj := range proj2.ProjectGroups {
		for _, grWf := range wfLoaded.Groups {
			if grProj.Group.Name == grWf.Group.Name {
				checked++
				assert.Equal(t, grProj.Permission, grWf.Permission)
				break
			}
		}
	}
	assert.Equal(t, 2, checked, "Haven't checked all groups")
}

// Test_PermissionOnWorkflowInferiorOfProject Useful to test when permission on wf is superior than permission on project
func Test_PermissionOnWorkflowInferiorOfProject(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertLambdaUser(api.mustDB(), &proj.ProjectGroups[0].Group)

	// Add a new group on project to let us update the previous group permission to READ (because we must have at least one RW permission on project)
	newGr := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, newGr.ID, permission.PermissionReadWriteExecute))
	test.NoError(t, group.InsertUserInGroup(db, newGr.ID, u.ID, true))
	test.NoError(t, group.UpdateGroupRoleInProject(db, proj.ID, proj.ProjectGroups[0].Group.ID, permission.PermissionRead))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	newWf := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	assert.NotEqual(t, 0, newWf.ID)

	// Update workflow group to change READ to RWX and get permission on project in READ and permission on workflow in RWX to test edition and run
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)

	newGp := sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: permission.PermissionReadWriteExecute,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, group.DeleteUserFromGroup(db, proj.ProjectGroups[0].Group.ID, u.ID))

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)
	wfLoaded, errL := workflow.Load(context.Background(), db, api.Cache, proj2, newWf.Name, u, workflow.LoadOptions{DeepPipeline: true})
	test.NoError(t, errL)
	assert.Equal(t, 2, len(wfLoaded.Groups))

	// Try to update workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	wfLoaded.HistoryLength = 300
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wfLoaded)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfLoaded, errL = workflow.Load(context.Background(), db, api.Cache, proj2, newWf.Name, u, workflow.LoadOptions{})
	test.NoError(t, errL)
	assert.Equal(t, 2, len(wfLoaded.Groups))
	assert.Equal(t, int64(300), wfLoaded.HistoryLength)

	// Try to run workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := sdk.WorkflowRunPostHandlerOption{FromNodeIDs: []int64{wfLoaded.WorkflowData.Node.ID}, Manual: &sdk.WorkflowNodeRunManual{User: *u}}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 202, w.Code)

	// Update permission group on workflow to switch RWX to RO
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)

	newGp = sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: permission.PermissionRead,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// try to run the workflow with user in read only
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	// create user in read only
	userRo, passRo := assets.InsertLambdaUser(api.mustDB(), &proj.ProjectGroups[0].Group)
	req = assets.NewAuthentifiedRequest(t, userRo, passRo, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

// Test_PermissionOnWorkflowWithRestrictionOnNode Useful to test when we add permission on a workflow node
func Test_PermissionOnWorkflowWithRestrictionOnNode(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertLambdaUser(api.mustDB(), &proj.ProjectGroups[0].Group)

	// Add a new group on project to let us update the previous group permission to READ (because we must have at least one RW permission on project)
	newGr := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, newGr.ID, permission.PermissionReadWriteExecute))
	test.NoError(t, group.InsertUserInGroup(db, newGr.ID, u.ID, true))
	test.NoError(t, group.UpdateGroupRoleInProject(db, proj.ID, proj.ProjectGroups[0].Group.ID, permission.PermissionRead))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	newWf := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
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
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	assert.NotEqual(t, 0, newWf.ID)

	// Update workflow group to change READ to RWX and get permission on project in READ and permission on workflow in RWX to test edition and run
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": newWf.Name,
		"groupName":        proj.ProjectGroups[0].Group.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)

	newGp := sdk.GroupPermission{
		Group:      proj.ProjectGroups[0].Group,
		Permission: permission.PermissionReadWriteExecute,
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &newGp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, group.DeleteUserFromGroup(db, proj.ProjectGroups[0].Group.ID, u.ID))

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)
	wfLoaded, errL := workflow.Load(context.Background(), db, api.Cache, proj2, newWf.Name, u, workflow.LoadOptions{DeepPipeline: true})
	test.NoError(t, errL)
	assert.Equal(t, 2, len(wfLoaded.Groups))

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
			Permission: permission.PermissionReadExecute,
		},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wfLoaded)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfLoaded, errL = workflow.Load(context.Background(), db, api.Cache, proj2, newWf.Name, u, workflow.LoadOptions{})
	test.NoError(t, errL)
	assert.Equal(t, 2, len(wfLoaded.Groups))
	assert.Equal(t, int64(300), wfLoaded.HistoryLength)

	// Try to run workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wfLoaded.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := sdk.WorkflowRunPostHandlerOption{FromNodeIDs: []int64{wfLoaded.WorkflowData.Node.ID}, Manual: &sdk.WorkflowNodeRunManual{User: *u}}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &opts)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
	var wfError sdk.Error
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfError))
	assert.Equal(t, "you don't have execution right", wfError.Message)
}
