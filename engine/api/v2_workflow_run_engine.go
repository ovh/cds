package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) V2WorkflowRunEngineChan(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		case wrEnqueue := <-api.workflowRunTriggerChan:
			if err := api.workflowRunV2Trigger(ctx, wrEnqueue); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		}
	}
}

func (api *API) V2WorkflowRunEngineDequeue(ctx context.Context) error {
	for {
		var wrEnqueue sdk.V2WorkflowRunEnqueue
		if err := api.Cache.DequeueWithContext(ctx, workflow_v2.WorkflowEngineKey, 250*time.Millisecond, &wrEnqueue); err != nil {
			log.Error(ctx, "V2WorkflowRunEngine > DequeueWithContext err: %v", err)
			continue
		}
		if err := api.workflowRunV2Trigger(ctx, wrEnqueue); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// TODO Manage stage
// TODO Manage job sub number
// TODO manage git context
// TODO manage vars context
func (api *API) workflowRunV2Trigger(ctx context.Context, wrEnqueue sdk.V2WorkflowRunEnqueue) error {
	_, next := telemetry.Span(ctx, "api.workflowRunV2Trigger.lock")
	lockKey := cache.Key("api:workflow:engine", wrEnqueue.RunID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug(ctx, "api.workflowRunV2Trigger> run %d is locked in cache", wrEnqueue.RunID)
		// re-enqueue workflow-run
		if err := api.Cache.Enqueue(workflow_v2.WorkflowEngineKey, wrEnqueue); err != nil {
			next()
			return err
		}
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	// Load run by id
	run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), wrEnqueue.RunID)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run %s", wrEnqueue.RunID)
	}

	if sdk.StatusIsTerminated(run.Status) && len(wrEnqueue.Jobs) == 0 {
		log.Debug(ctx, "workflow run already on a final state")
		return nil
	}

	u, err := user.LoadByID(ctx, api.mustDB(), wrEnqueue.UserID)
	if err != nil {
		return err
	}

	jobsToQueue, runMsgs, currentJobRunStatus, errRetrieveJobs := retrieveJobToQueue(ctx, api.mustDB(), run, wrEnqueue, u, api.Config.Workflow.JobDefaultRegion)
	log.Debug(ctx, "workflowRunV2Trigger> jobs to queue: %+v", jobsToQueue)

	tx, errTx := api.mustDB().Begin()
	if errTx != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if errRetrieveJobs != nil {
		for i := range runMsgs {
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsgs[i]); err != nil {
				return err
			}
			run.Status = sdk.StatusFail
			if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
		}
		return errRetrieveJobs
	}

	// Enqueue JOB
	for jobID, jobDef := range jobsToQueue {
		runJob := sdk.V2WorkflowRunJob{
			WorkflowRunID: run.ID,
			Status:        sdk.StatusWaiting,
			JobID:         jobID,
			Job:           jobDef,
			UserID:        wrEnqueue.UserID,
			Username:      u.Username,
		}
		if err := workflow_v2.InsertRunJob(ctx, tx, &runJob); err != nil {
			return err
		}
	}

	// Save Run message
	for i := range runMsgs {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsgs[i]); err != nil {
			return err
		}
	}

	// End workflow if there is no job to queue,  no running jobs and current status is not terminated
	if len(jobsToQueue) == 0 && sdk.StatusIsTerminated(currentJobRunStatus) && !sdk.StatusIsTerminated(run.Status) {
		run.Status = currentJobRunStatus

		if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
			return err
		}
	}

	return sdk.WithStack(tx.Commit())
}

// TODO manage re run
func retrieveJobToQueue(ctx context.Context, db *gorp.DbMap, run *sdk.V2WorkflowRun, wrEnqueue sdk.V2WorkflowRunEnqueue, u *sdk.AuthentifiedUser, defaultRegion string) (map[string]sdk.V2Job, []sdk.V2WorkflowRunInfo, string, error) {
	_, next := telemetry.Span(ctx, "retrieveJobToQueue")
	defer next()
	runInfos := make([]sdk.V2WorkflowRunInfo, 0)
	jobToQueue := make(map[string]sdk.V2Job)

	// Load run_jobs
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, db, run.ID)
	if err != nil {
		return nil, nil, "", sdk.WrapError(err, "unable to load workflow run jobs for run %s", wrEnqueue.RunID)
	}

	// If jobs has already been queue and wr not terminated and user want to trigger some job => error
	if !sdk.StatusIsTerminated(run.Status) && len(wrEnqueue.Jobs) > 0 && len(runJobs) > 0 {
		info := sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelWarning,
			Message:       "unable to start a job on a running workflow",
		}
		runInfos = append(runInfos, info)
		return nil, runInfos, "", nil
	}

	// Compute run context
	jobsContext := buildJobsContext(runJobs)

	// Select jobs to check ( all workflow or list of jobs from enqueue request )
	jobsToCheck := make(map[string]sdk.V2Job)
	if len(wrEnqueue.Jobs) == 0 {
		for jobID, jobDef := range run.WorkflowData.Workflow.Jobs {
			jobsToCheck[jobID] = jobDef
		}
	} else {
		for _, jobID := range wrEnqueue.Jobs {
			if jobDef, has := run.WorkflowData.Workflow.Jobs[jobID]; has {
				jobsToCheck[jobID] = jobDef
			}
		}
	}

	// Check jobs : Needs / Condition / User Right
	for jobID, jobDef := range jobsToCheck {

		// TODO manage re run
		if _, has := jobsContext[jobID]; has {
			log.Debug(ctx, "job %s: already executed, skip it", jobID)
			continue
		}

		// Check jobs needs
		requiredJob, ok := checkJobNeeds(jobsContext, jobDef)
		if !ok {
			// If not ok , and ask to run it => send message
			if len(wrEnqueue.Jobs) > 0 {
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelWarning,
					Message:       fmt.Sprintf("job %s: missing some required job: %s", jobID, requiredJob),
				})
			}
			continue
		}

		hasRight, err := checkUserRight(ctx, db, jobDef, *u, defaultRegion)
		if err != nil {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: unable to check right for user %s: %v", jobID, u.Username, err),
			})
			return nil, runInfos, sdk.StatusFail, err
		}
		if !hasRight {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelWarning,
				Message:       fmt.Sprintf("job %s: user %s does not have enough right", jobID, u.Username),
			})
			continue
		}

		canRun, err := checkJobCondition(ctx, jobID, run.Contexts, jobsContext, jobDef)
		if err != nil {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("%v", err),
			})
			return nil, runInfos, sdk.StatusFail, err
		}
		if !canRun && len(wrEnqueue.Jobs) > 0 {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelWarning,
				Message:       fmt.Sprintf("job %s: cannot be run because of if statement", jobID),
			})
			continue
		}
		if canRun {
			jobToQueue[jobID] = jobDef
		}
	}
	currentJobRunStatus := computeJobRunStatus(runJobs)

	return jobToQueue, runInfos, currentJobRunStatus, nil
}

func computeJobRunStatus(runJobs []sdk.V2WorkflowRunJob) string {
	finalStatus := sdk.StatusSuccess
	for _, rj := range runJobs {
		if rj.Status == sdk.StatusFail && sdk.StatusIsTerminated(finalStatus) {
			finalStatus = sdk.StatusFail
		}
		if rj.Status == sdk.StatusBuilding || rj.Status == sdk.StatusWaiting {
			finalStatus = rj.Status
		}
	}
	return finalStatus
}

func checkUserRight(ctx context.Context, db gorp.SqlExecutor, jobDef sdk.V2Job, u sdk.AuthentifiedUser, defaultRegion string) (bool, error) {
	_, next := telemetry.Span(ctx, "checkUserRight")
	defer next()
	if jobDef.Region == "" {
		jobDef.Region = defaultRegion
	}

	wantedRegion, err := region.LoadRegionByName(ctx, db, jobDef.Region)
	if err != nil {
		return false, err
	}

	allowedRegions, err := rbac.LoadRegionIDsByRoleAndUserID(ctx, db, sdk.RegionRoleExecute, u.ID)
	if err != nil {
		next()
		return false, err
	}
	next()
	for _, r := range allowedRegions {
		if r.RegionID == wantedRegion.ID {
			return true, nil
		}
	}
	return false, nil
}

func checkJobNeeds(jobsContext sdk.JobsResultContext, jobDef sdk.V2Job) (string, bool) {
	if len(jobDef.Needs) == 0 {
		return "", true
	}
	for _, need := range jobDef.Needs {
		if _, has := jobsContext[need]; !has {
			return need, false
		}
	}
	return "", true
}

func checkJobCondition(ctx context.Context, jobID string, runContext sdk.WorkflowRunContext, jobsContext sdk.JobsResultContext, jobDef sdk.V2Job) (bool, error) {
	_, next := telemetry.Span(ctx, "checkJobCondition")
	defer next()
	if jobDef.If == "" {
		return true, nil
	}
	if !strings.HasPrefix(jobDef.If, "${{") {
		jobDef.If = fmt.Sprintf("${{ %s }}", jobDef.If)
	}

	conditionContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: runContext,
		Jobs:               jobsContext,
	}
	bts, err := json.Marshal(conditionContext)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return false, sdk.WithStack(err)
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	interpolatedInput, err := ap.Interpolate(ctx, jobDef.If)
	if err != nil {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "job %s: unable to parse if statement", jobID)
	}

	if _, ok := interpolatedInput.(string); !ok {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "job %s: if statement does not return a boolean. Got %v", jobID, interpolatedInput)
	}

	booleanResult, err := strconv.ParseBool(interpolatedInput.(string))
	if err != nil {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "job %s: if statement does not return a boolean. Got %s", jobID, interpolatedInput)
	}
	return booleanResult, nil
}

func buildJobsContext(runJobs []sdk.V2WorkflowRunJob) sdk.JobsResultContext {
	// Compute jobs context
	jobsContext := sdk.JobsResultContext{}
	for _, rj := range runJobs {
		if sdk.StatusIsTerminated(rj.Status) {
			result := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: rj.Outputs,
			}
			jobsContext[rj.JobID] = result
		}
	}
	return jobsContext
}
