package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestAPI_detachRepositoriesManagerHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	srvs, err := services.LoadAll(context.TODO(), db)
	require.NoError(t, err)

	for _, srv := range srvs {
		require.NoError(t, services.Delete(db, &srv))
	}

	mockVCSSservice, _ := assets.InsertService(t, db, "TestAPI_detachRepositoriesManagerVCS", sdk.TypeVCS)
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
	}()

	mockServiceHook, _ := assets.InsertService(t, db, "TestAPI_detachRepositoriesManagerHook", sdk.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockServiceHook) // nolint
	}()

	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/vcs/github/repos/sguiheux/demo":
				repo := sdk.VCSRepo{
					URL:          "https",
					Name:         "demo",
					ID:           "123",
					Fullname:     "sguiheux/demo",
					Slug:         "sguiheux",
					HTTPCloneURL: "https://github.com/sguiheux/demo.git",
					SSHCloneURL:  "git://github.com/sguiheux/demo.git",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(w, err)
				}
				// Default payload on workflow insert
			case "/vcs/github/repos/sguiheux/demo/branches":
				b := sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode([]sdk.VCSBranch{b}); err != nil {
					return writeError(w, err)
				}
				// NEED GET BRANCH TO GET LATEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=master":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
			// 	// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "test",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
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
				for k, h := range hooks {
					if h.HookModelName == sdk.RepositoryWebHookModelName {
						cfg := hooks[k].Config
						cfg["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
					}
				}
				if err := enc.Encode(hooks); err != nil {
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
			case "/vcs/github/repos/sguiheux/demo/hooks":
				pr := sdk.VCSHook{
					ID: "666",
				}
				if err := enc.Encode(pr); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/sguiheux/demo/hooks?url=http%3A%2F%2Flolcat.host&id=666":
				// do nothing
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	// Add application
	appS := `version: v1.0
name: blabla
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, _, globalError := application.ParseAndImport(context.Background(), db, api.Cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelName: sdk.RepositoryWebHookModelName,
						UUID:          sdk.RandomString(10),
						Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
						HookModelID:   repoModel.ID,
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	t.Log("Inserting workflow=====")

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	t.Log("Inserting workflow run=====")

	// creates a run
	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	t.Log("Starting workflow run=====")
	_, errWr := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
				"git.author": "test",
			},
		},
	}, *consumer, nil)
	test.NoError(t, errWr)

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}

	t.Log("Trying to detach=====")

	uri := router.GetRoute("POST", api.detachRepositoriesManagerHandler, vars)

	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	rw := httptest.NewRecorder()
	router.Mux.ServeHTTP(rw, req)
	// as there is one repository webhook attached, 403 is expected
	assert.Equal(t, 403, rw.Code)

	t.Log("Loading the workflow=====")

	w2, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	// Delete repository webhook
	var index = 0
	for i, h := range w.WorkflowData.Node.Hooks {
		if h.HookModelID == repoModel.ID {
			index = i
		}
	}
	w2.WorkflowData.Node.Hooks = append(w2.WorkflowData.Node.Hooks[:index], w2.WorkflowData.Node.Hooks[index+1:]...)

	// save the workflow with the repositorywebhook deleted
	t.Log("Updating the workflo without the repositorywebhook=====")
	test.NoError(t, workflow.Update(context.TODO(), db, api.Cache, *proj, w2, workflow.UpdateOptions{}))

	req, err = http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	rw = httptest.NewRecorder()
	router.Mux.ServeHTTP(rw, req)
	// as there is one repository webhook is now removed, 200 is expected
	assert.Equal(t, 200, rw.Code)
}
