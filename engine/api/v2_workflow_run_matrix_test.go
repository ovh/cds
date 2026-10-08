package api

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestMatrixParallelism(t *testing.T) {
	cases := []struct {
		name                         string
		defaultValue, maxValue       int64
		expectedDefault, expectedMax int64
	}{
		{"unset", 0, 0, 20, 40},
		{"configured", 10, 30, 10, 30},
		{"negative", -1, -1, 20, 40},
		{"max lower than default", 30, 10, 30, 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			api := &API{}
			api.Config.WorkflowV2.MatrixDefaultParallelism = c.defaultValue
			api.Config.WorkflowV2.MatrixMaxParallelism = c.maxValue
			require.Equal(t, matrixParallelism{Default: c.expectedDefault, Max: c.expectedMax}, api.matrixParallelism())
		})
	}
}

func insertMatrixRun(t *testing.T, api *API, db *test.FakeTransaction, nbPermutations int, job sdk.V2Job, concurrencies []sdk.WorkflowConcurrency) (sdk.V2WorkflowRun, *sdk.V2Initiator) {
	t.Helper()
	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name
	require.NoError(t, rbac.Insert(context.TODO(), db, &sdk.RBAC{
		Name:           sdk.RandomString(10),
		RegionProjects: []sdk.RBACRegionProject{{RegionID: reg.ID, RBACProjectKeys: []string{proj.Key}, Role: sdk.RegionRoleExecute}},
	}))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}

	values := make([]string, 0, nbPermutations)
	for i := 0; i < nbPermutations; i++ {
		values = append(values, strconv.Itoa(i))
	}
	job.Strategy = &sdk.V2JobStrategy{Matrix: map[string]interface{}{"n": values}}
	job.Steps = []sdk.ActionStep{{Run: "echo test"}}

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "abcdef",
		WorkflowRef:  "refs/heads/master",
		RunNumber:    1,
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator:    initiator,
		Contexts:     sdk.WorkflowRunContext{Git: sdk.GitContext{Ref: "refs/heads/master", RefName: "master"}},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name:          "myworkflow",
			Concurrencies: concurrencies,
			Jobs:          map[string]sdk.V2Job{"test": job},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))
	return wr, initiator
}

func countJobStatuses(t *testing.T, api *API, wr sdk.V2WorkflowRun) map[sdk.V2WorkflowRunJobStatus]int {
	t.Helper()
	runJobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), api.mustDB(), wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	counts := make(map[sdk.V2WorkflowRunJobStatus]int)
	for _, rj := range runJobs {
		counts[rj.Status]++
	}
	return counts
}

// A matrix job with more permutations than the default parallelism gets an implicit concurrency
func TestMatrixImplicitConcurrency_AboveDefault(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	api.workflowRunTriggerChan = make(chan sdk.V2WorkflowRunEnqueue, 10)
	wr, initiator := insertMatrixRun(t, api, db, 5, sdk.V2Job{}, nil)

	// First pass: the rule is written in the run definition, no permutation is created yet
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))
	require.Empty(t, countJobStatuses(t, api, wr))
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, sdk.WorkflowConcurrency{Name: "matrix/test/master", Pool: 3, Order: sdk.ConcurrencyOrderOldestFirst}, wrDB.WorkflowData.Workflow.Concurrencies[0])
	assert.Equal(t, "matrix/test/master", wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "Job test: 5 permutations, CDS limits it to 3 jobs in parallel", infos[0].Message)
	assert.Equal(t, sdk.WorkflowRunInfoLevelWarning, infos[0].Level)
	select {
	case e := <-api.workflowRunTriggerChan:
		require.Equal(t, wr.ID, e.RunID)
	default:
		t.Fatal("the run must be enqueued again to create the permutations")
	}

	// Second pass: the permutations are created under the rule
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))
	counts := countJobStatuses(t, api, wr)
	assert.Equal(t, 3, counts[sdk.V2WorkflowRunJobStatusWaiting])
	assert.Equal(t, 2, counts[sdk.V2WorkflowRunJobStatusBlocked])
	infos, err = workflow_v2.LoadRunInfosByRunID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1, "the limit is reported once")
}

// A matrix job within the default parallelism runs without any rule
func TestMatrixImplicitConcurrency_WithinDefault(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	wr, initiator := insertMatrixRun(t, api, db, 3, sdk.V2Job{}, nil)

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))
	counts := countJobStatuses(t, api, wr)
	assert.Equal(t, 3, counts[sdk.V2WorkflowRunJobStatusWaiting])
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	assert.Empty(t, wrDB.WorkflowData.Workflow.Concurrencies)
	assert.Empty(t, wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

// A matrix job with its own concurrency keeps it, the implicit rule is not added
func TestMatrixImplicitConcurrency_WithUserConcurrency(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	rule := sdk.WorkflowConcurrency{Name: "mycc", Pool: 10, Order: sdk.ConcurrencyOrderOldestFirst}
	wr, initiator := insertMatrixRun(t, api, db, 5, sdk.V2Job{Concurrency: rule.Name}, []sdk.WorkflowConcurrency{rule})

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))
	counts := countJobStatuses(t, api, wr)
	assert.Equal(t, 5, counts[sdk.V2WorkflowRunJobStatusWaiting])
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, "mycc", wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
}
