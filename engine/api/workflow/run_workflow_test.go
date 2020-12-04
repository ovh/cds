package workflow_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestManualRun1(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	ctx := context.Background()

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	t.Logf("w1: %+v", w1)
	require.NoError(t, err)

	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1

	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	}, *consumer, nil)
	require.NoError(t, errS)

	wr2, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr2.Workflow = *w1
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
		},
	}, *consumer, nil)
	require.NoError(t, errS)

	//LoadLastRun
	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), lastrun.Number)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, err)

	//don't want to compare queueSeconds attributes and spawn infos attributes
	nodeRun.Stages[0].RunJobs[0].QueuedSeconds = 0
	lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].QueuedSeconds = 0
	nodeRun.Stages[0].RunJobs[0].SpawnInfos = nil
	lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].SpawnInfos = nil

	test.Equal(t, lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0], nodeRun)

	//TestLoadNodeJobRun
	filter := workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group)).ToIDs())
	require.NoError(t, err)
	test.Equal(t, 2, len(jobs))

	//TestprocessWorkflowRun
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
		},
		FromNodeIDs: []int64{wr2.Workflow.WorkflowData.Node.ID},
	}, *consumer, nil)
	require.NoError(t, errS)

	//TestLoadRuns
	runs, offset, limit, count, err := workflow.LoadRunsSummaries(db, proj.Key, w1.Name, 0, 50, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, offset)
	assert.Equal(t, 50, limit)
	assert.Equal(t, 2, count)
	assert.Len(t, runs, 2)

	//TestLoadRunByID
	_, err = workflow.LoadRunByIDAndProjectKey(db, proj.Key, wr2.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)
}

func TestManualRun2(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	ctx := context.Background()

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	require.NoError(t, errS)

	wr2, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr2.Workflow = *w1
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	require.NoError(t, errS)

	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual:      &sdk.WorkflowNodeRunManual{Username: u.Username},
		FromNodeIDs: []int64{wr.Workflow.WorkflowData.Node.ID},
	}, *consumer, nil)
	require.NoError(t, errS)

	filter := workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group)).ToIDs())
	require.NoError(t, err)

	assert.Len(t, jobs, 3)
}

func TestManualRun3(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	// Remove all job in queue
	filterClean := workflow.NewQueueFilter()
	nrj, _ := workflow.LoadNodeJobRunQueue(context.TODO(), db, cache, filterClean)
	for _, j := range nrj {
		_ = workflow.DeleteNodeJobRuns(db, j.WorkflowNodeRunID)
	}

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Add variable
	v := sdk.ProjectVariable{
		Name:  "foo",
		Type:  sdk.SecretVariable,
		Value: "bar",
	}
	if err := project.InsertVariable(db, proj.ID, &v, u); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	g0 := sdk.Group{Name: "g0"}
	g1 := sdk.Group{Name: "g1"}
	for _, g := range []sdk.Group{g0, g1} {
		oldg, _ := group.LoadByName(context.TODO(), db, g.Name)
		if oldg != nil {
			links, err := group.LoadLinksGroupProjectForGroupID(context.TODO(), db, oldg.ID)
			require.NoError(t, err)
			for _, l := range links {
				require.NoError(t, group.DeleteLinkGroupProject(db, &l))
			}
			require.NoError(t, group.Delete(context.TODO(), db, oldg))
		}
	}

	require.NoError(t, group.Insert(context.TODO(), db, &g0))
	require.NoError(t, group.Insert(context.TODO(), db, &g1))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g0.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g1.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	g, err := group.LoadByName(context.TODO(), db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	modelIntegration := sdk.IntegrationModel{
		Name:       sdk.RandomString(10),
		Deployment: true,
	}
	require.NoError(t, integration.InsertModel(db, &modelIntegration))
	t.Logf("### Integration model %s created with id: %d\n", modelIntegration.Name, modelIntegration.ID)

	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
			"mypassword": sdk.IntegrationConfigValue{
				Description: "here isa password",
				Type:        sdk.IntegrationConfigTypePassword,
				Value:       "mypassword",
			},
		},
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		Model:              modelIntegration,
		IntegrationModelID: modelIntegration.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &projInt))
	t.Logf("### Integration %s created with id: %d\n", projInt.Name, projInt.ID)

	p := sdk.GRPCPlugin{
		Author:             "unitTest",
		Description:        "desc",
		Name:               sdk.RandomString(10),
		Type:               sdk.GRPCPluginDeploymentIntegration,
		IntegrationModelID: &modelIntegration.ID,
		Integration:        modelIntegration.Name,
		Binaries: []sdk.GRPCPluginBinary{
			{
				OS:   "linux",
				Arch: "adm64",
				Name: "blabla",
			},
		},
	}

	require.NoError(t, plugin.Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	model, _ := workermodel.LoadByNameAndGroupID(context.TODO(), db, "TestManualRun", g.ID)
	if model == nil {
		model = &sdk.Model{
			Name:    "TestManualRun",
			GroupID: g.ID,
			Type:    sdk.Docker,
			ModelDocker: sdk.ModelDocker{
				Image: "buildpack-deps:jessie",
			},
			RegisteredCapabilities: sdk.RequirementList{
				{
					Name:  "capa1",
					Type:  sdk.BinaryRequirement,
					Value: "1",
				},
			},
		}

		if err := workermodel.Insert(context.TODO(), db, model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	// one pipeline with two stages
	s := sdk.NewStage("stage1-pipeline1")
	s2 := sdk.NewStage("stage2-pipeline1")
	s.Enabled = true
	s2.Enabled = true
	s.PipelineID = pip.ID
	s2.PipelineID = pip.ID
	require.NoError(t, pipeline.InsertStage(db, s))
	require.NoError(t, pipeline.InsertStage(db, s2))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled:      true,
			Name:         "job10",
			Requirements: []sdk.Requirement{{Name: "TestManualRun", Value: "TestManualRun", Type: sdk.ModelRequirement}},
		},
	}
	j2 := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Name:    "job11",
		},
	}
	require.NoError(t, pipeline.InsertJob(db, j, s.ID, &pip))
	require.NoError(t, pipeline.InsertJob(db, j2, s2.ID, &pip))
	s.Jobs = append(s.Jobs, *j)
	s2.Jobs = append(s.Jobs, *j2)

	pip.Stages = append(pip.Stages, *s)
	pip.Stages = append(pip.Stages, *s2)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip2))
	s = sdk.NewStage("stage 1-pipeline2")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled:      true,
			Name:         "job20",
			Requirements: []sdk.Requirement{{Name: "fooNameService", Value: "valueService", Type: sdk.ServiceRequirement}},
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

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
					PipelineID:           pip.ID,
					ProjectIntegrationID: projInt.ID,
				},
				Groups: []sdk.GroupPermission{
					{
						Group:      g0,
						Permission: 777,
					},
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip2.ID,
							},
							Groups: []sdk.GroupPermission{
								{
									Group:      g1,
									Permission: 777,
								},
							},
						},
					},
				},
			},
		},
	}

	proj, err = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups, project.LoadOptions.WithVariablesWithClearPassword, project.LoadOptions.WithKeys, project.LoadOptions.WithIntegrations)
	require.NoError(t, err)

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)

	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	require.NoError(t, errS)

	filter := workflow.NewQueueFilter()
	// test nil since/until
	_, err = workflow.CountNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group)).ToIDs())
	require.NoError(t, err)

	// queue should be empty with since 0,0 until 0,0
	t0 := time.Unix(0, 0)
	t1 := time.Unix(0, 0)

	filter.Since = &t0
	filter.Until = &t1

	countAlreadyInQueueNone, err := workflow.CountNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group)).ToIDs())
	require.NoError(t, err)
	assert.Equal(t, 0, int(countAlreadyInQueueNone.Count))

queueRun:
	filter3 := workflow.NewQueueFilter()
	filter3.Rights = sdk.PermissionReadExecute

	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter3, sdk.Groups(append(u.Groups, g0)).ToIDs())
	require.NoError(t, err)
	t.Logf("##### nb job in queue : %d\n", len(jobs))
	require.True(t, len(jobs) > 0)

	for i := range jobs {
		j := &jobs[i]

		t.Logf("##### work on job : %+v\n", j.Job.Action.Name)

		//BookNodeJobRun
		_, err = workflow.BookNodeJobRun(context.TODO(), cache, j.ID, &sdk.Service{
			CanonicalService: sdk.CanonicalService{
				Name: "Hatchery",
				ID:   1,
			},
		})
		require.NoError(t, err)

		sp := sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID}
		//AddSpawnInfosNodeJobRun
		err := workflow.AddSpawnInfosNodeJobRun(db, j.WorkflowNodeRunID, j.ID, []sdk.SpawnInfo{
			{
				APITime:     time.Now(),
				RemoteTime:  time.Now(),
				Message:     sp,
				UserMessage: sp.DefaultUserMessage(),
			},
		})
		assert.NoError(t, err)
		if t.Failed() {
			t.FailNow()
		}

		sp = sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID}
		//TakeNodeJobRun
		takenJobID := j.ID
		takenJob, _, _ := workflow.TakeNodeJobRun(context.TODO(), db, cache, *proj, takenJobID, "model", "worker", "1", []sdk.SpawnInfo{
			{
				APITime:     time.Now(),
				RemoteTime:  time.Now(),
				Message:     sp,
				UserMessage: sp.DefaultUserMessage(),
			},
		}, "hatchery_name")

		//Load workflow node run
		nodeRun, err := workflow.LoadNodeRunByID(db, takenJob.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			t.Fatal(err)
		}

		//Load workflow run
		workflowRun, err := workflow.LoadRunByID(db, nodeRun.WorkflowRunID, workflow.LoadRunOptions{})
		if err != nil {
			t.Fatal(err)
		}

		//TestAddLog
		require.NoError(t, workflow.AppendLog(db, j.ID, j.WorkflowNodeRunID, 1, "This is a log", workflow.DefaultMaxLogSize))
		require.NoError(t, workflow.AppendLog(db, j.ID, j.WorkflowNodeRunID, 1, "This is another log", workflow.DefaultMaxLogSize))

		j, err = workflow.LoadNodeJobRun(context.TODO(), db, cache, j.ID)
		require.NoError(t, err)
		assert.Equal(t, "hatchery_name", j.HatcheryName)
		assert.NotEmpty(t, j.WorkerName)
		assert.NotEmpty(t, j.Model)

		//TestUpdateNodeJobRunStatus
		_, err = workflow.UpdateNodeJobRunStatus(context.TODO(), db, cache, *proj, j, sdk.StatusSuccess)
		require.NoError(t, err)

		workflowRun, err = workflow.LoadRunByID(db, wr.ID, workflow.LoadRunOptions{})
		require.NoError(t, err)
		var jobRunFound bool
	checkJobRun:
		for _, noderuns := range workflowRun.WorkflowNodeRuns {
			for _, noderun := range noderuns {
				for _, stage := range noderun.Stages {
					for _, jobrun := range stage.RunJobs {
						t.Logf("checking job %d with %d", jobrun.ID, takenJobID)
						if jobrun.ID == j.ID {
							assert.Equal(t, "hatchery_name", jobrun.HatcheryName)
							assert.NotEmpty(t, jobrun.WorkerName)
							assert.NotEmpty(t, jobrun.Model)
							jobRunFound = true
							break checkJobRun
						}
					}
				}
			}
		}
		if !jobRunFound {
			t.Fatalf("unable to retrieve job run in the workflow run")
		}

		logs, err := workflow.LoadLogs(db, takenJob.ID)
		require.NoError(t, err)
		require.NotEmpty(t, logs)

		// check if there is another job to run
		if takenJob.Job.Action.Name == "job10" {
			goto queueRun
		} else if takenJob.Job.Action.Name == "job11" {
			assert.Equal(t, 2, len(takenJob.ExecGroups))
			// this pipeline is attached to an deployment integration
			// so, we check IntegrationPluginBinaries
			assert.Equal(t, 1, len(takenJob.IntegrationPluginBinaries))

			// Check ExecGroups
			var g0Found bool
			for _, eg := range takenJob.ExecGroups {
				if eg.Name != "shared.infra" && eg.Name != "g0" {
					t.Fatalf("this group %s should not be in execGroups", eg.Name)
				}
				if eg.Name == "g0" {
					g0Found = true
				}
			}
			if !g0Found {
				t.Fatal("g0 group not found in execGroups")
			}
		}
	}

	filter = workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs20, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
	require.NoError(t, err)
	assert.Equal(t, 1, len(jobs20))

	if len(jobs20) == 1 {
		assert.Equal(t, "Waiting", jobs20[0].Status)
		assert.Equal(t, "job20", jobs20[0].Job.Job.Action.Name)

		// test since / until
		t.Logf("##### jobs[0].Queued : %+v\n", jobs20[0].Queued)
		since := jobs20[0].Queued

		t0 := since.Add(-2 * time.Minute)
		t1 := since.Add(-1 * time.Minute)
		filter := workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.Since = &t0
		filter.Until = &t1
		jobsSince, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		for _, job := range jobsSince {
			if jobs20[0].ID == job.ID {
				assert.Fail(t, " this job should not be in queue since/until")
			}
		}

		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.Since = &t0
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		var found bool
		for _, job := range jobsSince {
			if jobs20[0].ID == job.ID {
				found = true
			}
		}
		if !found {
			assert.Fail(t, " this job should be in queue since")
		}

		t0 = since.Add(10 * time.Second)
		t1 = since.Add(15 * time.Second)
		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.Since = &t0
		filter.Until = &t1
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		for _, job := range jobsSince {
			if jobs20[0].ID == job.ID {
				assert.Fail(t, " this job should not be in queue since/until")
			}
		}

		// there is one job with a CDS Service prerequisiste
		// Getting queue with RatioService=100 -> we want this job only.
		// If we get a job without a service, it's a failure
		cent := 100
		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.RatioService = &cent
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		for _, job := range jobsSince {
			if !job.ContainsService {
				assert.Fail(t, " this job should not be in queue !job.ContainsService: job")
			}
		}

		// there is one job with a CDS Service prerequisiste
		// Getting queue with RatioService=0 -> we want job only without CDS Service.
		// If we get a job with a service, it's a failure
		zero := 0
		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.RatioService = &zero
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		for _, job := range jobsSince {
			if job.ContainsService {
				assert.Fail(t, " this job should not be in queue job.ContainsService")
			}
		}

		// there is one job with a CDS Model prerequisiste
		// we get the queue with a modelType openstack : we don't want
		// job with worker model type docker in result
		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.ModelType = []string{sdk.Openstack}
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.Groups(append(u.Groups, proj.ProjectGroups[0].Group, g0, g1)).ToIDs())
		require.NoError(t, err)
		// we don't want the job with the worker model "TestManualRun"
		for _, job := range jobsSince {
			if job.ModelType == sdk.Docker {
				assert.Fail(t, " this job should not be in queue with this model")
			}
		}
	}
}

func TestNoStage(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	require.NoError(t, errS)

	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	require.NoError(t, err)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, err)

	assert.Equal(t, sdk.StatusSuccess, nodeRun.Status)
}

func TestNoJob(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

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
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	require.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)

	wr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	require.NoError(t, errS)

	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	require.NoError(t, err)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, err)

	assert.Equal(t, sdk.StatusSuccess, nodeRun.Status)
}
