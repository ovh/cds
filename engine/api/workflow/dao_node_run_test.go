package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func TestCommitListWorkflowRun(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.TODO(), db.DbMap, cache, nil)

	mockVCSSservice, _ := assets.InsertService(t, db, "TestManualRunBuildParameterMultiApplication", sdk.TypeVCS)
	defer func() {
		services.Delete(db, mockVCSSservice) // nolint
	}()

	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = io.NopCloser(body)

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
			case "/vcs/github/repos/sguiheux/demo2":
				repo := sdk.VCSRepo{
					URL:          "https",
					Name:         "demo",
					ID:           "123",
					Fullname:     "sguiheux/demo2",
					Slug:         "sguiheux",
					HTTPCloneURL: "https://github.com/sguiheux/demo2.git",
					SSHCloneURL:  "git://github.com/sguiheux/demo2.git",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(w, err)
				}
				// Default payload on workflow insert
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=&default=true":
				b := sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode([]sdk.VCSBranch{b}); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/sguiheux/demo2/branches/?branch=&default=true":
				b := sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					LatestCommit: "mylastcommit2",
				}
				if err := enc.Encode([]sdk.VCSBranch{b}); err != nil {
					return writeError(w, err)
				}
				// NEED GET BRANCH TO GET LATEST COMMIT
			case "/vcs/github/repos/sguiheux/demo/branches/?branch=master&default=false":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode(b); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/sguiheux/demo2/branches/?branch=master&default=false":
				b := sdk.VCSBranch{
					Default:      false,
					DisplayID:    "master",
					LatestCommit: "mylastcommit2",
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
			case "/vcs/github/repos/sguiheux/demo2/commits/mylastcommit2":
				c := sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:  "test",
						Email: "sg@foo.bar",
					},
					Hash:      "mylastcommit2",
					Message:   "super commit2",
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
	consumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "github",
		Type:      sdk.VCSTypeGithub,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

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
	app, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u, nil)
	assert.NoError(t, globalError)

	// Add application2
	appS2 := `version: v1.0
name: blabla2
vcs_server: github
repo: sguiheux/demo2
vcs_ssh_key: proj-blabla
`
	var eapp2 = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS2), eapp2))
	app2, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp2, application.ImportOptions{Force: true}, nil, u, nil)
	assert.NoError(t, globalError)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_commits_list",
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
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	w2 := sdk.Workflow{
		Name:       "test_commits_list_second",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app2.ID,
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w2))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_commits_list", workflow.LoadOptions{DeepPipeline: true})
	test.NoError(t, err)
	w2t, err := workflow.Load(context.TODO(), db, cache, *proj, "test_commits_list_second", workflow.LoadOptions{DeepPipeline: true})
	test.NoError(t, err)

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

	wr2, errWR := workflow.CreateRun(db.DbMap, w2t, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, errWR)
	wr2.Workflow = *w2t
	_, errWr2 := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
				"git.author": "test",
			},
		},
	}, *consumer, nil)
	test.NoError(t, errWr2)

	current2 := sdk.BuildNumberAndHash{BuildNumber: 10000}
	r2, err := workflow.PreviousNodeRunVCSInfos(context.Background(), db, proj.Key, *w2t, "node1", current2, app2.ID, 0)
	require.NoError(t, err)
	require.NotEmpty(t, r2.Branch)
	t.Logf("key:%s w2t:%d current:%v app2.ID:%d", proj.Key, w1.ID, current2.Hash, app2.ID)
	require.Equal(t, "mylastcommit2", r2.Hash)

	current := sdk.BuildNumberAndHash{BuildNumber: 10000}
	r, err := workflow.PreviousNodeRunVCSInfos(context.Background(), db, proj.Key, *w1, "node1", current, app.ID, 0)
	require.NoError(t, err)
	require.NotEmpty(t, r.Branch)
	t.Logf("key:%s w1:%d current:%v app.ID:%d", proj.Key, w1.ID, current.Hash, app.ID)
	require.Equal(t, "mylastcommit", r.Hash)
}
