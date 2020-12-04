package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func TestUpdateAsCodePipelineHandler(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	event.OverridePubSubKey("events_pubsub_test")
	require.NoError(t, event.Initialize(context.Background(), api.mustDB(), api.Cache))
	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, jwt := assets.InsertAdminUser(t, db)

	client := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		User:                  u.Username,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})

	UUID := sdk.UUID()

	svcs, errS := services.LoadAll(context.TODO(), db)
	assert.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}

	a, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", sdk.TypeVCS)
	b, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", sdk.TypeRepositories)
	c, _ := assets.InsertService(t, db, "TestUpdateAsCodePipelineHandler", sdk.TypeHooks)
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
			case "/vcs/github/repos/foo/myrepo/pullrequests?state=open":
				vcsPRs := []sdk.VCSPullRequest{}
				if err := enc.Encode(vcsPRs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				vcsPR := sdk.VCSPullRequest{
					URL: "myURL",
				}
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
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	require.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	// Create Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

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
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	require.NoError(t, err)

	wk := initWorkflow(t, db, proj, &app, &pip, repoModel)
	wk.FromRepository = "myrepofrom"
	require.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, wk))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)

	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:         sdk.WebsocketFilterTypeAscodeEvent,
		ProjectKey:   proj.Key,
		WorkflowName: wk.Name,
	}}

	uri := api.Router.GetRoute("PUT", api.updateAsCodePipelineHandler, map[string]string{
		"permProjectKey": proj.Key,
		"pipelineKey":    pip.Name,
	})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "PUT", uri, pip)
	q := req.URL.Query()
	q.Set("branch", "master")
	q.Set("message", "my message")
	req.URL.RawQuery = q.Encode()

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-timeout.C:
		t.Fatal("test timeout")
	case err := <-chanErrorReceived:
		t.Fatal(err)
	case evt := <-chanMessageReceived:
		require.Equal(t, fmt.Sprintf("%T", sdk.EventAsCodeEvent{}), evt.Event.EventType)
		var ae sdk.EventAsCodeEvent
		require.NoError(t, json.Unmarshal(evt.Event.Payload, &ae))
		require.Equal(t, "myURL", ae.Event.PullRequestURL)
	}
}
