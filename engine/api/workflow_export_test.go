package api

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	hookModels, err := workflow.LoadHookModels(db)
	require.NoError(t, err)

	var repoWebHookModelID, webHookModelID int64
	for _, hm := range hookModels {
		switch hm.Name {
		case sdk.RepositoryWebHookModelName:
			repoWebHookModelID = hm.ID
		case sdk.WebHookModelName:
			webHookModelID = hm.ID
		}
	}

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	vcs := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")

	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	app := sdk.Application{
		Name:               "app1",
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          vcs.Name,
		FromRepository:     "myrepofrom",
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypeFork,
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
										},
									},
								},
							},
						},
					},
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelID: repoWebHookModelID,
						Config: map[string]sdk.WorkflowNodeHookConfigValue{
							"Method": {
								Type:  "string",
								Value: "POST",
							},
						},
						Conditions: sdk.WorkflowNodeConditions{
							LuaScript: "return git_branch == \"master\"",
						},
					},
					{
						HookModelID: webHookModelID,
						Config: map[string]sdk.WorkflowNodeHookConfigValue{
							"Method": {
								Type:  "string",
								Value: "GET",
							},
						},
						Conditions: sdk.WorkflowNodeConditions{
							LuaScript: "return git_branch == \"toto\"",
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.Background(), db, &w))

	proj, _ = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithGroups,
	)

	svcs, errS := services.LoadAll(context.TODO(), db)
	require.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}
	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method string, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
				actualHooks := in.(map[string]sdk.NodeHook)
				for k, h := range actualHooks {
					h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
						Value:        "http://lolcat.local",
						Configurable: false,
					}
					actualHooks[k] = h
				}
				out = actualHooks
				return nil, 200, nil
			},
		)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/foo/myrepo/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 200, nil)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/foo/myrepo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 200, nil)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
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
version: v2.0
workflow:
  fork:
    depends_on:
    - pip1
  pip1:
    pipeline: pip1
    application: app1
    payload:
      git.author: ""
      git.branch: ""
      git.hash: ""
      git.hash.before: ""
      git.message: ""
      git.repository: foo/myrepo
      git.tag: ""
  pip1_2:
    depends_on:
    - pip1
    pipeline: pip1
  pip1_3:
    depends_on:
    - fork
    pipeline: pip1
hooks:
  pip1:
  - type: RepositoryWebHook
    config:
      eventFilter: ""
    conditions:
      script: return git_branch == "master"
  - type: WebHook
    config:
      method: POST
    conditions:
      script: return git_branch == "toto"
metadata:
  default_tags: git.branch,git.author
notifications:
- type: vcs
  pipelines:
  - pip1
`, rec.Body.String())
}

func Test_getWorkflowExportHandlerWithPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	group2 := &sdk.Group{
		Name: "Test_getWorkflowExportHandlerWithPermissions-Group2",
	}

	g, _ := group.LoadByName(context.TODO(), api.mustDB(), group2.Name)
	if g != nil {
		group2 = g
	} else {
		require.NoError(t, group.Insert(context.TODO(), db, group2))
	}

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   group2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	w := sdk.Workflow{
		Name:            "test_1",
		ProjectID:       proj.ID,
		ProjectKey:      proj.Key,
		RetentionPolicy: "return true",
		HistoryLength:   25,
		Groups: []sdk.GroupPermission{
			{
				Group:      *group2,
				Permission: 7,
			},
		},
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))

	proj, _ = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithGroups,
	)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
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
version: v2.0
workflow:
  pip1:
    pipeline: pip1
  pip1_2:
    depends_on:
    - pip1
    pipeline: pip1
permissions:
  Test_getWorkflowExportHandlerWithPermissions-Group2: 7
retention_policy: return true
history_length: 25
`, rec.Body.String())
}

func Test_getWorkflowPullHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))

	proj, _ = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithGroups,
	)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
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
		btes, err := io.ReadAll(tr)
		test.NoError(t, err, "Unable to read the tar buffer")
		t.Logf("%s", string(btes))
	}
}
