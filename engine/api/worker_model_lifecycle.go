package api

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// eolDateFormat is the day granularity used to render an end of life date to end users.
const eolDateFormat = "2006-01-02"

// manageWorkerModelsLifecycle disables the deprecated worker models that reached their end of life
// date, and fails the queued jobs that no hatchery can ever start because they require a disabled
// worker model.
func (api *API) manageWorkerModelsLifecycle(ctx context.Context, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting manageWorkerModelsLifecycle: %v", ctx.Err())
			}
			return
		case <-ticker.C:
			if err := api.disableExpiredWorkerModels(ctx); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
			if err := api.failQueuedJobsForDisabledModels(ctx); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		}
	}
}

// disableExpiredWorkerModels disables the deprecated worker models whose end of life date is in the
// past. Once disabled a model is no longer served to the hatcheries, so it cannot be used anymore.
func (api *API) disableExpiredWorkerModels(ctx context.Context) error {
	now := time.Now()

	models, err := workermodel.LoadAllPastEOL(ctx, api.mustDB(), now, workermodel.LoadOptions.WithGroup)
	if err != nil {
		return err
	}

	for i := range models {
		disabled, err := workermodel.DisableExpired(ctx, api.mustDB(), models[i].ID, now)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		if disabled {
			log.Info(ctx, "worker model %s disabled: end of life date reached", models[i].Path())
		}
	}

	return nil
}

// failQueuedJobsForDisabledModels fails the queued jobs that explicitly require a disabled worker
// model: no hatchery is served such a model anymore, so those jobs would otherwise sit in the queue
// until stopRunsBlocked force stops their whole run, a day later and without any reason given.
//
// This only concerns the jobs that were already queued when the model got disabled. A run started
// afterwards never reaches the queue: processNodeJobRunRequirements already rejects a job whose
// model requirement points to a disabled model, with an explicit error.
func (api *API) failQueuedJobsForDisabledModels(ctx context.Context) error {
	// All the models are needed, not only the disabled ones: a requirement may spell a bare model
	// name, which can be matched by several models at once. If any enabled model matches, a hatchery
	// can still run the job and it must be left alone.
	models, err := workermodel.LoadAll(ctx, api.mustDB(), nil, workermodel.LoadOptions.WithGroup)
	if err != nil {
		return err
	}
	var enabled, disabled sdk.Models
	for i := range models {
		if models[i].Disabled {
			disabled = append(disabled, models[i])
		} else {
			enabled = append(enabled, models[i])
		}
	}
	if len(disabled) == 0 {
		return nil
	}

	jobs, err := workflow.LoadQueuedNodeJobRunWithModelRequirement(ctx, api.mustDB(), api.Cache)
	if err != nil {
		return err
	}

	for i := range jobs {
		model := findDisabledModelBlockingJob(jobs[i], enabled, disabled)
		if model == nil {
			continue
		}
		if err := api.failJobForDisabledModel(ctx, jobs[i], *model); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		log.Info(ctx, "job %d set to fail: worker model %s is disabled", jobs[i].ID, model.Path())
	}

	return nil
}

// findDisabledModelBlockingJob returns the disabled worker model that prevents the given queued job
// from ever starting, or nil when the job can still run.
func findDisabledModelBlockingJob(job sdk.WorkflowNodeJobRun, enabled, disabled sdk.Models) *sdk.Model {
	for _, r := range job.Job.Action.Requirements {
		if r.Type != sdk.ModelRequirement {
			continue
		}

		// an enabled model matching the requirement is enough for a hatchery to run the job
		for i := range enabled {
			if enabled[i].MatchesRequirementValue(r.Value) {
				return nil
			}
		}

		for i := range disabled {
			if disabled[i].MatchesRequirementValue(r.Value) {
				return &disabled[i]
			}
		}
	}

	// no model matches the requirement at all: the model was renamed or removed, out of scope here
	return nil
}

// failJobForDisabledModel sets a queued job to fail and explains why in its spawn infos.
//
// The job was never taken by a worker, so its stage is still waiting. In that state
// executeNodeRun leaves the node run untouched, because it only reconciles a stage that is
// already building (engine/api/workflow/execute_node_run.go). The failure would therefore stop at
// the job and leave the run building forever, so it is propagated to the stage, the node run and
// the workflow run by hand, the same way stopRunsBlocked does it.
func (api *API) failJobForDisabledModel(ctx context.Context, job sdk.WorkflowNodeJobRun, model sdk.Model) error {
	var msg sdk.SpawnMsg
	if model.EOL != nil {
		msg = sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoJobFailedCauseByModelEOL.ID,
			Args: []interface{}{model.Path(), model.EOL.Format(eolDateFormat)},
		}
	} else {
		msg = sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoJobFailedCauseByModelDisabled.ID,
			Args: []interface{}{model.Path()},
		}
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// the spawn info is saved first, like stopWorkflowNodeJobRun does: UpdateNodeJobRunStatus
	// reloads the job spawn infos to sync them into the node run stages
	if err := workflow.AddSpawnInfosNodeJobRun(tx, job.WorkflowNodeRunID, job.ID, []sdk.SpawnInfo{{Message: msg}}); err != nil {
		return sdk.WrapError(err, "unable to save spawn info on node job run %d", job.ID)
	}

	if _, err := workflow.UpdateNodeJobRunStatus(ctx, tx, api.Cache, sdk.Project{}, &job, sdk.StatusFail); err != nil {
		return sdk.WrapError(err, "unable to set node job run %d to fail", job.ID)
	}

	nodeRun, err := workflow.LoadAndLockNodeRunByID(ctx, tx, job.WorkflowNodeRunID)
	if err != nil {
		return sdk.WrapError(err, "unable to load node run %d", job.WorkflowNodeRunID)
	}
	if i := nodeRun.GetStageIndex(&job); i >= 0 && i < len(nodeRun.Stages) {
		nodeRun.Stages[i].Status = sdk.StatusFail
	}
	nodeRun.Status = sdk.StatusFail
	nodeRun.Done = time.Now()
	if err := workflow.UpdateNodeRunStatusAndStage(tx, nodeRun); err != nil {
		return sdk.WrapError(err, "unable to update node run %d", nodeRun.ID)
	}

	wr, err := workflow.LoadRunByID(ctx, tx, nodeRun.WorkflowRunID, workflow.LoadRunOptions{})
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run %d", nodeRun.WorkflowRunID)
	}
	if _, err := workflow.ResyncWorkflowRunStatus(ctx, tx, wr); err != nil {
		return sdk.WrapError(err, "unable to resync workflow run %d", wr.ID)
	}

	return sdk.WithStack(tx.Commit())
}
