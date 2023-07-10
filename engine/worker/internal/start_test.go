package internal_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func init() {
	cdslog.Initialize(context.TODO(), &cdslog.Conf{Level: "debug"})
}

func TestStartWorkerWithABookedJob(t *testing.T) {
	defer gock.Off()

	gock.New("http://cds-api.local").Get("/action/requirement").
		Reply(200).
		JSON([]sdk.Requirement{
			{
				Name:  "bash",
				Type:  sdk.BinaryRequirement,
				Value: "bash",
			},
		})

	modelID := int64(1)
	gock.New("http://cds-api.local").Post("/auth/consumer/worker/signin").
		HeaderPresent("Authorization").
		Reply(201).
		JSON(sdk.Worker{
			ID:      "xxxx-xxxx-xxxxx",
			ModelID: &modelID,
		}).AddHeader("X-CDS-JWT", "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJJRCI6IjJiNTA3ZDZiLTlhZWYtNGNlNS04MzhlLTA1OTU5NjhjMGU5NSIsIkdyb3VwSURzIjpbMV0sIlNjb3BlcyI6WyJXb3JrZXIiLCJSdW5FeGVjdXRpb24iXSwiZXhwIjoxNTYwNTA5NTQyLCJqdGkiOiIyYjUwN2Q2Yi05YWVmLTRjZTUtODM4ZS0wNTk1OTY4YzBlOTUiLCJpYXQiOjE1NjA1MDU5NDIsImlzcyI6ImNkc190ZXN0Iiwic3ViIjoiMTU4ODY5M2YtOTE5NC00ODg5LWJmYjAtZWY3Nzc5M2QzY2ViIn0.jLot6mtYHdnNAKxUS7OK7d6fVyMQyc7fS2NW4s727dxjx01Q2pPUQJBr16gKsS4ETSKh2ik7kqGGXdOz3i67DxMlPHcs0Azka1VOlefPcA77is-oVu0MPh4JbL0KA7fCu_98VKLJH3B0jYr4HEG9285ZOjFg7L5yuR7OqeFfCE3MrigyMKaNOrNE2FohOK9o50GyW_pAr6uNXcTu-yvqQUsz2B2gsd90HK2iWnvb8pKnBVVPg9Q0VA5l2IoFZR_p_UKSJZcFyFnjWKBVy33b70xupDnCpD-3OcbIFAQ5NPRvU_BnEjj_Jm59Ljrv3pJt1ErTMTkMA9QIFdYkDp5a6Q")

	gock.New("http://cds-api.local").Get("/worker/model").
		HeaderPresent("Authorization").
		Reply(200).
		JSON([]sdk.Model{
			{
				ID:   1,
				Name: "my-worker-model",
			},
		})

	gock.New("http://cds-api.local").Post("/worker/waiting").Times(1).
		HeaderPresent("Authorization").
		Reply(200).JSON(nil)

	gock.New("http://cds-api.local").Get("/queue/workflows/42/infos").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID:     42,
			Status: sdk.StatusWaiting,
		})

	gock.New("http://cds-api.local").Get("/queue/workflows/42/infos").Times(2).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID:     42,
			Status: sdk.StatusBuilding,
		})

	gock.New("http://cds-api.local").Post("/queue/workflows/42/spawn/infos").Times(3).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://cds-api.local").Get("project/proj_key/workflows/workflow_name/runs/0").Times(1).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowRun{
			Workflow: sdk.Workflow{
				Integrations: []sdk.WorkflowProjectIntegration{
					{
						ProjectIntegration: sdk.ProjectIntegration{
							Name: "artifactory",
						},
					},
				},
			},
		})

	gock.New("http://cds-api.local").Get("project/proj_key/integrations/artifactory/workerhooks").Times(1).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkerHookProjectIntegrationModel{
			Configuration: sdk.WorkerHookSetupTeardownConfig{
				ByCapabilities: map[string]sdk.WorkerHookSetupTeardownScripts{
					"bash": {
						Label: "first_hook",
						Setup: `
#!/bin/bash
export FOO_FROM_HOOK=BAR`,
						Teardown: "unset FOO_FROM_HOOK",
					},
				},
			},
		})

	gock.New("http://cds-api.local").Post("/queue/workflows/42/take").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(
			sdk.WorkflowNodeJobRunData{
				ProjectKey:   "proj_key",
				WorkflowName: "workflow_name",
				Secrets: []sdk.Variable{
					{
						Name:  "cds.myPassword",
						Value: "my very sensitive data",
					},
				},
				GelfServiceAddr: "localhost:8090",
				NodeJobRun: sdk.WorkflowNodeJobRun{
					ID: 42,
					Parameters: []sdk.Parameter{
						{
							Name:  "cds.run.number", // used to compute cds.semver
							Value: "1",
						},
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
						{
							Name:  "git.http_url", // simulate an application attached to the pipeline
							Value: "https://github.com/fsamin/dummy-empty-repo.git",
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
									},
									{
										Name:     sdk.GitCloneAction,
										Type:     sdk.BuiltinAction,
										Enabled:  true,
										StepName: "gitClone",
										Parameters: []sdk.Parameter{
											{
												Name:  "directory",
												Value: "{{.cds.workspace}}",
											},
											{
												Name:  "url",
												Value: "https://github.com/fsamin/dummy-empty-repo.git",
											},
											{
												Name:  "depth",
												Value: "false",
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
									{
										Name:     sdk.ScriptAction,
										Type:     sdk.BuiltinAction,
										Enabled:  true,
										StepName: "edit with failure",
										Parameters: []sdk.Parameter{
											{
												Name:  "script",
												Value: "#!/bin/bash\nset -ex\nexit 1",
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

	gock.New("http://cds-api.local").Post("/queue/workflows/42/step").Times(8).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://cds-api.local").Post("/queue/workflows/42/result").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://cds-api.local").Post("/auth/consumer/worker/signout").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	var logMessages []hook.Message
	listener, err := net.Listen("tcp", "localhost:8090")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		require.NoError(t, err)
		bufReader := bufio.NewReader(conn)
		defer conn.Close() //nolint
		for {
			bytes, err := bufReader.ReadBytes(byte(0))
			if err != nil {
				return
			}
			// remove byte(0)
			bytes = bytes[:len(bytes)-1]
			m := hook.Message{}
			require.NoError(t, m.UnmarshalJSON(bytes))
			logMessages = append(logMessages, m)
		}
	}()

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			switch mock.Request().URLStruct.String() {
			case "http://cds-api.local/queue/workflows/42/step":
				var result sdk.StepStatus
				err := json.Unmarshal(bodyContent, &result)
				require.NoError(t, err)

				switch result.StepOrder {
				case 0:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusSuccess {
						t.Logf("Wrong status on step 0")
						t.Fail()
					}
				case 1:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusSuccess {
						t.Logf("Wrong status on step 1")
						t.Fail()
					}
				case 2:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusSuccess {
						t.Logf("Wrong status on step 2")
						t.Fail()
					}
				case 3:
					if result.Status != sdk.StatusBuilding && result.Status != sdk.StatusFail {
						t.Logf("Wrong status on step 3")
						t.Fail()
					}
				default:
					t.Logf("This case should not happend")
					t.Fail()
				}
			case "http://cds-api.local/queue/workflows/42/result":
				var result sdk.Result
				require.NoError(t, json.Unmarshal(bodyContent, &result))
				require.Equal(t, int64(42), result.BuildID)
				require.Equal(t, sdk.StatusFail, result.Status)
				if len(result.NewVariables) > 0 {
					require.Equal(t, "cds.build.newvar", result.NewVariables[0].Name)
					// assert.Equal(t, "cds.semver", result.NewVariables[0].Name)
					// assert.Equal(t, "git.describe", result.NewVariables[0].Name)
					require.Equal(t, "newval", result.NewVariables[0].Value)
				} else {
					t.Error("missing new variables")
				}
			}
		}
	}

	gock.Observe(checkRequest)

	var w = new(internal.CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	log.Debug(context.TODO(), "creating basedir %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	cfg := &workerruntime.WorkerConfig{
		Name:                "test-worker",
		HatcheryName:        "test-hatchery",
		APIEndpoint:         "http://cds-api.local",
		APIToken:            "xxx-my-token",
		APIEndpointInsecure: true,
		Model:               "my-model",
		Region:              "local-test",
		Basedir:             basedir,
		GelfServiceAddr:     "localhost:8090",
	}
	if err := w.Init(cfg, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPNoTimeoutClient())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	require.NoError(t, internal.StartWorker(ctx, w, "42"))

	var isDone bool
	if gock.IsDone() {
		isDone = true
	}
	if !isDone {
		pending := gock.Pending()
		isDone = true
		for _, m := range pending {
			t.Logf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
			if m.Request().URLStruct.String() != "http://cds-api.local/queue/workflows/42/log" {
				isDone = false
			}
		}
	}
	require.True(t, isDone)
	if gock.HasUnmatchedRequest() {
		reqs := gock.GetUnmatchedRequests()
		for _, req := range reqs {
			if req.URL.String() != "http://cds-api.local/queue/workflows/42/log" {
				t.Logf("Request %s %s unmatched", req.Method, req.URL.String())
				t.Fail()
			}
		}
	}

	var logBuffer = new(bytes.Buffer)
	var countTerminatedEndStepLog int
	for i := range logMessages {
		logBuffer.WriteString(logMessages[i].Full + "\n")
		terminatedI := logMessages[i].Extra["_"+cdslog.ExtraFieldTerminated]
		if cast.ToBool(terminatedI) {
			countTerminatedEndStepLog++
		}
	}

	require.Equal(t, 4, countTerminatedEndStepLog, "Only root steps should send end log with terminated state")
	t.Logf("%v", logBuffer.String())

	assert.Equal(t, 2, strings.Count(logBuffer.String(), "Starting step \"Script\""))
	assert.Equal(t, 2, strings.Count(logBuffer.String(), "End of step \"Script\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "Starting step \"GitClone\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "End of step \"GitClone\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "Starting step \"my-default-action\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "Starting sub step \"/change directory\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "End of sub step \"/change directory\""))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "End of step \"my-default-action\""))

	assert.Equal(t, 2, strings.Count(logBuffer.String(), "my password should not be displayed here: **********\n"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_BUILD_NEWVAR=newval"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_KEY=********"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_API_URL=http://cds-api.local"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_SEMVER=0.1.0+cds.1"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "GIT_DESCRIBE=0.1.0"))
	assert.Equal(t, 0, strings.Count(logBuffer.String(), "CDS_BUILD_CDS_BUILD"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "HATCHERY_MODEL=my-model"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "HATCHERY_NAME=test-hatchery"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "HATCHERY_WORKER=test-worker"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "HATCHERY_REGION=local-test"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "BASEDIR="))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "FOO_FROM_HOOK=BAR"))
}
