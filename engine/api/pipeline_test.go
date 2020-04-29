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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestUpdateAsCodePipelineHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)

	UUID := sdk.UUID()

	svcs, errS := services.LoadAll(context.TODO(), db)
	assert.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}

	a, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", services.TypeVCS)
	b, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", services.TypeRepositories)
	c, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, a) // nolint
		_ = services.Delete(db, b) // nolint
		_ = services.Delete(db, c) // nolint
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
				ope.Status = sdk.OperationStatusProcessing
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/operations/" + UUID:
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
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
			case "/vcs/github/repos/foo/myrepo":
				vcsRepo := sdk.VCSRepo{
					Name:         "foo/myrepo",
					SSHCloneURL:  "git:foo",
					HTTPCloneURL: "https:foo",
				}
				if err := enc.Encode(vcsRepo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				hook := sdk.VCSHook{
					ID: "myod",
				}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				if r.Method == http.MethodGet {
					vcsPRs := []sdk.VCSPullRequest{}
					if err := enc.Encode(vcsPRs); err != nil {
						return writeError(w, err)
					}
				} else {
					vcsPR := sdk.VCSPullRequest{
						URL: "myURL",
					}
					if err := enc.Encode(vcsPR); err != nil {
						return writeError(w, err)
					}
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
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	// Create Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	wkf := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	pip := sdk.Pipeline{
		Name:           sdk.RandomString(10),
		ProjectID:      proj.ID,
		FromRepository: "myrepofrom",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	pip.Stages = []sdk.Stage{
		{
			Name:       "mystage",
			BuildOrder: 1,
			Enabled:    true,
		},
	}

	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
		FromRepository:     "myrepofrom",
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	wk := initWorkflow(t, db, proj, &app, &pip, repoModel)
	wk.FromRepository = "myrepofrom"
	require.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, wk))

	uri := api.Router.GetRoute("PUT", api.updateAsCodePipelineHandler, map[string]string{
		"permProjectKey": proj.Key,
		"pipelineKey":    pip.Name,
	})
	req := assets.NewJWTAuthentifiedRequest(t, pass, "PUT", uri, pip)

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	cpt := 0
	for {
		if cpt >= 10 {
			t.Fail()
			return
		}

		// Get operation
		uriGET := api.Router.GetRoute("GET", api.getWorkflowAsCodeHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"uuid":             myOpe.UUID,
		})
		reqGET, err := http.NewRequest("GET", uriGET, nil)
		test.NoError(t, err)
		assets.AuthentifyRequest(t, reqGET, u, pass)
		wrGet := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(wrGet, reqGET)
		assert.Equal(t, 200, wrGet.Code)
		myOpeGet := new(sdk.Operation)
		err = json.Unmarshal(wrGet.Body.Bytes(), myOpeGet)
		assert.NoError(t, err)

		if myOpeGet.Status < sdk.OperationStatusDone {
			cpt++
			time.Sleep(1 * time.Second)
			continue
		}
		test.NoError(t, json.Unmarshal(wrGet.Body.Bytes(), myOpeGet))
		assert.Equal(t, "myURL", myOpeGet.Setup.Push.PRLink)
		break
	}
}
