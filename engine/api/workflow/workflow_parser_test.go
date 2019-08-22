package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func TestParseAndImport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)

	pipelineName := sdk.RandomString(10)

	//Pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       pipelineName,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	//Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(db, cache, proj, app))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	//Reload project
	proj, _ = project.Load(db, cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	input := &exportentities.Workflow{
		Name: sdk.RandomString(10),
		Workflow: map[string]exportentities.NodeEntry{
			"root": {
				PipelineName:    pipelineName,
				ApplicationName: app.Name,
			},
			"first": {
				PipelineName: pipelineName,
				DependsOn:    []string{"root"},
			},
			"second": {
				PipelineName: pipelineName,
				DependsOn:    []string{"first"},
			},
			"fork": {
				DependsOn: []string{"root"},
			},
			"third": {
				PipelineName: pipelineName,
				DependsOn:    []string{"fork"},
			},
		},
	}

	_, _, err = workflow.ParseAndImport(context.TODO(), db, cache, proj, nil, input, localConsumer, workflow.ImportOptions{Force: true})
	assert.NoError(t, err)

	w, errW := workflow.Load(context.TODO(), db, cache, proj, input.Name, workflow.LoadOptions{})
	assert.NoError(t, errW)
	assert.NotNil(t, w)

	b, err := json.Marshal(w)
	t.Logf("Workflow = \n%s", string(b))
	assert.NoError(t, err)

	assert.Equal(t, w.FromRepository, "")
	assert.Len(t, w.WorkflowData.Node.Triggers, 2)
	if w.WorkflowData.Node.Triggers[0].ChildNode.Type == "fork" {
		assert.Equal(t, w.WorkflowData.Node.Triggers[0].ChildNode.Name, "fork")
		assert.Len(t, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers, 1)
		assert.Equal(t, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Name, "third")
	} else {
		assert.Equal(t, w.WorkflowData.Node.Triggers[1].ChildNode.Name, "fork")
		assert.Len(t, w.WorkflowData.Node.Triggers[1].ChildNode.Triggers, 1)
		assert.Equal(t, w.WorkflowData.Node.Triggers[1].ChildNode.Triggers[0].ChildNode.Name, "third")
	}
}

// TestParseAndImportFromRepository tests to import a workflow with FromRepository
func TestParseAndImportFromRepository(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	UUID := sdk.UUID()

	mockServiceVCS, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerVCS", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockServiceVCS) // nolint
	}()

	mockServiceHook, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerHook", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockServiceHook) // nolint
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			w.StatusCode = http.StatusOK
			switch r.URL.String() {
			case "/operations":
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo":
				vcsRepo := sdk.VCSRepo{
					Name:         "foo/myrepo",
					SSHCloneURL:  "git:foo",
					HTTPCloneURL: "https:foo",
				}
				if err := enc.Encode(vcsRepo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":
				hookInfo := repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
				}
				if err := enc.Encode(hookInfo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				hook := sdk.VCSHook{
					ID: "myod",
				}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/branches":
				vcsPR := []sdk.VCSBranch{{
					ID:        "master",
					DisplayID: "master",
					Default:   true,
				}}
				if err := enc.Encode(vcsPR); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				var hooks map[string]sdk.NodeHook
				bts, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(bts, &hooks); err != nil {
					return writeError(w, err)
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	pipelineName := sdk.RandomString(10)

	//Pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       pipelineName,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	//Application
	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(db, cache, proj, app))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	//Reload project
	proj, _ = project.Load(db, cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	input := &exportentities.Workflow{
		Name: sdk.RandomString(10),
		Workflow: map[string]exportentities.NodeEntry{
			"root": {
				PipelineName:    pipelineName,
				ApplicationName: app.Name,
			},
		},
	}

	_, _, err := workflow.ParseAndImport(context.TODO(), db, cache, proj, nil, input, u, workflow.ImportOptions{Force: true, FromRepository: "foo/myrepo"})
	assert.NoError(t, err)

	w, errW := workflow.Load(context.TODO(), db, cache, proj, input.Name, workflow.LoadOptions{})
	assert.NoError(t, errW)
	assert.NotNil(t, w)

	b, err := json.Marshal(w)
	t.Logf("Workflow = \n%s", string(b))
	assert.NoError(t, err)

	assert.Equal(t, w.FromRepository, "foo/myrepo")
}
