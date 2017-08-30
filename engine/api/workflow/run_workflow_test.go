package workflow

import (
	"context"
	"sort"
	"testing"
	"time"

	dump "github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestManualRun1(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	test.NoError(t, Insert(db, &w, u))
	w1, err := Load(db, key, "test_1", u)
	test.NoError(t, err)

	_, err = ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	test.NoError(t, err)

	wr1, err := ManualRun(db, w1, &sdk.WorkflowNodeRunManual{User: *u})
	test.NoError(t, err)

	m1, _ := dump.ToMap(wr1)

	keys1 := []string{}
	for k := range m1 {
		keys1 = append(keys1, k)
	}

	sort.Strings(keys1)
	for _, k := range keys1 {
		t.Logf("%s: \t%s", k, m1[k])
	}

	c, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	Scheduler(c, func() *gorp.DbMap { return db })

	time.Sleep(2 * time.Second)

	lastrun, err := LoadLastRun(db, proj.Key, "test_1")
	test.NoError(t, err)

	assert.Equal(t, int64(2), lastrun.Number)

	//TestLoadNodeRun
	nodeRun, err := LoadNodeRun(db, proj.Key, "test_1", 2, lastrun.WorkflowNodeRuns[w1.RootID][0].ID)
	test.NoError(t, err)
	test.Equal(t, lastrun.WorkflowNodeRuns[w1.RootID][0], nodeRun)

	//TestLoadNodeJobRun
	jobs, err := LoadNodeJobRunQueue(db, []int64{proj.ProjectGroups[0].Group.ID}, nil)
	test.NoError(t, err)

	//Print lastrun
	m, _ := dump.ToMap(jobs)
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s: \t%s", k, m[k])
	}
	test.Equal(t, 2, len(jobs))

	//Print jobs
	m, _ = dump.ToMap(jobs)
	keys = []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s: \t%s", k, m[k])
	}

	//TestprocessWorkflowRun
	wr2, err := ManualRunFromNode(db, w1, 2, &sdk.WorkflowNodeRunManual{User: *u}, w1.RootID)
	test.NoError(t, err)
	assert.NotNil(t, wr2)

	m, _ = dump.ToMap(wr2)
	keys = []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("- %s: \t%s", k, m[k])
	}

	//TestLoadRuns
	runs, offset, limit, count, err := LoadRuns(db, proj.Key, w1.Name, 0, 50)
	test.NoError(t, err)
	assert.Equal(t, 0, offset)
	assert.Equal(t, 50, limit)
	assert.Equal(t, 2, count)
	assert.Len(t, runs, 2)

	//TestLoadRunByID
	_, err = LoadRunByIDAndProjectKey(db, proj.Key, wr2.ID)
	test.NoError(t, err)

}

func TestManualRun2(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)

	test.NoError(t, project.AddKeyPair(db, proj, "key", u))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	test.NoError(t, Insert(db, &w, u))
	w1, err := Load(db, key, "test_1", u)
	test.NoError(t, err)

	_, err = ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	test.NoError(t, err)

	_, err = ManualRun(db, w1, &sdk.WorkflowNodeRunManual{User: *u})
	test.NoError(t, err)

	//TestprocessWorkflowRun
	_, err = ManualRunFromNode(db, w1, 1, &sdk.WorkflowNodeRunManual{User: *u}, w1.RootID)
	test.NoError(t, err)

	c, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	Scheduler(c, func() *gorp.DbMap { return db })

	time.Sleep(3 * time.Second)

	jobs, err := LoadNodeJobRunQueue(db, []int64{proj.ProjectGroups[0].Group.ID}, nil)
	test.NoError(t, err)

	assert.Len(t, jobs, 3)
}

func TestManualRun3(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)

	test.NoError(t, project.AddKeyPair(db, proj, "key", u))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Name:    "job20",
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
					},
				},
			},
		},
	}

	test.NoError(t, Insert(db, &w, u))
	w1, err := Load(db, key, "test_1", u)
	test.NoError(t, err)

	ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	test.NoError(t, err)

	c, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	Scheduler(c, func() *gorp.DbMap { return db })

	time.Sleep(3 * time.Second)

	jobs, err := LoadNodeJobRunQueue(db, []int64{proj.ProjectGroups[0].Group.ID}, nil)
	test.NoError(t, err)

	for i := range jobs {
		j := &jobs[i]
		tx, _ := db.Begin()

		//BookNodeJobRun
		_, err = BookNodeJobRun(j.ID, &sdk.Hatchery{
			Name: "Hatchery",
			ID:   1,
		})
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		//AddSpawnInfosNodeJobRun
		j, err := AddSpawnInfosNodeJobRun(db, j.ID, []sdk.SpawnInfo{
			sdk.SpawnInfo{
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
		j, err = TakeNodeJobRun(db, j.ID, "model", "worker", "1", []sdk.SpawnInfo{
			sdk.SpawnInfo{
				APITime:    time.Now(),
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{
					ID: sdk.MsgSpawnInfoJobTaken.ID,
				},
			},
		})

		//Load workflow node run
		nodeRun, err := LoadNodeRunByID(db, j.WorkflowNodeRunID)
		if err != nil {
			t.Fatal(err)
		}

		//Load workflow run
		workflowRun, err := LoadRunByID(db, nodeRun.WorkflowRunID)
		if err != nil {
			t.Fatal(err)
		}

		//TestLoadNodeJobRunSecrets
		secrets, err := LoadNodeJobRunSecrets(db, j, nodeRun, workflowRun)
		assert.NoError(t, err)
		assert.Len(t, secrets, 1)

		//TestAddLog
		assert.NoError(t, AddLog(db, j, &sdk.Log{
			Val: "This is a log",
		}))
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}
		assert.NoError(t, AddLog(db, j, &sdk.Log{
			Val: "This is another log",
		}))
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		//TestUpdateNodeJobRunStatus
		assert.NoError(t, UpdateNodeJobRunStatus(db, j, sdk.StatusSuccess))
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}

		logs, err := LoadLogs(db, j.ID)
		assert.NoError(t, err)
		if t.Failed() {
			tx.Rollback()
			t.FailNow()
		}
		assert.NotEmpty(t, logs)

		tx.Commit()
	}

	c, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	Scheduler(c, func() *gorp.DbMap { return db })

	time.Sleep(2 * time.Second)

	jobs, err = LoadNodeJobRunQueue(db, []int64{proj.ProjectGroups[0].Group.ID}, nil)
	test.NoError(t, err)
	assert.Equal(t, 1, len(jobs))

	if len(jobs) == 1 {
		assert.Equal(t, "Waiting", jobs[0].Status)
		assert.Equal(t, "job20", jobs[0].Job.Job.Action.Name)
	}
}
