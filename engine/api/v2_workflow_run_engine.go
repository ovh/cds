package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opencensus.io/trace"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func (api *API) TriggerBlockedWorkflowRuns(ctx context.Context) {
	tickTrigger := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickTrigger.C:
			wrs, err := workflow_v2.LoadBuildingRunWithEndedJobs(ctx, api.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, wr := range wrs {
				if err := api.triggerBlockedWorkflowRun(ctx, wr); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}

		}
	}
}

func (api *API) V2WorkflowRunEngineChan(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case wrEnqueue := <-api.workflowRunTriggerChan:
			if err := api.workflowRunV2Trigger(ctx, wrEnqueue); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		}
	}
}

func (api *API) V2WorkflowRunEngineDequeue(ctx context.Context) {
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
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		}
	}
}

func (api *API) workflowRunV2Trigger(ctx context.Context, wrEnqueue sdk.V2WorkflowRunEnqueue) error {
	ctx, next := telemetry.Span(ctx, "api.workflowRunV2Trigger")
	defer next()
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, wrEnqueue.RunID)

	_, next = telemetry.Span(ctx, "api.workflowRunV2Trigger.lock")
	lockKey := cache.Key("api:workflow:engine", wrEnqueue.RunID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug(ctx, "api.workflowRunV2Trigger> run %s is locked in cache", wrEnqueue.RunID)
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
	run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), wrEnqueue.RunID, workflow_v2.WithRunResults)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run %s", wrEnqueue.RunID)
	}

	telemetry.Current(ctx).AddAttributes(
		trace.StringAttribute(telemetry.TagProjectKey, run.ProjectKey),
		trace.StringAttribute(telemetry.TagWorkflow, run.WorkflowName),
		trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(run.RunNumber, 10)))
	ctx = context.WithValue(ctx, cdslog.Project, run.ProjectKey)
	ctx = context.WithValue(ctx, cdslog.Workflow, run.WorkflowName)

	proj, err := project.Load(ctx, api.mustDB(), run.ProjectKey, project.LoadOptions.WithIntegrations)
	if err != nil {
		return err
	}
	vcsServer, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), run.ProjectKey, run.VCSServerID)
	if err != nil {
		return sdk.WrapError(err, "unable to load vcs server%s", run.VCSServerID)
	}

	repo, err := repository.LoadRepositoryByID(ctx, api.mustDB(), run.RepositoryID)
	if err != nil {
		return sdk.WrapError(err, "unable to load repository %s", run.RepositoryID)
	}

	if sdk.StatusIsTerminated(run.Status) {
		log.Debug(ctx, "workflow run already on a final state")
		return nil
	}

	u, err := user.LoadByID(ctx, api.mustDB(), wrEnqueue.UserID)
	if err != nil {
		return err
	}

	jobsToQueue, skippedJobs, runMsgs, err := retrieveJobToQueue(ctx, api.mustDB(), run, wrEnqueue, u, api.Config.Workflow.JobDefaultRegion)
	log.Debug(ctx, "workflowRunV2Trigger> jobs to queue: %+v", jobsToQueue)
	if err != nil {
		tx, errTx := api.mustDB().Begin()
		if errTx != nil {
			return sdk.WithStack(errTx)
		}
		defer tx.Rollback()
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
		if errTx := tx.Commit(); errTx != nil {
			return sdk.WithStack(errTx)
		}
		return err
	}

	// Enqueue JOB
	runJobs := prepareRunJobs(ctx, *proj, *run, wrEnqueue, jobsToQueue, sdk.StatusWaiting, *u)

	// Compute worker model on runJobs if needed
	wref := WorkflowRunEntityFinder{
		run:              *run,
		runRepo:          *repo,
		runVcsServer:     *vcsServer,
		workerModelCache: make(map[string]sdk.V2WorkerModel),
		userName:         u.Username,
		repoCache: map[string]sdk.ProjectRepository{
			vcsServer.Name + "/" + repo.Name: *repo,
		},
		vcsServerCache: map[string]sdk.VCSProject{
			vcsServer.Name: *vcsServer,
		},
	}
	runJobsInfos := computeRunJobsWorkerModel(ctx, api.mustDB(), api.Cache, &wref, run, runJobs)

	runJobs = append(runJobs, prepareRunJobs(ctx, *proj, *run, wrEnqueue, skippedJobs, sdk.StatusSkipped, *u)...)

	tx, errTx := api.mustDB().Begin()
	if errTx != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	for i := range runJobs {
		rj := &runJobs[i]
		if err := workflow_v2.InsertRunJob(ctx, tx, rj); err != nil {
			return err
		}
		if info, has := runJobsInfos[rj.JobID]; has {
			info.WorkflowRunJobID = rj.ID
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
				return err
			}
		}
	}

	// Save Run message
	for i := range runMsgs {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsgs[i]); err != nil {
			return err
		}
	}

	// End workflow if there is no more job to handle,  no running jobs and current status is not terminated
	if len(jobsToQueue) == 0 && len(skippedJobs) == 0 && !sdk.StatusIsTerminated(run.Status) {
		finalStatus, err := computeJobRunStatus(ctx, tx, run.ID, run.RunAttempt)
		if err != nil {
			return err
		}
		if finalStatus != run.Status {
			run.Status = finalStatus
			if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	if len(skippedJobs) > 0 {
		// Re enqueue workflow to trigger job after
		api.EnqueueWorkflowRun(ctx, run.ID, run.UserID, run.WorkflowName, run.RunNumber)
	}

	// Send to websocket
	for _, rj := range runJobs {
		runJobEvent := sdk.WebsocketJobQueueEvent{
			Region:       rj.Region,
			ModelType:    rj.ModelType,
			JobRunID:     rj.ID,
			RunNumber:    run.RunNumber,
			WorkflowName: run.WorkflowName,
			ProjectKey:   rj.ProjectKey,
			JobID:        rj.JobID,
		}
		bts, _ := json.Marshal(runJobEvent)
		if err := api.Cache.Publish(ctx, event.JobQueuedPubSubKey, string(bts)); err != nil {
			log.Error(ctx, "%v", err)
		}
	}

	return nil
}

func prepareRunJobIntegration(proj sdk.Project, jobDef sdk.V2Job, runJob *sdk.V2WorkflowRunJob) {
	if !jobDef.Integrations.IsEmpty() {
		runJob.Integrations = &sdk.V2WorkflowRunJobIntegrations{}
		for i := range proj.Integrations {
			integ := &proj.Integrations[i]
			if integ.Name == jobDef.Integrations.Artifacts {
				if integ.Model.ArtifactManager {
					runJob.Integrations.ArtifactManager = integ
				}
				if integ.Model.Deployment {
					runJob.Integrations.Deployment = integ
				}
				break
			}
		}
	}
}

func computeRunJobsWorkerModel(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, run *sdk.V2WorkflowRun, runJobs []sdk.V2WorkflowRunJob) map[string]sdk.V2WorkflowRunJobInfo {
	runJobInfos := make(map[string]sdk.V2WorkflowRunJobInfo)
	for i := range runJobs {
		rj := &runJobs[i]
		if !strings.HasPrefix(rj.Job.RunsOn, "${{") {
			continue
		}
		computeModelCtx := sdk.WorkflowRunJobsContext{
			WorkflowRunContext: run.Contexts,
			Matrix:             rj.Matrix,
		}
		bts, _ := json.Marshal(computeModelCtx)

		var mapContexts map[string]interface{}
		if err := json.Unmarshal(bts, &mapContexts); err != nil {
			rj.Status = sdk.StatusFail
			runJobInfos[rj.JobID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("Job %s: unable to build context to compute worker model: %v", rj.JobID, err),
			}
			continue
		}

		ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
		interpolatedInput, err := ap.Interpolate(ctx, rj.Job.RunsOn)
		if err != nil {
			rj.Status = sdk.StatusFail
			runJobInfos[rj.JobID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s: %v", rj.JobID, rj.Job.RunsOn, err),
			}
			continue
		}

		model, ok := interpolatedInput.(string)
		if !ok {
			rj.Status = sdk.StatusFail
			runJobInfos[rj.JobID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s, it's not a string: %v", rj.JobID, rj.Job.RunsOn, interpolatedInput),
			}
			continue
		}

		completeName, msg, err := wref.checkWorkerModel(ctx, db, store, rj.JobID, model, rj.Region, "")
		if err != nil {
			rj.Status = sdk.StatusFail
			runJobInfos[rj.JobID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s, it's not a string: %v", rj.JobID, model, interpolatedInput),
			}
			continue
		}
		if msg != nil {
			rj.Status = sdk.StatusFail
			runJobInfos[rj.JobID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          msg.Message,
			}
			continue
		}

		if strings.HasPrefix(model, ".cds/worker-models/") {
			rj.ModelType = wref.localWorkerModelCache[model].Type
		} else {
			rj.ModelType = wref.workerModelCache[completeName].Type
		}
		rj.Job.RunsOn = completeName
	}
	return runJobInfos
}

func prepareRunJobs(ctx context.Context, proj sdk.Project, run sdk.V2WorkflowRun, wrEnqueue sdk.V2WorkflowRunEnqueue, jobsToQueue map[string]sdk.V2Job, jobStatus string, u sdk.AuthentifiedUser) []sdk.V2WorkflowRunJob {
	runJobs := make([]sdk.V2WorkflowRunJob, 0)
	for jobID, jobDef := range jobsToQueue {
		// Compute job matrix strategy
		keys := make([]string, 0)
		if jobDef.Strategy != nil {
			for k := range jobDef.Strategy.Matrix {
				keys = append(keys, k)
			}
		}

		alls := make([]map[string]string, 0)
		if jobDef.Strategy != nil {
			generateMatrix(jobDef.Strategy.Matrix, keys, 0, make(map[string]string), &alls)
		}

		if len(alls) == 0 {
			runJob := sdk.V2WorkflowRunJob{
				WorkflowRunID: run.ID,
				Status:        jobStatus,
				JobID:         jobID,
				Job:           jobDef,
				UserID:        wrEnqueue.UserID,
				Username:      u.Username,
				ProjectKey:    run.ProjectKey,
				Region:        jobDef.Region,
				WorkflowName:  run.WorkflowName,
				RunNumber:     run.RunNumber,
				RunAttempt:    run.RunAttempt,
			}
			if jobDef.RunsOn != "" {
				runJob.ModelType = run.WorkflowData.WorkerModels[jobDef.RunsOn].Type
			}
			prepareRunJobIntegration(proj, jobDef, &runJob)
			runJobs = append(runJobs, runJob)
		} else {
			for _, m := range alls {
				runJob := sdk.V2WorkflowRunJob{
					WorkflowRunID: run.ID,
					Status:        jobStatus,
					JobID:         jobID,
					Job:           jobDef,
					UserID:        wrEnqueue.UserID,
					Username:      u.Username,
					ProjectKey:    run.ProjectKey,
					Region:        jobDef.Region,
					WorkflowName:  run.WorkflowName,
					RunNumber:     run.RunNumber,
					RunAttempt:    run.RunAttempt,
					Matrix:        sdk.JobMatrix{},
				}
				for k, v := range m {
					runJob.Matrix[k] = v
				}
				if jobDef.RunsOn != "" {
					runJob.ModelType = run.WorkflowData.WorkerModels[jobDef.RunsOn].Type
				}
				prepareRunJobIntegration(proj, jobDef, &runJob)
				runJobs = append(runJobs, runJob)
			}
		}
	}
	return runJobs
}

func generateMatrix(matrix map[string][]string, keys []string, keyIndex int, current map[string]string, alls *[]map[string]string) {
	if len(current) == len(keys) {
		combinationCopy := make(map[string]string)
		for k, v := range current {
			combinationCopy[k] = v
		}
		*alls = append(*alls, combinationCopy)
		return
	}

	key := keys[keyIndex]
	values := matrix[key]
	for _, value := range values {
		current[key] = value
		generateMatrix(matrix, keys, keyIndex+1, current, alls)
		delete(current, key)
	}
}

// Return jobToQueue, skippedJob, runInfos, error
func retrieveJobToQueue(ctx context.Context, db *gorp.DbMap, run *sdk.V2WorkflowRun, wrEnqueue sdk.V2WorkflowRunEnqueue, u *sdk.AuthentifiedUser, defaultRegion string) (map[string]sdk.V2Job, map[string]sdk.V2Job, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "retrieveJobToQueue")
	defer next()
	runInfos := make([]sdk.V2WorkflowRunInfo, 0)
	jobToQueue := make(map[string]sdk.V2Job)

	// Load run_jobs
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, db, run.ID, run.RunAttempt)
	if err != nil {
		return nil, nil, nil, sdk.WrapError(err, "unable to load workflow run jobs for run %s", wrEnqueue.RunID)
	}

	// all current runJobs Status
	allrunJobsMap := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runJobs {
		allrunJobsMap[rj.JobID] = rj
	}

	// Compute run context
	jobsContext := buildJobsContext(*run, runJobs)

	// Select jobs to check ( all workflow or list of jobs from enqueue request )
	jobsToCheck := make(map[string]sdk.V2Job)

	for jobID, jobDef := range run.WorkflowData.Workflow.Jobs {
		// Do not enqueue jobs that have already a run
		if _, has := allrunJobsMap[jobID]; !has {
			jobsToCheck[jobID] = jobDef
		}
	}

	// Check jobs : Needs / Condition / User Right

	stages := run.GetStages()
	if len(stages) > 0 {
		for k, j := range allrunJobsMap {
			jobStage := run.WorkflowData.Workflow.Jobs[k].Stage
			stages[jobStage].Jobs[k] = j.Status
		}
		stages.ComputeStatus()
	}

	skippedJob := make(map[string]sdk.V2Job)
	for jobID, jobDef := range jobsToCheck {
		if len(stages) > 0 && !stages[jobDef.Stage].CanBeRun {
			continue
		}

		// Check jobs needs
		if ok := checkJobNeeds(jobsContext, jobDef); !ok {
			continue
		}

		canBeQueued, infos, err := checkJob(ctx, db, *u, wrEnqueue, *run, jobsContext, jobID, &jobDef, defaultRegion)
		runInfos = append(runInfos, infos...)
		if err != nil {
			skippedJob[jobID] = jobDef
			return nil, nil, runInfos, err
		}

		if canBeQueued {
			jobToQueue[jobID] = jobDef
		} else {
			skippedJob[jobID] = jobDef
		}
	}

	return jobToQueue, skippedJob, runInfos, nil
}

func checkJob(ctx context.Context, db gorp.SqlExecutor, u sdk.AuthentifiedUser, wrEnqueue sdk.V2WorkflowRunEnqueue, run sdk.V2WorkflowRun, jobsContext sdk.JobsResultContext, jobID string, jobDef *sdk.V2Job, defaultRegion string) (bool, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "checkJob", trace.StringAttribute(telemetry.TagJob, jobID))
	defer next()

	runInfos := make([]sdk.V2WorkflowRunInfo, 0)
	if _, has := jobsContext[jobID]; has {
		log.Debug(ctx, "job %s: already executed, skip it", jobID)
		return false, nil, nil
	}

	// Check user right
	hasRight, err := checkUserRight(ctx, db, jobDef, u, defaultRegion)
	if err != nil {
		runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("job %s: unable to check right for user %s: %v", jobID, u.Username, err),
		})
		return false, runInfos, err
	}
	if !hasRight {
		runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelWarning,
			Message:       fmt.Sprintf("job %s: user %s does not have enough right", jobID, u.Username),
		})
		return false, runInfos, nil
	}

	// check job condition
	canRun, err := checkJobCondition(ctx, jobID, run.Contexts, jobsContext, run.WorkflowData.Workflow.Jobs)
	if err != nil {
		runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("%v", err),
		})
		return false, runInfos, err
	}
	if !canRun {
		runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelInfo,
			Message:       fmt.Sprintf("Job %s: The condition is not satisfied.", jobID),
		})
	}
	return canRun, runInfos, nil
}

func computeJobRunStatus(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempt int64) (string, error) {
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, db, runID, runAttempt)
	if err != nil {
		return "", err
	}

	finalStatus := sdk.StatusSuccess
	for _, rj := range runJobs {
		if rj.Status == sdk.StatusFail && finalStatus != sdk.StatusStopped && sdk.StatusIsTerminated(finalStatus) && !rj.Job.ContinueOnError {
			finalStatus = rj.Status
		}
		if rj.Status == sdk.StatusStopped && sdk.StatusIsTerminated(finalStatus) {
			finalStatus = sdk.StatusStopped
		}
		if !sdk.StatusIsTerminated(rj.Status) {
			finalStatus = sdk.StatusBuilding
		}
	}
	return finalStatus, nil
}

func checkUserRight(ctx context.Context, db gorp.SqlExecutor, jobDef *sdk.V2Job, u sdk.AuthentifiedUser, defaultRegion string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkUserRight")
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

func checkJobNeeds(jobsContext sdk.JobsResultContext, jobDef sdk.V2Job) bool {
	if len(jobDef.Needs) == 0 {
		return true
	}
	for _, need := range jobDef.Needs {
		if _, has := jobsContext[need]; !has {
			return false
		}
	}
	return true
}

func checkJobCondition(ctx context.Context, jobID string, runContext sdk.WorkflowRunContext, jobsContext sdk.JobsResultContext, allJobs map[string]sdk.V2Job) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkJobCondition")
	defer next()

	// On keep ancestor of the current job
	currentJobContext := sdk.JobsResultContext{}
	buildAncestorJobContext(allJobs, jobID, currentJobContext, jobsContext)

	jobDef := allJobs[jobID]
	if jobDef.If == "" {
		jobDef.If = "${{success()}}"
	}
	if !strings.HasPrefix(jobDef.If, "${{") {
		jobDef.If = fmt.Sprintf("${{ %s }}", jobDef.If)
	}

	conditionContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: runContext,
		Jobs:               jobsContext,
		Needs:              sdk.NeedsContext{},
	}
	for _, n := range jobDef.Needs {
		if j, has := currentJobContext[n]; has {
			needContext := sdk.NeedContext{
				Result:  j.Result,
				Outputs: j.Outputs,
			}
			conditionContext.Needs[n] = needContext
		}
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

func buildAncestorJobContext(jobs map[string]sdk.V2Job, jobID string, currentJobContext sdk.JobsResultContext, jobsContext sdk.JobsResultContext) {
	jobDef := jobs[jobID]
	if len(jobDef.Needs) == 0 {
		return
	}
	for _, n := range jobDef.Needs {
		buildAncestorJobContext(jobs, n, currentJobContext, jobsContext)

		jobCtx := jobsContext[n]
		if jobs[n].ContinueOnError {
			jobCtx.Result = sdk.StatusSuccess
		}
		currentJobContext[n] = jobCtx
	}
}

func buildJobsContext(run sdk.V2WorkflowRun, runJobs []sdk.V2WorkflowRunJob) sdk.JobsResultContext {
	runResultMap := make(map[string][]sdk.V2WorkflowRunResultVariableDetail)
	for _, rr := range run.Results {
		if rr.Type != sdk.V2WorkflowRunResultTypeVariable {
			continue
		}
		detail := rr.Detail.Data.(*sdk.V2WorkflowRunResultVariableDetail)
		jobResults, has := runResultMap[rr.WorkflowRunJobID]
		if !has {
			jobResults = make([]sdk.V2WorkflowRunResultVariableDetail, 0)
		}
		jobResults = append(jobResults, *detail)
		runResultMap[rr.WorkflowRunJobID] = jobResults
	}

	// Compute jobs context
	jobsContext := sdk.JobsResultContext{}
	matrixJobs := make(map[string][]sdk.JobResultContext)
	for _, rj := range runJobs {
		if sdk.StatusIsTerminated(rj.Status) && len(rj.Matrix) == 0 {
			result := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: sdk.JobResultOutput{},
			}
			if rr, has := runResultMap[rj.ID]; has {
				for _, r := range rr {
					result.Outputs[r.Name] = r.Value
				}
			}
			jobsContext[rj.JobID] = result
		} else if len(rj.Matrix) > 0 {
			jobs, has := matrixJobs[rj.JobID]
			if !has {
				jobs = make([]sdk.JobResultContext, 0)
			}
			jobResultContext := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: sdk.JobResultOutput{},
			}
			rr, has := runResultMap[rj.ID]
			if has {
				for _, r := range rr {
					jobResultContext.Outputs[r.Name] = r.Value
				}
			}
			jobs = append(jobs, jobResultContext)
			matrixJobs[rj.JobID] = jobs
		}
	}

	// Manage matric jobs
nextjob:
	for k := range matrixJobs {
		outputs := sdk.JobResultOutput{}
		var finalStatus string
		for _, rj := range matrixJobs[k] {
			if !sdk.StatusIsTerminated(rj.Result) {
				continue nextjob
			}
			for outputK, outputV := range rj.Outputs {
				outputs[outputK] = outputV
			}

			switch finalStatus {
			case "":
				finalStatus = rj.Result
			case sdk.StatusSuccess:
				if rj.Result == sdk.StatusStopped || rj.Result == sdk.StatusFail {
					finalStatus = rj.Result
				}
			case sdk.StatusFail:
				if rj.Result == sdk.StatusStopped {
					finalStatus = rj.Result
				}
			}
		}
		result := sdk.JobResultContext{
			Result:  finalStatus,
			Outputs: outputs,
		}
		jobsContext[k] = result
	}

	return jobsContext
}

func (api *API) triggerBlockedWorkflowRun(ctx context.Context, wr sdk.V2WorkflowRun) error {
	ctx = context.WithValue(ctx, cdslog.Workflow, wr.WorkflowName)
	ctx, next := telemetry.Span(ctx, "api.triggerBlockedWorkflowRun")
	defer next()

	_, next = telemetry.Span(ctx, "api.triggerBlockedWorkflowRun.lock")
	lockKey := cache.Key("api:workflow:engine", wr.ID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	log.Info(ctx, "triggerBlockedWorkflowRun: trigger workflow %s for run %d", wr.WorkflowName, wr.RunNumber)
	if wr.Status != sdk.StatusBuilding {
		return nil
	}

	// Search last userID that trigger a job in this run
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
	if err != nil {
		return err
	}
	var lastJobs sdk.V2WorkflowRunJob
	for _, rj := range runJobs {
		if !sdk.StatusIsTerminated(rj.Status) {
			return nil
		}
		if rj.Started.After(lastJobs.Started) {
			lastJobs = rj
		}
	}

	userID := lastJobs.UserID
	// No job have been triggered
	if userID == "" {
		userID = wr.UserID
	}

	api.EnqueueWorkflowRun(ctx, wr.ID, userID, wr.WorkflowName, wr.RunNumber)
	return nil
}

func (api *API) EnqueueWorkflowRun(ctx context.Context, runID string, userID string, workflowName string, runNumber int64) {
	// Continue workflow
	enqueueRequest := sdk.V2WorkflowRunEnqueue{
		RunID:  runID,
		UserID: userID,
	}
	select {
	case api.workflowRunTriggerChan <- enqueueRequest:
		log.Debug(ctx, "workflow run %s %d trigger in chan", workflowName, runNumber)
	default:
		if err := api.Cache.Enqueue(workflow_v2.WorkflowEngineKey, enqueueRequest); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}
}
