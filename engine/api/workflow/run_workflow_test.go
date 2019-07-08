package workflow_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestManualRun1(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	ctx := context.Background()

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

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
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))
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

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	}, u, nil)
	test.NoError(t, errS)

	wr2, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr2.Workflow = *w1
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
		},
	}, u, nil)
	test.NoError(t, errS)

	//LoadLastRun
	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.Equal(t, int64(2), lastrun.Number)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", 2, lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, err)

	//don't want to compare queueSeconds attributes and spawn infos attributes
	nodeRun.Stages[0].RunJobs[0].QueuedSeconds = 0
	lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].QueuedSeconds = 0
	nodeRun.Stages[0].RunJobs[0].SpawnInfos = nil
	lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].SpawnInfos = nil

	test.Equal(t, lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0], nodeRun)

	//TestLoadNodeJobRun
	filter := workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, append(sdk.GroupsToIDs(u.OldUserStruct.Groups), proj.ProjectGroups[0].Group.ID))
	test.NoError(t, err)
	test.Equal(t, 2, len(jobs))

	//TestprocessWorkflowRun
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
		},
		FromNodeIDs: []int64{wr2.Workflow.WorkflowData.Node.ID},
	}, u, nil)
	test.NoError(t, errS)

	//TestLoadRuns
	runs, offset, limit, count, err := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 50, nil)
	test.NoError(t, err)
	assert.Equal(t, 0, offset)
	assert.Equal(t, 50, limit)
	assert.Equal(t, 2, count)
	assert.Len(t, runs, 2)

	//TestLoadRunByID
	_, err = workflow.LoadRunByIDAndProjectKey(db, proj.Key, wr2.ID, workflow.LoadRunOptions{})
	test.NoError(t, err)
}

func TestManualRun2(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	ctx := context.Background()

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

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
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))
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

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, u, nil)
	test.NoError(t, errS)

	wr2, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr2.Workflow = *w1
	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr2, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, u, nil)
	test.NoError(t, errS)

	_, errS = workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual:      &sdk.WorkflowNodeRunManual{Username: u.Username},
		FromNodeIDs: []int64{wr.Workflow.WorkflowData.Node.ID},
	}, u, nil)
	test.NoError(t, errS)

	filter := workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, append(sdk.GroupsToIDs(u.OldUserStruct.Groups), proj.ProjectGroups[0].Group.ID))
	test.NoError(t, err)

	assert.Len(t, jobs, 3)
}

func TestManualRun3(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	ctx := context.Background()

	test.NoError(t, project.AddKeyPair(db, proj, "key", u))

	g, err := group.LoadByName(ctx, db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	model, _ := workermodel.LoadByNameAndGroupID(db, "TestManualRun", g.ID)
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

		if err := workermodel.Insert(db, model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled:      true,
			Name:         "job20",
			Requirements: []sdk.Requirement{{Name: "TestManualRun", Value: "TestManualRun", Type: sdk.ModelRequirement}},
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
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))
	s = sdk.NewStage("stage 1")
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
		WorkflowData: &sdk.WorkflowData{
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
								PipelineID: pip2.ID,
							},
						},
					},
				},
			},
		},
	}

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups, project.LoadOptions.WithVariablesWithClearPassword, project.LoadOptions.WithKeys)

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, u, nil)
	test.NoError(t, errS)

	filter := workflow.NewQueueFilter()
	// test nil since/until
	_, err = workflow.CountNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
	test.NoError(t, err)

	// queue should be empty with since 0,0 until 0,0
	t0 := time.Unix(0, 0)
	t1 := time.Unix(0, 0)

	filter.Since = &t0
	filter.Until = &t1

	countAlreadyInQueueNone, err := workflow.CountNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
	test.NoError(t, err)
	assert.Equal(t, 0, int(countAlreadyInQueueNone.Count))

	filter3 := workflow.NewQueueFilter()
	filter3.Rights = sdk.PermissionReadExecute

	jobs, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter3, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
	test.NoError(t, err)

	for i := range jobs {
		j := &jobs[i]
		tx, _ := db.Begin()

		//BookNodeJobRun
		_, err = workflow.BookNodeJobRun(cache, j.ID, &sdk.Service{
			CanonicalService: sdk.CanonicalService{
				Name: "Hatchery",
				ID:   1,
			},
		})
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		//AddSpawnInfosNodeJobRun
		err := workflow.AddSpawnInfosNodeJobRun(db, j.ID, []sdk.SpawnInfo{
			{
				APITime:    time.Now(),
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{
					ID: sdk.MsgSpawnInfoHatcheryStarts.ID,
				},
			},
		})
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		//TakeNodeJobRun
		j, _, _ = workflow.TakeNodeJobRun(context.TODO(), func() *gorp.DbMap { return db }, db, cache, proj, j.ID, "model", "worker", "1", []sdk.SpawnInfo{
			{
				APITime:    time.Now(),
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{
					ID: sdk.MsgSpawnInfoJobTaken.ID,
				},
			},
		})

		//Load workflow node run
		nodeRun, err := workflow.LoadNodeRunByID(db, j.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			t.Fatal(err)
		}

		//Load workflow run
		workflowRun, err := workflow.LoadRunByID(db, nodeRun.WorkflowRunID, workflow.LoadRunOptions{})
		if err != nil {
			t.Fatal(err)
		}

		secrets, err := workflow.LoadSecrets(db, cache, nodeRun, workflowRun, proj.Variable)
		assert.NoError(t, err)
		assert.Len(t, secrets, 1)

		//TestAddLog
		assert.NoError(t, workflow.AddLog(db, j, &sdk.Log{
			Val: "This is a log",
		}, workflow.DefaultMaxLogSize))
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}
		assert.NoError(t, workflow.AddLog(db, j, &sdk.Log{
			Val: "This is another log",
		}, workflow.DefaultMaxLogSize))
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		//TestUpdateNodeJobRunStatus
		_, err = workflow.UpdateNodeJobRunStatus(context.TODO(), func() *gorp.DbMap { return db }, db, cache, proj, j, sdk.StatusSuccess)
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		logs, err := workflow.LoadLogs(db, j.ID)
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}
		assert.NotEmpty(t, logs)

		tx.Commit()
	}

	filter = workflow.NewQueueFilter()
	filter.Rights = sdk.PermissionReadExecute
	jobs, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
	test.NoError(t, err)
	assert.Equal(t, 1, len(jobs))

	if len(jobs) == 1 {
		assert.Equal(t, "Waiting", jobs[0].Status)
		assert.Equal(t, "job20", jobs[0].Job.Job.Action.Name)

		// test since / until
		t.Logf("##### jobs[0].Queued : %+v\n", jobs[0].Queued)
		since := jobs[0].Queued

		t0 := since.Add(-2 * time.Minute)
		t1 := since.Add(-1 * time.Minute)
		filter := workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.Since = &t0
		filter.Until = &t1
		jobsSince, err := workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
		for _, job := range jobsSince {
			if jobs[0].ID == job.ID {
				assert.Fail(t, " this job should not be in queue since/until")
			}
		}

		filter = workflow.NewQueueFilter()
		filter.Rights = sdk.PermissionReadExecute
		filter.Since = &t0
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
		var found bool
		for _, job := range jobsSince {
			if jobs[0].ID == job.ID {
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
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
		for _, job := range jobsSince {
			if jobs[0].ID == job.ID {
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
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
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
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
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
		jobsSince, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, db, cache, filter, sdk.GroupsToIDs(append(u.OldUserStruct.Groups, proj.ProjectGroups[0].Group)))
		test.NoError(t, err)
		// we don't want the job with the worker model "TestManualRun"
		for _, job := range jobsSince {
			if job.ModelType == sdk.Docker {
				assert.Fail(t, " this job should not be in queue with this model")
			}
		}
	}
}

func TestNoStage(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))
	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, u, nil)
	test.NoError(t, errS)

	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	test.NoError(t, err)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", 1, lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, err)

	assert.Equal(t, sdk.StatusSuccess, nodeRun.Status)
}

func TestNoJob(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))
	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errS := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, u, nil)
	test.NoError(t, errS)

	lastrun, err := workflow.LoadLastRun(db, proj.Key, "test_1", workflow.LoadRunOptions{})
	test.NoError(t, err)

	//TestLoadNodeRun
	nodeRun, err := workflow.LoadNodeRun(db, proj.Key, "test_1", 1, lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, err)

	assert.Equal(t, sdk.StatusSuccess, nodeRun.Status)
}
