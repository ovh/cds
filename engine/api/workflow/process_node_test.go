package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

type mockServiceClient struct {
	f func(r *http.Request) (*http.Response, error)
}

// Payload: nothing
func TestHookRunWithoutPayloadProcessNodeBuildParameter(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	webHookModel, err := workflow.LoadHookModelByName(db, sdk.WebHookModelName)
	assert.NoError(t, err)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestHookRunWithoutPayloadProcessNodeBuildParameter", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=master":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createBuildPipeline(t, db, cache, proj, u)
	app := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Config:        sdk.WebHookModel.DefaultConfig,
						HookModelName: sdk.WebHookModelName,
						HookModelID:   webHookModel.ID,
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()

	w.WorkflowData.Node.Context.DefaultPayload = map[string]string{
		"git.branch":     "master",
		"git.repository": "sguiheux/demo",
	}
	w.Root.Context.DefaultPayload = w.WorkflowData.Node.Context.DefaultPayload

	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var hookEvent sdk.WorkflowNodeRunHookEvent
	hookEvent.WorkflowNodeHookUUID = w.WorkflowData.Node.Hooks[0].UUID
	hookEvent.Payload = nil

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &hookEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 1, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "master", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
}

// Payload: commit only
func TestHookRunWithHashOnlyProcessNodeBuildParameter(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	webHookModel, err := workflow.LoadHookModelByName(db, sdk.WebHookModelName)
	assert.NoError(t, err)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestHookRunWithHashOnlyProcessNodeBuildParameter", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//This is a mock for the vcs service
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
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/currentcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "currentcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createBuildPipeline(t, db, cache, proj, u)
	app := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Config:        sdk.WebHookModel.DefaultConfig,
						HookModelName: sdk.WebHookModelName,
						HookModelID:   webHookModel.ID,
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()

	w.WorkflowData.Node.Context.DefaultPayload = map[string]string{
		"git.branch":     "master",
		"git.repository": "sguiheux/demo",
	}
	w.Root.Context.DefaultPayload = w.WorkflowData.Node.Context.DefaultPayload

	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var hookEvent sdk.WorkflowNodeRunHookEvent
	hookEvent.WorkflowNodeHookUUID = w.WorkflowData.Node.Hooks[0].UUID
	hookEvent.Payload = map[string]string{
		"git.hash": "currentcommit",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &hookEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 1, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "", mapParams["git.branch"])
	assert.Equal(t, "currentcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
}

// Payload: branch only
func TestManualRunWithPayloadProcessNodeBuildParameter(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunWithPayloadProcessNodeBuildParameter", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createBuildPipeline(t, db, cache, proj, u)
	app := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 1, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
}

// Payload: branch and commit
func TestManualRunBranchAndCommitInPayloadProcessNodeBuildParameter(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBranchAndCommitInPayloadProcessNodeBuildParameter", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice)
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				t.Fatalf("No need to get branch: %s", r.URL.String())
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/currentcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "currentcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createBuildPipeline(t, db, cache, proj, u)
	app := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
		"git.hash":   "currentcommit",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 1, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "currentcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
}

// Payload: multi application, multi repo
func TestManualRunBuildParameterMultiApplication(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice)
	}()

	//This is a mock for the vcs service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/stash/repos/ovh/cds":
				repo := sdk.VCSRepo{
					URL:          "https",
					Name:         "cds",
					ID:           "123",
					Fullname:     "ovh/cds",
					Slug:         "ovh",
					HTTPCloneURL: "https://stash.com/ovh/cds.git",
					SSHCloneURL:  "git://stash.com/ovh/cds.git",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/stash/repos/ovh/cds/branches":
				bs := []sdk.VCSBranch{
					{
						LatestCommit: "defaultCommit",
						DisplayID:    "defaultBranch",
						Default:      true,
						ID:           "1",
					},
				}
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/stash/repos/ovh/cds/branches/?branch=feat%2Fbranch":
				return writeError(w, sdk.ErrNotFound)
			case "/vcs/github/repos/sguiheux/demo/branches":
				bs := []sdk.VCSBranch{
					{
						LatestCommit: "defaultCommit",
						DisplayID:    "defaultBranch",
						Default:      true,
						ID:           "1",
					},
				}
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/stash/repos/ovh/cds/commits/defaultCommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "john.snow",
						Email: "john.snow@winterfell",
					},
					Hash:      "defaultCommit",
					Message:   "super default commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)
	app2 := createApplication2(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1, *app2)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child1",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    proj.Pipelines[0].ID,
								ApplicationID: proj.Applications[1].ID,
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "child2",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID:    proj.Pipelines[0].ID,
											ApplicationID: proj.Applications[0].ID,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
			proj.Applications[1].ID: proj.Applications[1],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 3, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
	assert.Equal(t, "github", wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].VCSServer)

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "defaultBranch", mapParams2["git.branch"])
	assert.Equal(t, "defaultCommit", mapParams2["git.hash"])
	assert.Equal(t, "john.snow", mapParams2["git.author"])
	assert.Equal(t, "super default commit", mapParams2["git.message"])
	assert.Equal(t, "mylastcommit", mapParams2["workflow.root.git.hash"])
	assert.Equal(t, "stash", wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].VCSServer)

	mapParams3 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams3["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams3["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams3["git.author"])
	assert.Equal(t, "super commit", mapParams3["git.message"])
	assert.Equal(t, "defaultBranch", mapParams3["workflow.child1.git.branch"])
	assert.Equal(t, "github", wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.ID][0].VCSServer)
}

// Payload: branch only
func TestGitParamOnPipelineWithoutApplication(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice)
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)
	app2 := createApplication2(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1, *app2)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child1",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: proj.Pipelines[0].ID,
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
			proj.Applications[1].ID: proj.Applications[1],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	// Load run
	var errRun error
	wr, errRun = workflow.LoadRunByID(db, wr.ID, workflow.LoadRunOptions{})
	assert.NoError(t, errRun)

	assert.Equal(t, 2, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams2["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams2["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams2["git.author"])
	assert.Equal(t, "super commit", mapParams2["git.message"])

}

// Payload: branch only
func TestGitParamOnApplicationWithoutRepo(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice)
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)
	app2 := createApplicationWithoutRepo(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1, *app2)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child1",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    proj.Pipelines[0].ID,
								ApplicationID: proj.Applications[1].ID,
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
			proj.Applications[1].ID: proj.Applications[1],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 2, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	t.Logf("%+v", mapParams2)
	assert.Equal(t, "feat/branch", mapParams2["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams2["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams2["git.author"])
	assert.Equal(t, "super commit", mapParams2["git.message"])

}

// Payload: branch only
func TestGitParamOn2ApplicationSameRepo(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice)
	}()

	repoRoute := 0
	repoBranch := 0
	repoCommit := 0
	//This is a mock for the vcs service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/vcs/github/repos/sguiheux/demo":
				repoRoute++
				if repoRoute == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				repoBranch++
				if repoBranch == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				repoCommit++
				if repoCommit == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)
	app3 := createApplication3WithSameRepoAsA(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1, *app3)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child1",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    proj.Pipelines[0].ID,
								ApplicationID: proj.Applications[1].ID,
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
			proj.Applications[1].ID: proj.Applications[1],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
		"my.value":   "bar",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 2, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
	assert.Equal(t, "bar", mapParams["my.value"])

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	t.Logf("%+v", mapParams2)
	assert.Equal(t, "feat/branch", mapParams2["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams2["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams2["git.author"])
	assert.Equal(t, "super commit", mapParams2["git.message"])
	assert.Equal(t, "bar", mapParams2["my.value"])
	assert.Equal(t, "build", mapParams2["workflow.root.pipeline"])
	assert.Equal(t, "github", wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].VCSServer)

}

// Payload: branch only
func TestGitParamWithJoin(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
	}()

	repoRoute := 0
	repoBranch := 0
	repoCommit := 0
	//This is a mock for the vcs service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/vcs/github/repos/sguiheux/demo":
				repoRoute++
				if repoRoute == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				repoBranch++
				if repoBranch == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				repoCommit++
				if repoCommit == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Ref:  "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
			},
			Joins: []sdk.Node{
				{
					Name: "join",
					Type: sdk.NodeTypeJoin,
					JoinContext: []sdk.NodeJoin{
						{
							ParentName: "root",
						},
					},
					Triggers: []sdk.NodeTrigger{
						{
							ChildNode: sdk.Node{
								Name: "child1",
								Type: sdk.NodeTypePipeline,
								Context: &sdk.NodeContext{
									PipelineID:    proj.Pipelines[0].ID,
									ApplicationID: proj.Applications[0].ID,
								},
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
		"my.value":   "bar",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 3, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
	assert.Equal(t, "bar", mapParams["my.value"])

	mapParamsJoin := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Joins[0].ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParamsJoin["git.branch"])
	assert.Equal(t, "mylastcommit", mapParamsJoin["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParamsJoin["git.author"])
	assert.Equal(t, "super commit", mapParamsJoin["git.message"])
	assert.Equal(t, "bar", mapParamsJoin["my.value"])
	assert.Equal(t, "build", mapParamsJoin["workflow.root.pipeline"])

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Joins[0].Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams2["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams2["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams2["git.author"])
	assert.Equal(t, "super commit", mapParams2["git.message"])
	assert.Equal(t, "bar", mapParams2["my.value"])
	assert.Equal(t, "build", mapParams2["workflow.root.pipeline"])
	assert.Equal(t, "join", mapParams2["workflow.join.node"])
	assert.Equal(t, "feat/branch", wr.WorkflowNodeRuns[w.WorkflowData.Joins[0].Triggers[0].ChildNode.ID][0].VCSBranch)
}

// Payload: branch only
func TestGitParamOn2ApplicationSameRepoWithFork(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "stash",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		services.Delete(db, mockVCSSservice) //nolint
	}()

	repoRoute := 0
	repoBranch := 0
	repoCommit := 0
	//This is a mock for the vcs service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/vcs/github/repos/sguiheux/demo":
				repoRoute++
				if repoRoute == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				repoBranch++
				if repoBranch == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				repoCommit++
				if repoCommit == 2 {
					t.Fatalf("Must not be call twice: %s", r.URL.String())
				}
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	app1 := createApplication1(t, db, cache, proj, u)
	app3 := createApplication3WithSameRepoAsA(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app1, *app3)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Ref:  "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "fork",
							Type: sdk.NodeTypeFork,
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "child1",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID:    proj.Pipelines[0].ID,
											ApplicationID: proj.Applications[1].ID,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
			proj.Applications[1].ID: proj.Applications[1],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
		"my.value":   "bar",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 3, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
	assert.Equal(t, "bar", mapParams["my.value"])

	mapParamsFork := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParamsFork["git.branch"])
	assert.Equal(t, "mylastcommit", mapParamsFork["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParamsFork["git.author"])
	assert.Equal(t, "super commit", mapParamsFork["git.message"])
	assert.Equal(t, "bar", mapParamsFork["my.value"])
	assert.Equal(t, "build", mapParamsFork["workflow.root.pipeline"])

	mapParams2 := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams2["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams2["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams2["git.author"])
	assert.Equal(t, "super commit", mapParams2["git.message"])
	assert.Equal(t, "bar", mapParams2["my.value"])
	assert.Equal(t, "build", mapParams2["workflow.root.pipeline"])
	assert.Equal(t, "fork", mapParams2["workflow.fork.node"])

}

// Payload: branch only  + run condition on git.branch
func TestManualRunWithPayloadAndRunCondition(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockVCSSservice := &sdk.Service{Name: "TestManualRunWithPayloadProcessNodeBuildParameter", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//This is a mock for the vcs service
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
				// NEED GET BRANCH TO GET LASTEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=feat%2Fbranch":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "feat/branch",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
			case "/vcs/github/repos/sguiheux/demo/commits/mylastcommit":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "steven.guiheux",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit",
					Message:   "super commit",
					Timestamp: time.Now().Unix(),
				}
				if err := enc.Encode(c); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	pip := createEmptyPipeline(t, db, cache, proj, u)
	//pip2 := createBuildPipeline(t, db, cache, proj, u)
	app := createApplication1(t, db, cache, proj, u)

	// RELOAD PROJECT WITH DEPENDENCIES
	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	// WORKFLOW TO RUN
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    proj.Pipelines[0].ID,
								ApplicationID: proj.Applications[0].ID,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Variable: "git.branch",
											Operator: "eq",
											Value:    "feat/branch",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}

	(&w).RetroMigrate()
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// CREATE RUN
	var manualEvent sdk.WorkflowNodeRunManual
	manualEvent.Payload = map[string]string{
		"git.branch": "feat/branch",
	}

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &manualEvent,
	}
	wr, err := workflow.CreateRun(db, &w, opts, u)
	assert.NoError(t, err)
	wr.Workflow = w

	_, errR := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, opts, u, nil)
	assert.NoError(t, errR)

	assert.Equal(t, 2, len(wr.WorkflowNodeRuns))
	assert.Equal(t, 1, len(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID]))

	mapParams := sdk.ParametersToMap(wr.WorkflowNodeRuns[w.WorkflowData.Node.Triggers[0].ChildNode.ID][0].BuildParameters)
	assert.Equal(t, "feat/branch", mapParams["git.branch"])
	assert.Equal(t, "mylastcommit", mapParams["git.hash"])
	assert.Equal(t, "steven.guiheux", mapParams["git.author"])
	assert.Equal(t, "super commit", mapParams["git.message"])
}

func createEmptyPipeline(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Pipeline {
	pip := &sdk.Pipeline{
		Name: "build",
		Stages: []sdk.Stage{
			{
				Name:       "stage1",
				BuildOrder: 1,
				Enabled:    true,
			},
		},
	}
	assert.NoError(t, pipeline.Import(db, cache, proj, pip, nil, u))
	var errPip error
	pip, errPip = pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, errPip)
	return pip
}

func createBuildPipeline(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Pipeline {
	pip := &sdk.Pipeline{
		Name: "build",
		Stages: []sdk.Stage{
			{
				Name:       "stage1",
				BuildOrder: 1,
				Enabled:    true,
				Jobs: []sdk.Job{
					{
						Enabled: true,
						Action: sdk.Action{
							Name:    "JOb1",
							Enabled: true,
							Actions: []sdk.Action{
								{
									Name:    "gitClone",
									Type:    sdk.BuiltinAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "branch",
											Value: "{{.git.branch}}",
										},
										{
											Name:  "commit",
											Value: "{{.git.hash}}",
										},
										{
											Name:  "directory",
											Value: "{{.cds.workspace}}",
										},
										{
											Name:  "password",
											Value: "",
										},
										{
											Name:  "privateKey",
											Value: "",
										},
										{
											Name:  "url",
											Value: "{{.git.url}}",
										},
										{
											Name:  "user",
											Value: "",
										},
										{
											Name:  "depth",
											Value: "12",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, pipeline.Import(db, cache, proj, pip, nil, u))
	var errPip error
	pip, errPip = pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, errPip)
	return pip
}

func createApplication1(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Application {
	// Add application
	appS := `version: v1.0
name: blabla
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(db, cache, proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)
	return app
}

func createApplication2(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Application {
	// Add application
	appS := `version: v1.0
name: bloublou
vcs_server: stash
repo: ovh/cds
vcs_ssh_key: proj-bloublou
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(db, cache, proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)
	return app
}

func createApplication3WithSameRepoAsA(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Application {
	// Add application
	appS := `version: v1.0
name: blabla2
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(db, cache, proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)
	return app
}

func createApplicationWithoutRepo(t *testing.T, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, u *sdk.User) *sdk.Application {
	// Add application
	appS := `version: v1.0
name: app-no-repo
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(db, cache, proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)
	return app
}

func mock(f func(r *http.Request) (*http.Response, error)) sdk.HTTPClient {
	return &mockServiceClient{f}
}

func (m *mockServiceClient) Do(r *http.Request) (*http.Response, error) {
	return m.f(r)
}

func writeError(w *http.Response, err error) (*http.Response, error) {
	body := new(bytes.Buffer)
	enc := json.NewEncoder(body)
	w.Body = ioutil.NopCloser(body)
	sdkErr := sdk.ExtractHTTPError(err, "")
	_ = enc.Encode(sdkErr) // nolint
	w.StatusCode = sdkErr.Status
	return w, sdkErr
}
