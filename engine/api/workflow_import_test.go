package api

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

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
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

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
  pip1_2:
    depends_on:
      - pip1
    pipeline: pip1`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(db, api.Cache, proj.Key, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	assert.NotNil(t, w)

	m, _ := dump.ToStringMap(w)
	assert.Equal(t, "test_1", m["Workflow.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Pipeline.Name"])
	assert.Equal(t, "pip1_2", m["Workflow.Root.Triggers.Triggers0.WorkflowDestNode.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Triggers.Triggers0.WorkflowDestNode.Pipeline.Name"])

}

func Test_getWorkflowPushHandler(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, key, key, u)
	group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
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
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	v1 := sdk.Variable{
		Name:  "var1",
		Value: "value 1",
		Type:  sdk.StringVariable,
	}

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, v1, u))

	v2 := sdk.Variable{
		Name:  "var2",
		Value: "value 2",
		Type:  sdk.SecretVariable,
	}

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, v2, u))

	//Insert ssh and gpg keys
	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: sdk.KeyTypePGP,
		},
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)

	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k))

	k2 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "mykey-ssh",
			Type: sdk.KeyTypeSSH,
		},
		ApplicationID: app.ID,
	}
	kssh, errK := keys.GenerateSSHKey(k2.Name)
	test.NoError(t, errK)

	k2.Public = kssh.Public
	k2.Private = kssh.Private
	k2.KeyID = kssh.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k2))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				ApplicationID: app.ID,
			},
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	proj, _ = project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, key, "test_1", u, workflow.LoadOptions{DeepPipeline: true})
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
