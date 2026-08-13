package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// insertLifecycleModel inserts a worker model in the shared.infra group so that a bare model name in
// a job requirement resolves to it.
func insertLifecycleModel(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string, groupID int64, deprecated, disabled bool, eol *time.Time) *sdk.Model {
	m := sdk.Model{
		Name:         name,
		Type:         sdk.Docker,
		GroupID:      groupID,
		IsDeprecated: deprecated,
		Disabled:     disabled,
		EOL:          eol,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker",
		},
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m))
	return &m
}

// runWorkflowRequiringModel queues a workflow job holding a model requirement on the given value.
func runWorkflowRequiringModel(t *testing.T, api *API, router *Router, requirementValue string) testRunWorkflowCtx {
	ctx := testRunWorkflow(t, api, router, func(tt *testing.T, tx gorpmapper.SqlExecutorWithTx, pip *sdk.Pipeline, app *sdk.Application) {
		pip.Stages[0].Jobs[0].Action.Requirements = []sdk.Requirement{{
			Name:  requirementValue,
			Type:  sdk.ModelRequirement,
			Value: requirementValue,
		}}
		require.NoError(tt, pipeline.UpdateJob(context.TODO(), tx, &pip.Stages[0].Jobs[0]))
	})
	require.NotNil(t, ctx.job)
	require.Equal(t, sdk.StatusWaiting, ctx.job.Status)
	return ctx
}

// requireRunFailed asserts that the failure reached every level. Checking the job alone is not
// enough: a job can be set to fail while its stage, node run and workflow run stay building, which
// leaves the run hanging exactly like before the feature.
func requireRunFailed(t *testing.T, api *API, wctx testRunWorkflowCtx, expectedSpawnInfoID string) {
	assert.Equal(t, sdk.StatusFail, reloadJobStatus(t, api, wctx.job), "the job must fail")
	assert.Contains(t, jobSpawnInfoIDs(t, api, wctx.job), expectedSpawnInfoID, "the job must say why it failed")

	nodeRun, err := workflow.LoadNodeRunByID(context.TODO(), api.mustDB(), wctx.job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	assert.Equal(t, sdk.StatusFail, nodeRun.Status, "the failure must be propagated to the node run")
	require.NotEmpty(t, nodeRun.Stages)
	assert.Equal(t, sdk.StatusFail, nodeRun.Stages[0].Status, "the failure must be propagated to the stage")

	run, err := workflow.LoadRunByID(context.TODO(), api.mustDB(), wctx.run.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	assert.Equal(t, sdk.StatusFail, run.Status, "the failure must be propagated to the workflow run")
}

func jobSpawnInfoIDs(t *testing.T, api *API, job *sdk.WorkflowNodeJobRun) []string {
	infos, err := workflow.LoadNodeRunJobInfo(context.TODO(), api.mustDB(), job.WorkflowNodeRunID, job.ID)
	require.NoError(t, err)
	ids := make([]string, 0, len(infos))
	for _, i := range infos {
		ids = append(ids, i.Message.ID)
	}
	return ids
}

func reloadJobStatus(t *testing.T, api *API, job *sdk.WorkflowNodeJobRun) string {
	reloaded, err := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, job.ID)
	require.NoError(t, err)
	return reloaded.Status
}

// Test_manageWorkerModelsLifecycle_EOL is the end to end test of the feature: once the end of life
// date of a deprecated model is reached, the model is disabled and the jobs that require it are
// failed with an explicit reason. Without the second pass those jobs would sit in the queue until
// stopRunsBlocked force stops their run a day later, without any reason given.
func Test_manageWorkerModelsLifecycle_EOL(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	past := time.Now().Add(-24 * time.Hour)
	name := sdk.RandomString(10)
	model := insertLifecycleModel(t, db, name, group.SharedInfraGroup.ID, true, false, &past)

	wctx := runWorkflowRequiringModel(t, api, router, name)

	require.NoError(t, api.disableExpiredWorkerModels(context.TODO()))

	reloadedModel, err := workermodel.LoadByID(context.TODO(), api.mustDB(), model.ID)
	require.NoError(t, err)
	require.True(t, reloadedModel.Disabled, "the model must be disabled once its end of life date is reached")

	require.NoError(t, api.failQueuedJobsForDisabledModels(context.TODO()))

	requireRunFailed(t, api, wctx, sdk.MsgSpawnInfoJobFailedCauseByModelEOL.ID)
}

// Test_failQueuedJobsForDisabledModels_ManualDisable checks the same protection when an administrator
// disables a model by hand.
//
// The model has to be disabled after the job is queued: processNodeJobRunRequirements already refuses
// to queue a job whose model requirement points to an already disabled model
// (engine/api/workflow/process_requirements.go). The gap this routine closes is only about the jobs
// that were already in the queue when the model went away.
func Test_failQueuedJobsForDisabledModels_ManualDisable(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	name := sdk.RandomString(10)
	model := insertLifecycleModel(t, db, name, group.SharedInfraGroup.ID, false, false, nil)

	wctx := runWorkflowRequiringModel(t, api, router, name)

	_, err := api.mustDB().Exec("UPDATE worker_model SET disabled = true WHERE id = $1", model.ID)
	require.NoError(t, err)

	require.NoError(t, api.failQueuedJobsForDisabledModels(context.TODO()))

	requireRunFailed(t, api, wctx, sdk.MsgSpawnInfoJobFailedCauseByModelDisabled.ID)
}

// Test_failQueuedJobsForDisabledModels_BareNameMatchedByAnEnabledModel is the regression guard on the
// subtle part of the feature. A model requirement may spell a bare name without its group, kept for
// backward compatibility with old runs, so the same value can match several models at once. Looking
// only at the disabled models would fail a job that an enabled model can still run.
func Test_failQueuedJobsForDisabledModels_BareNameMatchedByAnEnabledModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	name := sdk.RandomString(10)
	g := &sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(context.TODO(), db, g))

	// same name, one disabled in a group and one still enabled in shared.infra
	insertLifecycleModel(t, db, name, g.ID, true, true, nil)
	insertLifecycleModel(t, db, name, group.SharedInfraGroup.ID, false, false, nil)

	wctx := runWorkflowRequiringModel(t, api, router, name)

	require.NoError(t, api.failQueuedJobsForDisabledModels(context.TODO()))

	assert.Equal(t, sdk.StatusWaiting, reloadJobStatus(t, api, wctx.job),
		"an enabled model matches the requirement, a hatchery can still run this job")
	assert.NotContains(t, jobSpawnInfoIDs(t, api, wctx.job), sdk.MsgSpawnInfoJobFailedCauseByModelDisabled.ID)
}

// Test_failQueuedJobsForDisabledModels_HealthyModel checks the routine is a no-op on a healthy queue.
func Test_failQueuedJobsForDisabledModels_HealthyModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	name := sdk.RandomString(10)
	insertLifecycleModel(t, db, name, group.SharedInfraGroup.ID, false, false, nil)
	// a disabled model unrelated to the job, so the routine does not early return
	insertLifecycleModel(t, db, sdk.RandomString(10), group.SharedInfraGroup.ID, false, true, nil)

	wctx := runWorkflowRequiringModel(t, api, router, name)

	require.NoError(t, api.failQueuedJobsForDisabledModels(context.TODO()))

	assert.Equal(t, sdk.StatusWaiting, reloadJobStatus(t, api, wctx.job))
}
