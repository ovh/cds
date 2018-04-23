package api

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowExportHandler(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(context.Background()), api.Cache, key, key, u)
	group.InsertUserInGroup(api.mustDB(context.Background()), proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(context.Background()), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(context.Background()), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	proj, _ = project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj, u))
	w1, err := workflow.Load(api.mustDB(context.Background()), api.Cache, key, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	assert.Equal(t, `name: test_1
version: v1.0
workflow:
  pip1:
    pipeline: pip1
  pip1_2:
    depends_on:
    - pip1
    pipeline: pip1
`, rec.Body.String())

}

func Test_getWorkflowExportHandlerWithPermissions(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(context.Background()), api.Cache, key, key, u)
	group.InsertUserInGroup(api.mustDB(context.Background()), proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	group2 := &sdk.Group{
		Name: "Test_getWorkflowExportHandlerWithPermissions-Group2",
	}
	group.InsertGroup(api.mustDB(context.Background()), group2)
	group2, _ = group.LoadGroup(api.mustDB(context.Background()), "Test_getWorkflowExportHandlerWithPermissions-Group2")

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(context.Background()), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(context.Background()), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	proj, _ = project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj, u))

	workflow.AddGroup(api.mustDB(context.Background()), &w, sdk.GroupPermission{
		Group:      *group2,
		Permission: 7,
	})

	w1, err := workflow.Load(api.mustDB(context.Background()), api.Cache, key, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?withPermissions=true", nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	assert.Equal(t, `name: test_1
version: v1.0
workflow:
  pip1:
    pipeline: pip1
  pip1_2:
    depends_on:
    - pip1
    pipeline: pip1
permissions:
  Test_getWorkflowExportHandlerWithPermissions-Group2: 7
`, rec.Body.String())

}

func Test_getWorkflowPullHandler(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(context.Background()), api.Cache, key, key, u)
	group.InsertUserInGroup(api.mustDB(context.Background()), proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(context.Background()), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(context.Background()), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	proj, _ = project.Load(api.mustDB(context.Background()), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(api.mustDB(context.Background()), api.Cache, &w, proj, u))
	w1, err := workflow.Load(api.mustDB(context.Background()), api.Cache, key, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowPullHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Header().Get("Content-Type"))

	// Open the tar archive for reading.
	r := bytes.NewReader(rec.Body.Bytes())
	tr := tar.NewReader(r)

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		test.NoError(t, err, "Unable to iterate over the tar buffer")
		t.Logf("Contents of %s:", hdr.Name)
		btes, err := ioutil.ReadAll(tr)
		test.NoError(t, err, "Unable to read the tar buffer")
		t.Logf("%s", string(btes))
	}
}
