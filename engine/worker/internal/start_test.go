package internal_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func init() {
	log.Initialize(context.TODO(), &log.Conf{Level: "debug"})
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
				GelfServiceAddr: "localhost:8090",
				NodeJobRun: sdk.WorkflowNodeJobRun{
					ID: 42,
					Parameters: []sdk.Parameter{
						{
							Name:  "cds.version", // used to compute cds.semver
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
									}, {
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

	gock.New("http://lolcat.host").Post("/queue/workflows/42/step").Times(8).
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
			logBuffer.WriteString(m.Full + "\n")
		}

	}()

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
			case "http://lolcat.host/queue/workflows/42/result":
				var result sdk.Result
				err := json.Unmarshal(bodyContent, &result)
				assert.NoError(t, err)
				assert.Equal(t, int64(42), result.BuildID)
				assert.Equal(t, sdk.StatusFail, result.Status)
				if len(result.NewVariables) > 0 {
					assert.Equal(t, "cds.build.newvar", result.NewVariables[0].Name)
					// assert.Equal(t, "cds.semver", result.NewVariables[0].Name)
					// assert.Equal(t, "git.describe", result.NewVariables[0].Name)
					assert.Equal(t, "newval", result.NewVariables[0].Value)
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
	log.Debug("creating basedir %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPSSEClient())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = internal.StartWorker(ctx, w, 42)
	assert.NoError(t, err)

	var isDone bool
	if gock.IsDone() {
		isDone = true
	}
	if !isDone {
		pending := gock.Pending()
		isDone = true
		for _, m := range pending {
			t.Logf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
			if m.Request().URLStruct.String() != "http://lolcat.host/queue/workflows/42/log" {
				isDone = false
			}
		}
	}
	assert.True(t, isDone)
	if gock.HasUnmatchedRequest() {
		reqs := gock.GetUnmatchedRequests()
		for _, req := range reqs {
			if req.URL.String() != "http://lolcat.host/queue/workflows/42/log" {
				t.Logf("Request %s %s unmatched", req.Method, req.URL.String())
				t.Fail()
			}
		}
	}

	assert.Equal(t, 2, strings.Count(logBuffer.String(), "my password should not be displayed here: **********\n"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_BUILD_NEWVAR=newval"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_KEY=********"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_API_URL=http://lolcat.host"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "CDS_SEMVER=0.1.0+cds.1"))
	assert.Equal(t, 1, strings.Count(logBuffer.String(), "GIT_DESCRIBE=0.1.0"))
	assert.Equal(t, 0, strings.Count(logBuffer.String(), "CDS_BUILD_CDS_BUILD"))
}
