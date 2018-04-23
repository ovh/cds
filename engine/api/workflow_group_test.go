package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
)

func Test_postWorkflowGroupHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}

	proj2, errP := project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj2, u))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	reqG := sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID: 1,
		},
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

	assert.Equal(t, len(wFromAPI.Groups), 1)
	assert.Equal(t, wFromAPI.Groups[0].Group.Name, reqG.Group.Name)
}

func Test_putWorkflowGroupHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}

	proj2, errP := project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj2, u))

	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	_, _, errG := group.AddGroup(api.mustDB(context.Background()), &gr)
	test.NoError(t, errG)

	workflow.AddGroup(api.mustDB(context.Background()), &w, sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID:   gr.ID,
			Name: gr.Name,
		},
	})

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

	uri := router.GetRoute("PUT", api.putWorkflowGroupHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, reqG)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var wFromAPI sdk.Workflow
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wFromAPI))

	assert.Equal(t, len(wFromAPI.Groups), 1)
	assert.Equal(t, wFromAPI.Groups[0].Permission, 4)
}

func Test_deleteWorkflowGroupHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name: sdk.RandomString(10),
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}

	proj2, errP := project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj2, u))

	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	_, _, errG := group.AddGroup(api.mustDB(context.Background()), &gr)
	test.NoError(t, errG)

	workflow.AddGroup(api.mustDB(context.Background()), &w, sdk.GroupPermission{
		Permission: 7,
		Group: sdk.Group{
			ID:   gr.ID,
			Name: gr.Name,
		},
	})

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

	var wFromAPI sdk.Workflow
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wFromAPI))

	assert.Equal(t, len(wFromAPI.Groups), 0)
}
