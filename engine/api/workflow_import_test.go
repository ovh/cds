package api

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_postWorkflowImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	sdk.AddParameter(&pip.Parameter, "name", sdk.StringParameter, "value")
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postWorkflowImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `name: test_1
version: v1.0
workflow:
  pip1:
    pipeline: pip1
    parameters:
      name: value
  pip1_2:
    depends_on:
      - pip1
    pipeline: pip1
metadata:
  default_tags: git.branch,git.author,git.hash`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.NotNil(t, w)

	m, _ := dump.ToStringMap(w)
	t.Logf("%+v", m)
	assert.Equal(t, "test_1", m["Workflow.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Context.PipelineName"])
	assert.Equal(t, "name", m["Workflow.WorkflowData.Node.Context.DefaultPipelineParameters.DefaultPipelineParameters0.Name"])
	assert.Equal(t, "value", m["Workflow.WorkflowData.Node.Context.DefaultPipelineParameters.DefaultPipelineParameters0.Value"])
	assert.Equal(t, "pip1_2", m["Workflow.WorkflowData.Node.Triggers.Triggers0.ChildNode.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Triggers.Triggers0.ChildNode.Context.PipelineName"])
	assert.Equal(t, "git.branch,git.author,git.hash", m["Workflow.Metadata.default_tags"])
}

func Test_postWorkflowImportWithPermissionHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	g1 := assets.InsertTestGroup(t, db, "b-"+sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, "c-"+sdk.RandomString(10))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), "a-"+sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g1.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadExecute,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadExecute,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postWorkflowImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `name: test2
version: v2.0
workflow:
  test:
    pipeline: pip1
    permissions:
      ` + g1.Name + `: 7
permissions:
  ` + g1.Name + `: 5
  ` + g2.Name + `: 7`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test2", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.NotNil(t, w)

	m, _ := dump.ToStringMap(w)
	t.Logf("%+v", m)
	assert.Equal(t, "test2", m["Workflow.Name"])
	assert.Equal(t, "test", m["Workflow.WorkflowData.Node.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Context.PipelineName"])
	assert.Equal(t, "7", m["Workflow.WorkflowData.Node.Groups.Groups0.Permission"])
	assert.Equal(t, g1.Name, m["Workflow.WorkflowData.Node.Groups.Groups0.Group.Name"])
}

func Test_postWorkflowImportHandlerWithExistingIcon(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	sdk.AddParameter(&pip.Parameter, "name", sdk.StringParameter, "value")
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postWorkflowImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `name: test_1
version: v1.0
workflow:
  pip1:
    pipeline: pip1
    parameters:
      name: value
  pip1_2:
    depends_on:
      - pip1
    pipeline: pip1
metadata:
  default_tags: git.branch,git.author,git.hash`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.NotNil(t, w)

	m, _ := dump.ToStringMap(w)
	t.Logf("%+v", m)
	assert.Equal(t, "test_1", m["Workflow.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Context.PipelineName"])
	assert.Equal(t, "name", m["Workflow.WorkflowData.Node.Context.DefaultPipelineParameters.DefaultPipelineParameters0.Name"])
	assert.Equal(t, "value", m["Workflow.WorkflowData.Node.Context.DefaultPipelineParameters.DefaultPipelineParameters0.Value"])
	assert.Equal(t, "pip1_2", m["Workflow.WorkflowData.Node.Triggers.Triggers0.ChildNode.Name"])
	assert.Equal(t, "pip1", m["Workflow.WorkflowData.Node.Triggers.Triggers0.ChildNode.Context.PipelineName"])
	assert.Equal(t, "git.branch,git.author,git.hash", m["Workflow.Metadata.default_tags"])

	w.Icon = "data:image/png;base64,example"

	test.NoError(t, workflow.Update(context.TODO(), db, api.Cache, *proj, w, workflow.UpdateOptions{}))

	wfLoaded, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{WithIcon: true})
	test.NoError(t, err)
	test.NotEmpty(t, wfLoaded.Icon, "Workflow icon must be the same as before")
}

func Test_putWorkflowImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	pip := &sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	sdk.AddParameter(&pip.Parameter, "name", sdk.StringParameter, "value")
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	// create the workflow
	uri := api.Router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	var wf = &sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Name: "pip1",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf))
	assert.Equal(t, 201, rec.Code)

	// update the workflow
	uri = api.Router.GetRoute("PUT", api.putWorkflowImportHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "test_1",
	})
	test.NotEmpty(t, uri)
	body := `name: test_renamed
version: v1.0
workflow:
  pip1:
    pipeline: pip1
    parameters:
      name: value
  pip1_2:
    depends_on:
      - pip1
    pipeline: pip1`
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
}

func Test_putWorkflowImportHandlerWithJoinAndCondition(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	pip := &sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	// create the workflow
	uri := api.Router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	var wf = &sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Name: "pip1",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf))
	assert.Equal(t, 201, rec.Code)

	// update the workflow
	uri = api.Router.GetRoute("PUT", api.putWorkflowImportHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "test_1",
	})
	test.NotEmpty(t, uri)
	body := `name: test_1
version: v1.0
workflow:
  build_admin-panel-api:
    depends_on:
    - root
    pipeline: pip1
  build_admin-panel-ui:
    depends_on:
    - root
    pipeline: pip1
  build_cache-manager:
    depends_on:
    - root
    pipeline: pip1
  build_health-checker:
    depends_on:
    - root
    pipeline: pip1
  deploy_admin-panel-api_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_admin-panel-api_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_admin-panel-ui_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_admin-panel-ui_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_cache-manager_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_cache-manager_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_health-checker_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_health-checker_prod:
    depends_on:
    - join_2
    pipeline: pip1
  join:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    conditions:
      script: return cds_status == "Success" and cds_manual == "true" -- and (cds_manual
        == "true" or git_branch == "master" or git_branch:find("^release/") ~= nil)
  join_2:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    conditions:
      script: return cds_status == "Success" and cds_manual == "true"
  root:
    pipeline: pip1
    payload:
      git.branch: master
metadata:
  default_tags: git.branch,git.tag`

	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_putWorkflowImportHandlerWithJoinWithOrWithoutCondition(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	pip := &sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	// create the workflow
	uri := api.Router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	var wf = &sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Name: "pip1",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf))
	assert.Equal(t, 201, rec.Code)

	// update the workflow
	uri = api.Router.GetRoute("PUT", api.putWorkflowImportHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "test_1",
	})
	test.NotEmpty(t, uri)
	body := `name: test_1
version: v1.0
workflow:
  build_admin-panel-api:
    depends_on:
    - root
    pipeline: pip1
  build_admin-panel-ui:
    depends_on:
    - root
    pipeline: pip1
  build_cache-manager:
    depends_on:
    - root
    pipeline: pip1
  build_health-checker:
    depends_on:
    - root
    pipeline: pip1
  deploy_admin-panel-api_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_admin-panel-api_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_admin-panel-ui_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_admin-panel-ui_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_cache-manager_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_cache-manager_prod:
    depends_on:
    - join_2
    pipeline: pip1
  deploy_health-checker_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_health-checker_prod:
    depends_on:
    - join_2
    pipeline: pip1
    conditions:
      script: return cds_status == "Success" and cds_manual == "true" -- and (cds_manual
        == "true" or git_branch == "master" or git_branch:find("^release/") ~= nil)
  join_2:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    conditions:
      script: return cds_status == "Success" and cds_manual == "true"
  root:
    pipeline: pip1
    payload:
      git.branch: master
metadata:
  default_tags: git.branch,git.tag`

	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_putWorkflowImportHandlerWithJoinWithoutCondition(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	pip := &sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	// create the workflow
	uri := api.Router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	var wf = &sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Name: "pip1",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf))
	assert.Equal(t, 201, rec.Code)

	// update the workflow
	uri = api.Router.GetRoute("PUT", api.putWorkflowImportHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "test_1",
	})
	test.NotEmpty(t, uri)
	body := `name: test_1
version: v1.0
workflow:
  build_admin-panel-api:
    depends_on:
    - root
    pipeline: pip1
  build_admin-panel-ui:
    depends_on:
    - root
    pipeline: pip1
  build_cache-manager:
    depends_on:
    - root
    pipeline: pip1
  build_health-checker:
    depends_on:
    - root
    pipeline: pip1
  deploy_admin-panel-api_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_admin-panel-api_prod:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    pipeline: pip1
  deploy_admin-panel-ui_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_admin-panel-ui_prod:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    pipeline: pip1
  deploy_cache-manager_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_cache-manager_prod:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    pipeline: pip1
  deploy_health-checker_dev:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    pipeline: pip1
  deploy_health-checker_prod:
    depends_on:
    - deploy_admin-panel-api_dev
    - deploy_admin-panel-ui_dev
    - deploy_cache-manager_dev
    - deploy_health-checker_dev
    pipeline: pip1
    conditions:
      script: return cds_status == "Success" and cds_manual == "true" -- and (cds_manual
        == "true" or git_branch == "master" or git_branch:find("^release/") ~= nil)
    conditions:
      script: return cds_status == "Success" and cds_manual == "true"
  root:
    pipeline: pip1
    payload:
      git.branch: master
metadata:
  default_tags: git.branch,git.tag`

	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_getWorkflowPushHandler(t *testing.T) {
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
			Name:    "myjob",
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	require.NoError(t, application.Insert(db, proj.ID, app))

	v1 := sdk.ApplicationVariable{
		Name:  "var1",
		Value: "value 1",
		Type:  sdk.StringVariable,
	}
	require.NoError(t, application.InsertVariable(db, app.ID, &v1, u))

	v2 := sdk.ApplicationVariable{
		Name:  "var2",
		Value: "value 2",
		Type:  sdk.SecretVariable,
	}
	require.NoError(t, application.InsertVariable(db, app.ID, &v2, u))

	//Insert ssh and gpg keys
	k := &sdk.ApplicationKey{
		Name:          "app-mykey",
		Type:          sdk.KeyTypePGP,
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	require.NoError(t, err)

	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	require.NoError(t, application.InsertKey(db, k))

	k2 := &sdk.ApplicationKey{
		Name:          "app-mykey-ssh",
		Type:          sdk.KeyTypeSSH,
		ApplicationID: app.ID,
	}
	kssh, errK := keys.GenerateSSHKey(k2.Name)
	require.NoError(t, errK)

	k2.Public = kssh.Public
	k2.Private = kssh.Private
	k2.KeyID = kssh.KeyID
	require.NoError(t, application.InsertKey(db, k2))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID: app.ID,
					PipelineID:    pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child",
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

	proj, _ = project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))
	test.NoError(t, workflow.RenameNode(context.TODO(), api.mustDB(), &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{DeepPipeline: true})
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

	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postWorkflowPushHandler, vars)

	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)
	req.Body = ioutil.NopCloser(r)
	req.Header.Set("Content-Type", "application/tar")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_putWorkflowImportHandlerMustNotHave2Joins(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	pip := &sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	// create the workflow
	uri := api.Router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	var wf = &sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Name: "pip1",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf))
	assert.Equal(t, 201, rec.Code)

	// update the workflow
	uri = api.Router.GetRoute("PUT", api.putWorkflowImportHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "test_1",
	})
	test.NotEmpty(t, uri)
	body := `name: test_1
version: v1.0
workflow:
  build_admin-panel-api:
    depends_on:
    - root
    pipeline: pip1
  build_admin-panel-ui:
    depends_on:
    - root
    pipeline: pip1
  build_cache-manager:
    depends_on:
    - root
    pipeline: pip1
  build_health-checker:
    depends_on:
    - root
    pipeline: pip1
  deploy_admin-panel-api_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_admin-panel-api_prod:
    depends_on:
    - fork
    pipeline: pip1
  deploy_admin-panel-ui_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_admin-panel-ui_prod:
    depends_on:
    - fork
    pipeline: pip1
  deploy_cache-manager_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_cache-manager_prod:
    depends_on:
    - fork
    pipeline: pip1
  deploy_health-checker_dev:
    depends_on:
    - join
    pipeline: pip1
  deploy_health-checker_prod:
    depends_on:
    - fork
    pipeline: pip1
  join:
    depends_on:
    - build_admin-panel-api
    - build_admin-panel-ui
    - build_cache-manager
    - build_health-checker
    conditions:
      script: return cds_status == "Success" and cds_manual == "true" -- and (cds_manual
        == "true" or git_branch == "master" or git_branch:find("^release/") ~= nil)
  fork:
    depends_on:
    - deploy_admin-panel-api_dev
    conditions:
      script: return cds_status == "Success" and cds_manual == "true"
  root:
    pipeline: pip1
    payload:
      git.branch: master
metadata:
  default_tags: git.branch,git.tag
`

	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	p, errP := project.Load(context.TODO(), db, proj.Key)
	assert.NoError(t, errP)
	wUpdated, err := workflow.Load(context.TODO(), db, api.Cache, *p, "test_1", workflow.LoadOptions{})
	assert.NoError(t, err)

	t.Logf("%+v", wUpdated.WorkflowData)
	assert.Equal(t, 1, len(wUpdated.WorkflowData.Joins))
}

func Test_postWorkflowImportHandler_editPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	g1 := assets.InsertTestGroup(t, db, "b-"+sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, "c-"+sdk.RandomString(10))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), "a-"+sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g1.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadExecute,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadExecute,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	uri := api.Router.GetRoute("POST", api.postWorkflowImportHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `name: test_1
version: v2.0
workflow:
  pip1:
    pipeline: pip1`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	// Workflow permissions should be inherited from project
	require.Len(t, w.Groups, 3)
	sort.Slice(w.Groups, func(i, j int) bool {
		return w.Groups[i].Group.Name < w.Groups[j].Group.Name
	})
	assert.Equal(t, proj.ProjectGroups[0].Group.Name, w.Groups[0].Group.Name)
	assert.Equal(t, sdk.PermissionReadWriteExecute, w.Groups[0].Permission)
	assert.Equal(t, g1.Name, w.Groups[1].Group.Name)
	assert.Equal(t, sdk.PermissionReadExecute, w.Groups[1].Permission)
	assert.Equal(t, g2.Name, w.Groups[2].Group.Name)
	assert.Equal(t, sdk.PermissionReadExecute, w.Groups[2].Permission)

	// We want to change to permisison for g2 and remove the permission for g1
	uri = api.Router.GetRoute("POST", api.postWorkflowImportHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	q := req.URL.Query()
	q.Set("force", "true")
	req.URL.RawQuery = q.Encode()

	body = `name: test_1
version: v2.0
workflow:
  pip1:
    pipeline: pip1
permissions:
  ` + proj.ProjectGroups[0].Group.Name + `: 7
  ` + g2.Name + `: 4`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	t.Logf(">>%s", rec.Body.String())

	w, err = workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, w.Groups, 2)
	sort.Slice(w.Groups, func(i, j int) bool {
		return w.Groups[i].Group.Name < w.Groups[j].Group.Name
	})
	assert.Equal(t, proj.ProjectGroups[0].Group.Name, w.Groups[0].Group.Name)
	assert.Equal(t, sdk.PermissionReadWriteExecute, w.Groups[0].Permission)
	assert.Equal(t, g2.Name, w.Groups[1].Group.Name)
	assert.Equal(t, sdk.PermissionRead, w.Groups[1].Permission)

	// Import again the workflow without permissions should reset to project permissions
	uri = api.Router.GetRoute("POST", api.postWorkflowImportHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	q = req.URL.Query()
	q.Set("force", "true")
	req.URL.RawQuery = q.Encode()

	body = `name: test_1
version: v2.0
workflow:
  pip1:
    pipeline: pip1`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	t.Logf(">>%s", rec.Body.String())

	w, err = workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, w.Groups, 3)
	sort.Slice(w.Groups, func(i, j int) bool {
		return w.Groups[i].Group.Name < w.Groups[j].Group.Name
	})
	assert.Equal(t, proj.ProjectGroups[0].Group.Name, w.Groups[0].Group.Name)
	assert.Equal(t, sdk.PermissionReadWriteExecute, w.Groups[0].Permission)
	assert.Equal(t, g1.Name, w.Groups[1].Group.Name)
	assert.Equal(t, sdk.PermissionReadExecute, w.Groups[1].Permission)
	assert.Equal(t, g2.Name, w.Groups[2].Group.Name)
	assert.Equal(t, sdk.PermissionReadExecute, w.Groups[2].Permission)
}
