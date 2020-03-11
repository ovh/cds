package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

type mockHTTPClient struct {
	f func(r *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(r *http.Request) (*http.Response, error) {
	return m.f(r)
}

func mock(f func(r *http.Request) (*http.Response, error)) cdsclient.HTTPClient {
	return &mockHTTPClient{f}
}

func writeError(w *http.Response, err error) (*http.Response, error) {
	body := new(bytes.Buffer)
	enc := json.NewEncoder(body)
	w.Body = ioutil.NopCloser(body)
	sdkErr := sdk.ExtractHTTPError(err, "")
	enc.Encode(sdkErr)
	w.StatusCode = sdkErr.Status
	return w, sdkErr
}

func Test_postImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	assert.NoError(t, repositoriesmanager.InsertForProject(db, p, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	a, _ := assets.InsertService(t, db, "Test_postImportAsCodeHandler", services.TypeRepositories)
	b, _ := assets.InsertService(t, db, "Test_VCSService", services.TypeVCS)

	defer func() {
		_ = services.Delete(db, a)
		_ = services.Delete(db, b)
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/github/repos/myrepo/branches":
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
				}
				bs = append(bs, b)
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			default:
				ope := new(sdk.Operation)
				btes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				defer r.Body.Close()
				if err := json.Unmarshal(btes, ope); err != nil {
					return writeError(w, err)
				}
				ope.UUID = sdk.UUID()
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusCreated
			}

			return w, nil
		},
	)

	ope := `{"repo_fullname":"myrepo",  "vcs_server": "github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`

	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": p.Key,
	})
	req, err := http.NewRequest("POST", uri, strings.NewReader(ope))
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)
}

func Test_getImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	a, _ := assets.InsertService(t, db, "Test_getImportAsCodeHandler", services.TypeRepositories)
	defer func() {
		_ = services.Delete(db, a)
	}()

	UUID := sdk.UUID()

	feature.SetClient(nil)

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			ope := new(sdk.Operation)
			ope.URL = "https://github.com/fsamin/go-repo.git"
			ope.UUID = UUID
			ope.Status = sdk.OperationStatusDone
			ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
			ope.LoadFiles.Results = map[string][]byte{
				"w-go-repo.yml": []byte(`name: w-go-repo
					version: v1.0
					pipeline: build
					application: go-repo
					pipeline_hooks:
					- type: RepositoryWebHook
					`),
			}
			if err := enc.Encode(ope); err != nil {
				return writeError(w, err)
			}

			w.StatusCode = http.StatusOK
			return w, nil
		},
	)

	uri := api.Router.GetRoute("GET", api.getImportAsCodeHandler, map[string]string{
		"permProjectKey": p.Key,
		"uuid":           UUID,
	})
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)
}

func Test_postPerformImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	//Insert Project
	pkey := sdk.RandomString(10)
	_ = assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	a, _ := assets.InsertService(t, db, "Test_postPerformImportAsCodeHandler_Repo", services.TypeRepositories)
	b, _ := assets.InsertService(t, db, "Test_postPerformImportAsCodeHandler_VCS", services.TypeHooks)

	defer func() {
		_ = services.Delete(db, a)
		_ = services.Delete(db, b)
	}()

	UUID := sdk.UUID()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("RequestURI: %s", r.URL.Path)
			switch r.URL.Path {
			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				if err := service.UnmarshalBody(r, &hooks); err != nil {
					return nil, sdk.WithStack(err)
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

				body := new(bytes.Buffer)
				w := new(http.Response)
				enc := json.NewEncoder(body)
				w.Body = ioutil.NopCloser(body)
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusOK
				return w, nil
			default:
				body := new(bytes.Buffer)
				w := new(http.Response)
				enc := json.NewEncoder(body)
				w.Body = ioutil.NopCloser(body)

				ope := new(sdk.Operation)
				ope.URL = "https://github.com/fsamin/go-repo.git"
				ope.UUID = UUID
				ope.VCSServer = "github"
				ope.RepoFullName = "fsamin/go-repo"
				ope.RepositoryInfo = &sdk.OperationRepositoryInfo{
					Name:          "go-repo",
					FetchURL:      ope.URL,
					DefaultBranch: "master",
				}
				ope.Status = sdk.OperationStatusDone
				ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
				ope.LoadFiles.Results = map[string][]byte{
					"w-go-repo.yml": []byte(`name: w-go-repo
version: v1.0
pipeline: build
application: go-repo`),
					"go-repo.app.yml": []byte(`name: go-repo
version: v1.0`),
					"go-repo.pip.yml": []byte(`name: build
version: v1.0`),
				}
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusOK
				return w, nil
			}
		},
	)

	uri := api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": pkey,
		"uuid":           UUID,
	})
	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())
}

func Test_postResyncPRAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)
	pkey := sdk.RandomString(10)
	p := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	// Clean as code event
	as, err := ascode.LoadAsCodeEventByRepo(context.TODO(), db, "urltomyrepo")
	assert.NoError(t, err)
	for _, a := range as {
		assert.NoError(t, ascode.DeleteAsCodeEvent(db, a))
	}

	assert.NoError(t, repositoriesmanager.InsertForProject(db, p, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	// Add application
	appS := `version: v1.0
name: blabla
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(context.Background(), db, api.Cache, *p, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	app.FromRepository = "urltomyrepo"
	assert.NoError(t, application.Update(db, api.Cache, app))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  p.ID,
		ProjectKey: p.Key,
		Name:       sdk.RandomString(10),
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	wf := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  p.ID,
		ProjectKey: p.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, p.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &wf))

	// mock service
	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InsertService(t, db, "Test_postResyncPRAsCodeHandler", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/github/repos/sguiheux/demo/pullrequests/666":
				pr := sdk.VCSPullRequest{
					ID:     666,
					URL:    "urltomypr",
					Merged: true,
				}
				if err := enc.Encode(pr); err != nil {
					return writeError(w, err)
				}

			default:
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	// Add some events to resync
	asCodeEvent := sdk.AsCodeEvent{
		Username:       u.GetUsername(),
		CreateDate:     time.Now(),
		FromRepo:       "urltomyrepo",
		Migrate:        true,
		PullRequestID:  666,
		PullRequestURL: "urltomypr",
		Data: sdk.AsCodeEventData{
			Workflows: map[int64]string{
				wf.ID: wf.Name,
			},
			Applications: map[int64]string{
				app.ID: app.Name,
			},
			Pipelines: map[int64]string{
				pip.ID: pip.Name,
			},
		},
	}
	assert.NoError(t, ascode.InsertOrUpdateAsCodeEvent(db, &asCodeEvent))

	uri := api.Router.GetRoute("POST", api.postResyncPRAsCodeHandler, map[string]string{
		"key": pkey,
	})

	uri = fmt.Sprintf("%s?appName=blabla&repo=urltomyrepo", uri)
	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
	t.Logf(w.Body.String())

	// Check there is no more events in db
	assDB, err := ascode.LoadAsCodeEventByRepo(context.TODO(), db, "urltomyrepo")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(assDB))

	// Check workflow has been migrated
	wUpdated, err := workflow.Load(context.TODO(), db, api.Cache, *p, wf.Name, workflow.LoadOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "urltomyrepo", wUpdated.FromRepository)
}
