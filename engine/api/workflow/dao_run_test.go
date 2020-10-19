package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
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

func TestCanBeRun(t *testing.T) {
	wnrs := map[int64][]sdk.WorkflowNodeRun{}
	node1 := sdk.Node{ID: 25}
	nodeRoot := sdk.Node{
		ID:   10,
		Type: sdk.NodeTypePipeline,
		Triggers: []sdk.NodeTrigger{
			{
				ChildNode: node1,
			},
		},
	}
	wnrs[nodeRoot.ID] = []sdk.WorkflowNodeRun{
		{ID: 3, WorkflowNodeID: nodeRoot.ID, Status: sdk.StatusBuilding},
	}
	wnrs[node1.ID] = []sdk.WorkflowNodeRun{
		{ID: 3, WorkflowNodeID: node1.ID, Status: sdk.StatusFail},
	}
	wr := &sdk.WorkflowRun{
		Workflow: sdk.Workflow{
			Name:       "test_1",
			ProjectID:  1,
			ProjectKey: "key",
			WorkflowData: sdk.WorkflowData{
				Node: nodeRoot,
			},
		},
		WorkflowID:       2,
		WorkflowNodeRuns: wnrs,
	}

	wnr := &sdk.WorkflowNodeRun{
		WorkflowNodeID: node1.ID,
		Status:         sdk.StatusSuccess, // a node node always have a status
	}

	ts := []struct {
		status   string
		canBeRun bool
	}{
		{status: sdk.StatusBuilding, canBeRun: false},
		{status: sdk.StatusSuccess, canBeRun: true},
	}

	for _, tc := range ts {
		wnrs[nodeRoot.ID][0].Status = tc.status
		test.Equal(t, tc.canBeRun, workflow.CanBeRun(wr, wnr))
	}
}

func TestPurgeWorkflowRun(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	mockVCSSservice, _ := assets.InsertService(t, db, "TestManualRunBuildParameterMultiApplication", sdk.TypeVCS)
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
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
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
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
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
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
	app, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
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
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		require.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	_, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 2, count, "Number of workflow runs isn't correct")
}

func TestPurgeWorkflowRunWithRunningStatus(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

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

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)
		wfr.Status = sdk.StatusBuilding
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	_, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 5, count, "Number of workflow runs isn't correct")
}

func TestPurgeWorkflowRunWithOneSuccessWorkflowRun(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	mockVCSSservice, _ := assets.InsertService(t, db, "TestManualRunBuildParameterMultiApplication", sdk.TypeVCS)
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
	}()

	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/vcs/github/repos/sguiheux/demo/branches":
				branches := []sdk.VCSBranch{
					{
						ID:        "master",
						DisplayID: "master",
					},
				}
				if err := enc.Encode(branches); err != nil {
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
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
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
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
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
	app, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
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
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
				"git.author": "test",
			},
		},
	}, *consumer, nil)
	test.NoError(t, errWr)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)

		wfr.Status = sdk.StatusFail
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 3, count, "Number of workflow runs isn't correct")
	wfInSuccess := false
	for _, wfRun := range wruns {
		if wfRun.Status == sdk.StatusSuccess {
			wfInSuccess = true
		}
	}

	test.Equal(t, true, wfInSuccess, "The workflow should keep at least one workflow run in success")
}

func TestPurgeWorkflowRunWithNoSuccessWorkflowRun(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	mockVCSSservice, _ := assets.InsertService(t, db, "TestManualRunBuildParameterMultiApplication", sdk.TypeVCS)
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
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
				// NEED GET COMMIT TO GET AUTHOR AND MESSAGE
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
			default:

			}

			return w, nil
		},
	)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
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
	app, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
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
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)

		wfr.Status = sdk.StatusFail
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	n := workflow.CountWorkflowRunsMarkToDelete(context.TODO(), db, nil)
	assert.True(t, n >= 3, "At least 3 runs must be mark to delete")

	_, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 2, count, "Number of workflow runs isn't correct")
}

func TestPurgeWorkflowRunWithoutTags(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

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

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
	}
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	branches := []string{"master", "master", "master", "develop", "develop", "testBr", "testBr", "testBr", "testBr", "test4"}
	for i := 0; i < 10; i++ {
		wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": branches[i],
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	_, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 3, count, "Number of workflow runs isn't correct")
}

func TestPurgeWorkflowRunWithoutTagsBiggerHistoryLength(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

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

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 20,
	}
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_purge_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	branches := []string{"master", "master", "master", "develop", "develop", "testBr", "testBr", "testBr", "testBr", "test4"}
	for i := 0; i < 10; i++ {
		wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.Username,
				Payload: map[string]string{
					"git.branch": branches[i],
					"git.author": "test",
				},
			},
		}, *consumer, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.TODO(), db, *w1)
	test.NoError(t, errP)

	_, _, _, count, errRuns := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 10, count, "Number of workflow runs isn't correct")
}

func TestLoadRunsIDsToDelete(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_, _ = db.Exec("update workflow_run set to_delete=false ")

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	wr1, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	assert.NoError(t, err)

	wr2, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	assert.NoError(t, err)

	wr1.ToDelete = true
	wr2.ToDelete = true

	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr1))
	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr2))

	ids, offset, limit, count, err := workflow.LoadRunsIDsToDelete(db, 0, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, ids[0], wr1.ID)
	require.Equal(t, int64(0), offset)
	require.Equal(t, int64(1), limit)
	require.Equal(t, int64(2), count)

	ids, offset, limit, count, err = workflow.LoadRunsIDsToDelete(db, 1, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, ids[0], wr2.ID)
	require.Equal(t, int64(1), offset)
	require.Equal(t, int64(1), limit)
	require.Equal(t, int64(2), count)

	ids, offset, limit, count, err = workflow.LoadRunsIDsToDelete(db, 0, 50)
	require.NoError(t, err)
	require.Len(t, ids, 2)

}
