package internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func TestStartWorkerWithABookedJob(t *testing.T) {
	defer gock.Off()

	gock.New("http://lolcat.host").Get("/action/requirement").
		Reply(200).
		JSON([]sdk.Requirement{
			{
				Name:  "bash",
				Type:  sdk.BinaryRequirement,
				Value: "bash",
			},
		})

	modelID := int64(1)
	gock.New("http://lolcat.host").Post("/auth/consumer/worker/signin").
		HeaderPresent("Authorization").
		Reply(201).
		JSON(sdk.Worker{
			ID:      "xxxx-xxxx-xxxxx",
			ModelID: &modelID,
		}).AddHeader("X-CDS-JWT", "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJJRCI6IjJiNTA3ZDZiLTlhZWYtNGNlNS04MzhlLTA1OTU5NjhjMGU5NSIsIkdyb3VwSURzIjpbMV0sIlNjb3BlcyI6WyJXb3JrZXIiLCJSdW5FeGVjdXRpb24iXSwiZXhwIjoxNTYwNTA5NTQyLCJqdGkiOiIyYjUwN2Q2Yi05YWVmLTRjZTUtODM4ZS0wNTk1OTY4YzBlOTUiLCJpYXQiOjE1NjA1MDU5NDIsImlzcyI6ImNkc190ZXN0Iiwic3ViIjoiMTU4ODY5M2YtOTE5NC00ODg5LWJmYjAtZWY3Nzc5M2QzY2ViIn0.jLot6mtYHdnNAKxUS7OK7d6fVyMQyc7fS2NW4s727dxjx01Q2pPUQJBr16gKsS4ETSKh2ik7kqGGXdOz3i67DxMlPHcs0Azka1VOlefPcA77is-oVu0MPh4JbL0KA7fCu_98VKLJH3B0jYr4HEG9285ZOjFg7L5yuR7OqeFfCE3MrigyMKaNOrNE2FohOK9o50GyW_pAr6uNXcTu-yvqQUsz2B2gsd90HK2iWnvb8pKnBVVPg9Q0VA5l2IoFZR_p_UKSJZcFyFnjWKBVy33b70xupDnCpD-3OcbIFAQ5NPRvU_BnEjj_Jm59Ljrv3pJt1ErTMTkMA9QIFdYkDp5a6Q")

	gock.New("http://lolcat.host").Get("/worker/model").
		HeaderPresent("Authorization").
		Reply(200).
		JSON([]sdk.Model{
			{
				ID:   1,
				Name: "my-worker-model",
			},
		})

	gock.New("http://lolcat.host").Post("/worker/waiting").Times(2).
		HeaderPresent("Authorization").
		Reply(200).JSON(nil)

	gock.New("http://lolcat.host").Get("/queue/workflows/42/infos").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID:     42,
			Status: sdk.StatusWaiting,
		})

	gock.New("http://lolcat.host").Get("/queue/workflows/42/infos").Times(2).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID:     42,
			Status: sdk.StatusBuilding,
		})

	gock.New("http://lolcat.host").Post("/queue/workflows/42/take").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(
			sdk.WorkflowNodeJobRunData{
				Secrets: []sdk.Variable{
					{
						Name:  "cds.myPassword",
						Value: "my very sensitive data",
					},
				},
				NodeJobRun: sdk.WorkflowNodeJobRun{
					ID: 42,
					Parameters: []sdk.Parameter{
						{
							Name:  "cds.project",
							Value: "my-project",
						},
						{
							Name:  "cds.workflow",
							Value: "my-workflow",
						},
						{
							Name:  "cds.node",
							Value: "my-node",
						},
						{
							Name:  "cds.job",
							Value: "my-job",
						},
					},
					Job: sdk.ExecutedJob{
						Job: sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name:    "First Job",
								Enabled: true,
								Actions: []sdk.Action{
									{
										Name:     sdk.ScriptAction,
										Type:     sdk.BuiltinAction,
										Enabled:  true,
										StepName: "sleep",
										Parameters: []sdk.Parameter{
											{
												Name:  "script",
												Value: "#!/bin/bash\nset -ex\nsleep 10\necho my password should not be displayed here: {{.cds.myPassword}}\necho $CDS_EXPORT_PORT\nworker export newvar newval",
											},
										},
									}, {
										Name:     sdk.CheckoutApplicationAction,
										Type:     sdk.BuiltinAction,
										Enabled:  true,
										StepName: "checkout",
										Parameters: []sdk.Parameter{
											{
												Name:  "directory",
												Value: "{{.cds.workspace}}",
											},
										},
									},
									{
										Name:           "my-default-action",
										Type:           sdk.DefaultAction,
										Enabled:        true,
										AlwaysExecuted: true,
										Parameters: []sdk.Parameter{
											{
												Name:  "directory",
												Value: "{{.cds.workspace}}",
											},
										},
										Actions: []sdk.Action{
											{
												Name:     sdk.ScriptAction,
												Type:     sdk.BuiltinAction,
												Enabled:  true,
												StepName: "change directory",
												Parameters: []sdk.Parameter{
													{
														Name:  "script",
														Value: "cd {{.directory}}\nenv",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/step").Times(6).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/log").Times(2).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/result").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/auth/consumer/worker/signout").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	var logBuffer = new(bytes.Buffer)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))

			switch mock.Request().URLStruct.String() {
			case "http://lolcat.host/queue/workflows/42/step":
				var result sdk.StepStatus
				err := json.Unmarshal(bodyContent, &result)
				assert.NoError(t, err)
				switch result.StepOrder {
				case 0:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusSuccess {
						t.Fail()
					}
				case 1:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusFail {
						t.Fail()
					}
				case 2:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusSuccess {
						t.Fail()
					}
				default:
					t.Fail()
				}
			case "http://lolcat.host/queue/workflows/42/log":
				var log sdk.Log
				err := json.Unmarshal(bodyContent, &log)
				assert.NoError(t, err)
				logBuffer.WriteString(log.GetVal()) // nolint
			case "http://lolcat.host/queue/workflows/42/result":
				var result sdk.Result
				err := json.Unmarshal(bodyContent, &result)
				assert.NoError(t, err)
				assert.Equal(t, int64(42), result.BuildID)
				assert.Equal(t, sdk.StatusFail, result.Status)
				assert.Equal(t, "cds.build.newvar", result.NewVariables[0].Name)
				assert.Equal(t, "newval", result.NewVariables[0].Value)
			}
		}
	}

	gock.Observe(checkRequest)

	var w = new(internal.CurrentWorker)

	fs := afero.NewOsFs()
	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, fs); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPSSEClient())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := internal.StartWorker(ctx, w, 42)
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
	if !gock.IsDone() {
		pending := gock.Pending()
		for _, m := range pending {
			t.Logf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
		}
	}
	assert.False(t, gock.HasUnmatchedRequest(), "gock should not have unmatched request")
	if gock.HasUnmatchedRequest() {
		reqs := gock.GetUnmatchedRequests()
		for _, req := range reqs {
			t.Logf("Request %s %s unmatched", req.Method, req.URL.String())
		}
	}

	assert.Equal(t, 2, strings.Count(logBuffer.String(), "my password should not be displayed here: **********\n"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "[INFO] CDS_BUILD_NEWVAR=newval"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "[INFO] CDS_KEY=********"))

}
