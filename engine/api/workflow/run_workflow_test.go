package workflow

import (
	"context"
	"sort"
	"testing"

	"time"

	"github.com/fsamin/go-dump"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestManualRun1(t *testing.T) {
	db := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip, u))
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

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, Insert(db, &w, u))
	w1, err := Load(db, key, "test_1", u)
	test.NoError(t, err)

	wr, err := ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	test.NoError(t, err)

	m, _ := dump.ToMap(wr)

	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s: \t%s", k, m[k])
	}

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
	Scheduler(c)
}
