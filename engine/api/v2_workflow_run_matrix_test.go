package api

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/entity"
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

// matrixTestEnv is a project where matrix jobs can be enqueued
type matrixTestEnv struct {
	proj      *sdk.Project
	vcsServer *sdk.VCSProject
	repo      *sdk.ProjectRepository
	initiator *sdk.V2Initiator
}

func newMatrixTestEnv(t *testing.T, api *API, db *test.FakeTransaction) matrixTestEnv {
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
	return matrixTestEnv{proj: proj, vcsServer: vcsServer, repo: repo, initiator: &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}}
}

// insertMatrixRun inserts a building run of "myworkflow" with one matrix job "test" of nbPermutations
func insertMatrixRun(t *testing.T, db *test.FakeTransaction, env matrixTestEnv, runNumber int64, refName string, nbPermutations int, job sdk.V2Job, concurrencies []sdk.WorkflowConcurrency) sdk.V2WorkflowRun {
	t.Helper()
	values := make([]string, 0, nbPermutations)
	for i := 0; i < nbPermutations; i++ {
		values = append(values, strconv.Itoa(i))
	}
	if job.Strategy == nil {
		job.Strategy = &sdk.V2JobStrategy{}
	}
	job.Strategy.Matrix = map[string]interface{}{"n": values}
	job.Steps = []sdk.ActionStep{{Run: "echo test"}}

	wr := sdk.V2WorkflowRun{
		ProjectKey:   env.proj.Key,
		VCSServerID:  env.vcsServer.ID,
		VCSServer:    env.vcsServer.Name,
		RepositoryID: env.repo.ID,
		Repository:   env.repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "abcdef",
		WorkflowRef:  "refs/heads/" + refName,
		RunNumber:    runNumber,
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator:    env.initiator,
		Contexts:     sdk.WorkflowRunContext{Git: sdk.GitContext{Ref: "refs/heads/" + refName, RefName: refName}},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name:          "myworkflow",
			Concurrencies: concurrencies,
			Jobs:          map[string]sdk.V2Job{"test": job},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))
	return wr
}

// triggerRun runs one engine pass on the run
func triggerRun(t *testing.T, api *API, env matrixTestEnv, wr sdk.V2WorkflowRun) {
	t.Helper()
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *env.initiator}))
}

// endOnePermutation ends one waiting permutation of the run with a success, as a worker would
func endOnePermutation(t *testing.T, db *test.FakeTransaction, wr sdk.V2WorkflowRun) {
	t.Helper()
	runJobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	for i := range runJobs {
		if runJobs[i].Status == sdk.V2WorkflowRunJobStatusWaiting {
			now := time.Now()
			runJobs[i].Status = sdk.V2WorkflowRunJobStatusSuccess
			runJobs[i].Ended = &now
			require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &runJobs[i]))
			return
		}
	}
	t.Fatal("no waiting permutation to end")
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
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{}, nil)

	// First pass: the rule is written in the run definition, no permutation is created yet
	triggerRun(t, api, env, wr)
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
	triggerRun(t, api, env, wr)
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
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 3, sdk.V2Job{}, nil)

	triggerRun(t, api, env, wr)
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
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{Concurrency: rule.Name}, []sdk.WorkflowConcurrency{rule})

	triggerRun(t, api, env, wr)
	counts := countJobStatuses(t, api, wr)
	assert.Equal(t, 5, counts[sdk.V2WorkflowRunJobStatusWaiting])
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), api.mustDB(), wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, "mycc", wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
}

// A permutation ending frees a slot for the oldest blocked one
func TestMatrixImplicitConcurrency_UnlockOnPermutationEnd(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{}, nil)
	triggerRun(t, api, env, wr)
	triggerRun(t, api, env, wr)
	require.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 2}, countJobStatuses(t, api, wr))

	endOnePermutation(t, db, wr)
	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusSuccess: 1, sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, wr))
}

// Two runs on the same branch share the rule: the newest one waits behind the oldest one
func TestMatrixImplicitConcurrency_RunsOnSameBranchQueueUp(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	env := newMatrixTestEnv(t, api, db)
	first := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{}, nil)
	triggerRun(t, api, env, first)
	triggerRun(t, api, env, first)
	require.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 2}, countJobStatuses(t, api, first))

	second := insertMatrixRun(t, db, env, 2, "master", 5, sdk.V2Job{}, nil)
	triggerRun(t, api, env, second)
	triggerRun(t, api, env, second)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusBlocked: 5}, countJobStatuses(t, api, second), "the slots are held by the first run")

	// A freed slot goes to the first run, which arrived first
	endOnePermutation(t, db, first)
	triggerRun(t, api, env, second)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusBlocked: 5}, countJobStatuses(t, api, second), "the second run must wait for the first one")
	triggerRun(t, api, env, first)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusSuccess: 1, sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, first))
}

// Two branches have their own rule and don't wait for each other
func TestMatrixImplicitConcurrency_BranchesAreIndependent(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	env := newMatrixTestEnv(t, api, db)
	master := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{}, nil)
	triggerRun(t, api, env, master)
	triggerRun(t, api, env, master)
	require.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 2}, countJobStatuses(t, api, master))

	feature := insertMatrixRun(t, db, env, 2, "feature", 5, sdk.V2Job{}, nil)
	triggerRun(t, api, env, feature)
	triggerRun(t, api, env, feature)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 3, sdk.V2WorkflowRunJobStatusBlocked: 2}, countJobStatuses(t, api, feature))
	featureDB, err := workflow_v2.LoadRunByID(context.TODO(), db, feature.ID)
	require.NoError(t, err)
	require.Len(t, featureDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, "matrix/test/feature", featureDB.WorkflowData.Workflow.Concurrencies[0].Name)
}

// Without configuration, a matrix job runs up to 20 permutations in parallel and the rule applies above
func TestMatrixImplicitConcurrency_DefaultConfig(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.workflowRunTriggerChan = make(chan sdk.V2WorkflowRunEnqueue, 10)
	require.Equal(t, int64(defaultMatrixParallelism), api.matrixParallelism().Default)
	env := newMatrixTestEnv(t, api, db)
	nb := int(defaultMatrixParallelism)

	atDefault := insertMatrixRun(t, db, env, 1, "master", nb, sdk.V2Job{}, nil)
	triggerRun(t, api, env, atDefault)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: nb}, countJobStatuses(t, api, atDefault))
	atDefaultDB, err := workflow_v2.LoadRunByID(context.TODO(), db, atDefault.ID)
	require.NoError(t, err)
	assert.Empty(t, atDefaultDB.WorkflowData.Workflow.Concurrencies)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, atDefault.ID)
	require.NoError(t, err)
	assert.Empty(t, infos)

	aboveDefault := insertMatrixRun(t, db, env, 2, "feature", nb+1, sdk.V2Job{}, nil)
	triggerRun(t, api, env, aboveDefault)
	triggerRun(t, api, env, aboveDefault)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: nb, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, aboveDefault))
	aboveDefaultDB, err := workflow_v2.LoadRunByID(context.TODO(), db, aboveDefault.ID)
	require.NoError(t, err)
	require.Len(t, aboveDefaultDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, sdk.WorkflowConcurrency{Name: "matrix/test/feature", Pool: defaultMatrixParallelism, Order: sdk.ConcurrencyOrderOldestFirst}, aboveDefaultDB.WorkflowData.Workflow.Concurrencies[0])
	infos, err = workflow_v2.LoadRunInfosByRunID(context.TODO(), db, aboveDefault.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "Job test: 21 permutations, CDS limits it to 20 jobs in parallel", infos[0].Message)
}

// max-parallel replaces the default parallelism as the threshold and the pool of the implicit rule
func TestMatrixMaxParallel_AbovePermutations(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	api.Config.WorkflowV2.MatrixMaxParallelism = 6
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{Strategy: &sdk.V2JobStrategy{MaxParallel: 4}}, nil)

	triggerRun(t, api, env, wr)
	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 4, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, wr))
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, sdk.WorkflowConcurrency{Name: "matrix/test/master", Pool: 4, Order: sdk.ConcurrencyOrderOldestFirst}, wrDB.WorkflowData.Workflow.Concurrencies[0])
	assert.Equal(t, "matrix/test/master", wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "Job test: 5 permutations, limited to 4 jobs in parallel (max-parallel)", infos[0].Message)
	assert.Equal(t, sdk.WorkflowRunInfoLevelInfo, infos[0].Level)
}

// A max-parallel covering the permutations gives no rule, even above the default parallelism
func TestMatrixMaxParallel_CoversPermutations(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{Strategy: &sdk.V2JobStrategy{MaxParallel: 5}}, nil)

	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 5}, countJobStatuses(t, api, wr))
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	assert.Empty(t, wrDB.WorkflowData.Workflow.Concurrencies)
	assert.Empty(t, wrDB.WorkflowData.Workflow.Jobs["test"].Concurrency)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

// A max-parallel above the maximum parallelism, as a template may carry, is bounded to it
func TestMatrixMaxParallel_AboveMax(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	api.Config.WorkflowV2.MatrixMaxParallelism = 4
	env := newMatrixTestEnv(t, api, db)
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{Strategy: &sdk.V2JobStrategy{MaxParallel: 100}}, nil)

	triggerRun(t, api, env, wr)
	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 4, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, wr))
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.WorkflowData.Workflow.Concurrencies, 1)
	assert.Equal(t, int64(4), wrDB.WorkflowData.Workflow.Concurrencies[0].Pool)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "Job test: 5 permutations, max-parallel 100 is above the maximum, CDS limits it to 4 jobs in parallel", infos[0].Message)
	assert.Equal(t, sdk.WorkflowRunInfoLevelWarning, infos[0].Level)
}

// With a user concurrency, the permutations are created with its pool bounded to the maximum parallelism
func TestMatrixMaxParallelism_AboveMax(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	api.Config.WorkflowV2.MatrixMaxParallelism = 4
	env := newMatrixTestEnv(t, api, db)
	rule := sdk.WorkflowConcurrency{Name: "mycc", Pool: 10, Order: sdk.ConcurrencyOrderOldestFirst}
	wr := insertMatrixRun(t, db, env, 1, "master", 5, sdk.V2Job{Concurrency: rule.Name}, []sdk.WorkflowConcurrency{rule})

	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 4, sdk.V2WorkflowRunJobStatusBlocked: 1}, countJobStatuses(t, api, wr))
	runJobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	for _, rj := range runJobs {
		require.NotNil(t, rj.Concurrency)
		assert.Equal(t, int64(4), rj.Concurrency.Pool)
		assert.Equal(t, "mycc", rj.Concurrency.Name)
	}
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(10), wrDB.WorkflowData.Workflow.Concurrencies[0].Pool, "the rule itself is not modified")
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, `Job test: 5 permutations, concurrency "mycc" pool limited to 4 jobs in parallel`, infos[0].Message)

	// The limit holds on the next passes and is not reported again
	endOnePermutation(t, db, wr)
	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusSuccess: 1, sdk.V2WorkflowRunJobStatusWaiting: 4}, countJobStatuses(t, api, wr))
	infos, err = workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
}

// A user concurrency within the maximum parallelism is used as is
func TestMatrixMaxParallelism_WithinMax(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.WorkflowV2.MatrixDefaultParallelism = 3
	api.Config.WorkflowV2.MatrixMaxParallelism = 4
	env := newMatrixTestEnv(t, api, db)

	// As many permutations as the maximum: no cap
	rule := sdk.WorkflowConcurrency{Name: "mycc", Pool: 10, Order: sdk.ConcurrencyOrderOldestFirst}
	wr := insertMatrixRun(t, db, env, 1, "master", 4, sdk.V2Job{Concurrency: rule.Name}, []sdk.WorkflowConcurrency{rule})
	triggerRun(t, api, env, wr)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 4}, countJobStatuses(t, api, wr))
	runJobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	assert.Equal(t, int64(10), runJobs[0].Concurrency.Pool)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	assert.Empty(t, infos)

	// A pool below the maximum applies as is, whatever the number of permutations
	small := sdk.WorkflowConcurrency{Name: "small", Pool: 2, Order: sdk.ConcurrencyOrderOldestFirst}
	wr2 := insertMatrixRun(t, db, env, 2, "master", 5, sdk.V2Job{Concurrency: small.Name}, []sdk.WorkflowConcurrency{small})
	triggerRun(t, api, env, wr2)
	assert.Equal(t, map[sdk.V2WorkflowRunJobStatus]int{sdk.V2WorkflowRunJobStatusWaiting: 2, sdk.V2WorkflowRunJobStatusBlocked: 3}, countJobStatuses(t, api, wr2))
	infos, err = workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr2.ID)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

// The analysis bounds max-parallel by the configured maximum parallelism
func TestLint_MaxParallelBound(t *testing.T) {
	api, db, _ := newTestAPI(t)
	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *admin)
	initiator := sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()}
	require.NoError(t, entity.Insert(context.TODO(), db, &sdk.Entity{ProjectKey: proj.Key, ProjectRepositoryID: repo.ID, Type: sdk.EntityTypeWorkerModel, Name: "mymodel", FilePath: ".cds/worker-models/mymodel.yml", Commit: "abcdef", Ref: "refs/heads/master", Data: "name: mymodel\ntype: docker\nosarch: linux-amd64\nspec:\n  image: debian:12", Initiator: &initiator}))
	ef, err := NewEntityFinder(context.TODO(), db.DbMap, proj.Key, "refs/heads/master", "abcdef", *repo, *vcsServer, initiator, "")
	require.NoError(t, err)

	workflow := func(maxParallel int64) sdk.V2Workflow {
		return sdk.V2Workflow{
			Name: "myworkflow",
			Jobs: map[string]sdk.V2Job{
				"test": {
					Strategy: &sdk.V2JobStrategy{Matrix: map[string]interface{}{"os": []interface{}{"linux", "windows"}}, MaxParallel: maxParallel},
					RunsOn:   sdk.V2JobRunsOn{Model: ".cds/worker-models/mymodel.yml"},
					Steps:    []sdk.ActionStep{{Run: "echo"}},
				},
			},
		}
	}
	require.Empty(t, Lint(context.TODO(), db.DbMap, api.Cache, workflow(4), ef, proj.Key, nil, 4))
	errs := Lint(context.TODO(), db.DbMap, api.Cache, workflow(5), ef, proj.Key, nil, 4)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "max-parallel 5 is above the maximum of 4")
	require.Empty(t, Lint(context.TODO(), db.DbMap, api.Cache, workflow(5), ef, proj.Key, nil, 0), "no bound when unknown")
}
