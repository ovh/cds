package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type prepareJobData struct {
	wrEnqueue       sdk.V2WorkflowRunEnqueue
	runJobContext   sdk.WorkflowRunJobsContext
	existingRunJobs []sdk.V2WorkflowRunJob
	jobID           string
	jobToTrigger    JobToTrigger
	defaultRegion   string
	regionPermCache map[string]*sdk.V2WorkflowRunJobInfo
	allVariableSets []sdk.ProjectVariableSet
}

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
			ctxTrigger := context.WithValue(ctx, cdslog.WorkflowRunID, wrEnqueue.RunID)
			if err := api.workflowRunV2Trigger(ctxTrigger, wrEnqueue); err != nil {
				log.ErrorWithStackTrace(ctxTrigger, err)
			}
		}
	}
}

func (api *API) V2WorkflowRunEngineDequeue(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "V2WorkflowRunEngine> Exiting: %v", err)
			return
		}

		var wrEnqueue sdk.V2WorkflowRunEnqueue
		if err := api.Cache.DequeueWithContext(ctx, workflow_v2.WorkflowEngineKey, 250*time.Millisecond, &wrEnqueue); err != nil {
			log.Error(ctx, "V2WorkflowRunEngine> DequeueWithContext err: %v", err)
			continue
		}
		// Compatibility code
		if wrEnqueue.DeprecatedUserID != "" && wrEnqueue.Initiator.UserID == "" {
			wrEnqueue.Initiator.UserID = wrEnqueue.DeprecatedUserID
		}
		ctxTrigger := context.WithValue(ctx, cdslog.WorkflowRunID, wrEnqueue.RunID)
		if err := api.workflowRunV2Trigger(ctxTrigger, wrEnqueue); err != nil {
			log.ErrorWithStackTrace(ctxTrigger, err)
		}
	}
}

func (api *API) workflowRunV2Trigger(ctx context.Context, wrEnqueue sdk.V2WorkflowRunEnqueue) error {
	ctx, next := telemetry.Span(ctx, "api.workflowRunV2Trigger")
	defer next()

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
	run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), wrEnqueue.RunID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
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

	if run.Status.IsTerminated() {
		log.Debug(ctx, "workflow run already on a final state")
		return nil
	}

	runResults, err := workflow_v2.LoadRunResultsByRunIDAttempt(ctx, api.mustDB(), run.ID, run.RunAttempt)
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run results for run %s", wrEnqueue.RunID)
	}

	allRunJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), run.ID, run.RunAttempt)
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run jobs for run %s", wrEnqueue.RunID)
	}
	allrunJobsMap := make(map[string]sdk.V2WorkflowRunJob)
	allreadyExistRunJobs := make(map[string]struct{})
	for _, rj := range allRunJobs {
		allreadyExistRunJobs[rj.ID] = struct{}{}
		// addition check for matrix job, only keep not terminated one if present
		runjob, has := allrunJobsMap[rj.JobID]
		if !has || runjob.Status.IsTerminated() {
			allrunJobsMap[rj.JobID] = rj
		}
	}

	runJobsContexts, runGatesContexts := computeExistingRunJobContexts(ctx, allRunJobs, runResults)

	// Compute annotations
	if err := api.computeWorkflowRunAnnotations(ctx, run, runJobsContexts, runGatesContexts); err != nil {
		return err
	}

	jobsToQueue, runMsgs, errRetrieve := retrieveJobToQueue(ctx, api.mustDB(), wrEnqueue, run, allRunJobs, allrunJobsMap, runJobsContexts, api.Config.Workflow.JobDefaultRegion)
	if errRetrieve != nil {
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()
		for i := range runMsgs {
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsgs[i]); err != nil {
				return err
			}
		}
		run.Status = sdk.V2WorkflowRunStatusFail
		if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, allrunJobsMap, runResults, &wrEnqueue.Initiator)
		return errRetrieve
	}

	vss := make([]sdk.ProjectVariableSet, 0, len(run.WorkflowData.Workflow.VariableSets))
	for _, vs := range run.WorkflowData.Workflow.VariableSets {
		vsDB, err := project.LoadVariableSetByName(ctx, api.mustDB(), run.ProjectKey, vs)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		// If not found stop the run
		if err != nil {
			return failRunWithMessage(ctx, api.mustDB(), api.Cache, run, []sdk.V2WorkflowRunInfo{
				{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("variable set %s not found on project", vs),
				},
			}, allrunJobsMap, runResults, &wrEnqueue.Initiator)
		}
		vsDB.Items, err = project.LoadVariableSetAllItem(ctx, api.mustDB(), vsDB.ID)
		if err != nil {
			return err
		}
		vss = append(vss, *vsDB)
	}
	variableSetCtx, _, err := buildVarsContext(ctx, vss)
	if err != nil {
		return failRunWithMessage(ctx, api.mustDB(), api.Cache, run, []sdk.V2WorkflowRunInfo{
			{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("unable to compute variableset into job context: %v", err),
			},
		}, allrunJobsMap, runResults, &wrEnqueue.Initiator)
	}

	// Compute worker model / region on runJobs if needed
	wref, err := NewWorkflowRunEntityFinder(ctx, api.mustDB(), *proj, *run, *repo, *vcsServer, run.WorkflowRef, run.WorkflowSha, api.Config.WorkflowV2.LibraryProjectKey, &wrEnqueue.Initiator)
	if err != nil {
		return err
	}
	wref.ef.repoCache[vcsServer.Name+"/"+repo.Name] = *repo
	wref.ef.vcsServerCache[vcsServer.Name] = *vcsServer

	// Enqueue JOB
	hasTemplatedMatrixedJob := false
	for _, j := range jobsToQueue {
		if !j.Status.IsTerminated() && j.Job.From != "" && j.Job.Strategy != nil && len(j.Job.Strategy.Matrix) > 0 {
			hasTemplatedMatrixedJob = true
		}
	}
	runJobs, runJobsInfos, errorMsg, runUpdated, err := prepareRunJobs(ctx, api.mustDB(), api.Cache, proj, wref, run, allRunJobs, variableSetCtx, wrEnqueue, jobsToQueue, runJobsContexts, api.Config.Workflow.JobDefaultRegion)
	if err != nil {
		return err
	}

	if errorMsg != nil {
		return failRunWithMessage(ctx, api.mustDB(), api.Cache, run, errorMsg, allrunJobsMap, runResults, &wrEnqueue.Initiator)
	}

	tx, errTx := api.mustDB().Begin()
	if errTx != nil {
		return sdk.WithStack(errTx)
	}
	defer tx.Rollback() // nolint

	hasNoStepsJobs := false
	for i := range runJobs {
		rj := &runJobs[i]
		if len(rj.Job.Steps) == 0 {
			hasNoStepsJobs = true
		}
		if rj.Status.IsTerminated() && (rj.Ended == nil || rj.Ended.IsZero()) {
			now := time.Now()
			rj.Ended = &now
		}
		if _, has := allreadyExistRunJobs[rj.ID]; !has {
			if err := workflow_v2.InsertRunJob(ctx, tx, rj); err != nil {
				return err
			}
		} else {
			if err := workflow_v2.UpdateJobRun(ctx, tx, rj); err != nil {
				return err
			}
		}

		if info, has := runJobsInfos[rj.ID]; has {
			info.WorkflowRunJobID = rj.ID
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
				return err
			}
		}
		if rj.GateInputs != nil {
			jobInfo := sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    rj.WorkflowRunID,
				Level:            sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:         time.Now(),
				WorkflowRunJobID: rj.ID,
				Message:          wrEnqueue.Initiator.Username() + " triggers manually this job",
			}
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &jobInfo); err != nil {
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

	hasSkippedOrFailedJob := false
	for _, rj := range runJobs {
		if rj.Status == sdk.V2WorkflowRunJobStatusSkipped || rj.Status == sdk.V2WorkflowRunJobStatusFail {
			hasSkippedOrFailedJob = true
			break
		}
	}

	// End workflow if there is no more job to handle,  no running jobs and current status is not terminated
	if runUpdated || (len(jobsToQueue) == 0 && !hasSkippedOrFailedJob && !run.Status.IsTerminated()) {
		finalStatus, err := computeRunStatusFromJobsStatus(ctx, tx, run.ID, run.RunAttempt, len(run.WorkflowData.Workflow.Jobs))
		if err != nil {
			return err
		}
		if finalStatus != run.Status || runUpdated {
			run.Status = finalStatus
		}
	}

	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	// Synchronize run result in a separate transaction
	api.GoRoutines.Exec(ctx, "api.synchronizeRunResults", func(ctx context.Context) {
		if err := api.synchronizeRunResults(ctx, api.mustDBWithCtx(ctx), run.ID); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	})

	if run.Status.IsTerminated() {
		// Send event
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, allrunJobsMap, runResults, &wrEnqueue.Initiator)

		// Send event to hook uservice
		hookServices, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
		if err != nil {
			return err
		}
		if len(hookServices) < 1 {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to find hook service")
		}
		req := sdk.HookWorkflowRunEvent{
			WorkflowProject:    run.ProjectKey,
			WorkflowVCSServer:  run.VCSServer,
			WorkflowRepository: run.Repository,
			WorkflowName:       run.WorkflowName,
			WorkflowStatus:     run.Status,
			WorkflowRunID:      run.ID,
			WorkflowRef:        run.Contexts.Git.Ref,
			Request: sdk.HookWorkflowRunEventRequest{
				WorkflowRun: sdk.HookWorkflowRunEventRequestWorkflowRun{
					CDS:                run.Contexts.CDS,
					Git:                run.Contexts.Git,
					DeprecatedUserID:   run.Initiator.UserID,
					DeprecatedUserName: run.Initiator.Username(),
					Conclusion:         string(run.Status),
					CreatedAt:          run.Started,
					Jobs:               make(map[string]sdk.HookWorkflowRunEventJob),
					Initiator:          wrEnqueue.Initiator,
				},
			},
		}
		for _, rj := range allRunJobs {
			req.Request.WorkflowRun.Jobs[rj.JobID] = sdk.HookWorkflowRunEventJob{
				Conclusion: string(rj.Status),
			}
		}
		if _, _, err := services.NewClient(hookServices).DoJSONRequest(ctx, http.MethodPost, "/v2/workflow/outgoing", req, nil); err != nil {
			return err
		}
		return nil
	}

	if hasSkippedOrFailedJob || hasNoStepsJobs || hasTemplatedMatrixedJob {
		// Re enqueue workflow to trigger job after
		api.EnqueueWorkflowRun(ctx, run.ID, wrEnqueue.Initiator, run.WorkflowName, run.RunNumber)
	}

	// Send to websocket
	for _, rj := range runJobs {
		switch rj.Status {
		case sdk.V2WorkflowRunJobStatusFail:
			event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, *run, rj)
		default:
			event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnqueued, *run, rj)
		}
	}
	return nil
}

func failRunWithMessage(ctx context.Context, db *gorp.DbMap, cache cache.Store, run *sdk.V2WorkflowRun, msgs []sdk.V2WorkflowRunInfo, jobRunMap map[string]sdk.V2WorkflowRunJob, runResult []sdk.V2WorkflowRunResult, initiator *sdk.V2Initiator) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()
	for _, msg := range msgs {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &msg); err != nil {
			return err
		}
	}
	run.Status = sdk.V2WorkflowRunStatusFail
	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	event_v2.PublishRunEvent(ctx, cache, sdk.EventRunEnded, *run, jobRunMap, runResult, initiator)
	return err
}

func (api *API) computeWorkflowRunAnnotations(ctx context.Context, run *sdk.V2WorkflowRun, runJobsContexts sdk.JobsResultContext, runGatesContexts sdk.JobsGateContext) error {
	// Build the context that is available for expression syntax
	computeAnnotationsJosbCtx := make(map[string]sdk.ComputeAnnotationsJobContext)
	for jobID, jobResult := range runJobsContexts {
		computeAnnotationsJosbCtx[jobID] = sdk.ComputeAnnotationsJobContext{
			Results: jobResult,
			Gate:    runGatesContexts[jobID],
		}
	}

	computeAnnotationsCtx := sdk.ComputeAnnotationsContext{
		WorkflowRunContext: run.Contexts,
		Jobs:               computeAnnotationsJosbCtx,
	}

	bts, _ := json.Marshal(computeAnnotationsCtx)
	var mapContexts map[string]interface{}
	_ = json.Unmarshal(bts, &mapContexts) // error cannot happen here

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)

	for k, v := range run.WorkflowData.Workflow.Annotations {
		if _, exist := run.Annotations[k]; exist { // If the annotation has already been set: next
			continue
		}
		value, err := ap.InterpolateToString(ctx, v)
		if err != nil {
			// If error, insert a run info
			runInfo := sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelWarning,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("unable to compute annotation %q: %v", v, err),
			}
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
				tx.Rollback()
				return err
			}
			if err := tx.Commit(); err != nil {
				tx.Rollback()
				return sdk.WithStack(err)
			}
			continue
		}
		annotationValue := strings.TrimSpace(value)
		if annotationValue != "" && strings.ToLower(annotationValue) != sdk.FalseString {
			if run.Annotations == nil {
				run.Annotations = sdk.WorkflowRunAnnotations{}
			}
			run.Annotations[k] = annotationValue
		}
	}
	return nil
}

type ArtifactSignature map[string]string

/*
synchronizeRunResults : for a runID, this func:
- get the integration ArtifactManager on the workflow if exist
- for each run results, add properties and signed properties
- delete build, then create build info.
*/
func (api *API) synchronizeRunResults(ctx context.Context, db gorp.SqlExecutor, runID string) error {
	run, err := workflow_v2.LoadRunByID(ctx, db, runID)
	if err != nil {
		return err
	}

	proj, err := project.Load(ctx, db, run.ProjectKey, project.LoadOptions.WithClearIntegrations)
	if err != nil {
		return err
	}

	runResults, err := workflow_v2.LoadRunResultsByRunIDAttempt(ctx, db, runID, run.RunAttempt)
	if err != nil {
		return err
	}

	// Prepare artifactClient if available
	// Only one artifact_manager integration is available on a workflow run
	var (
		artifactClient         artifact_manager.ArtifactManager
		artifactoryIntegration *sdk.ProjectIntegration
		rtToken                string
		rtTokenName            string
		rtURL                  string
		artifactoryProjectKey  string
	)

	var integrations []sdk.ProjectIntegration
	for _, integName := range run.WorkflowData.Workflow.Integrations {
		for i := range proj.Integrations {
			if proj.Integrations[i].Name == integName {
				integrations = append(integrations, proj.Integrations[i])
				break
			}
		}
	}

	for i := range integrations {
		integ := integrations[i]

		if integ.Model.ArtifactManager {
			rtName := integ.Config[sdk.ArtifactoryConfigPlatform].Value
			rtURL = integ.Config[sdk.ArtifactoryConfigURL].Value
			rtToken = integ.Config[sdk.ArtifactoryConfigToken].Value
			rtTokenName = integ.Config[sdk.ArtifactoryConfigTokenName].Value
			artifactoryProjectKey = integ.Config[sdk.ArtifactoryConfigProjectKey].Value

			var err error
			artifactClient, err = artifact_manager.NewClient(rtName, rtURL, rtToken)
			if err != nil {
				return err
			}
			artifactoryIntegration = &integ
			break
		}
	}

	if artifactClient == nil || artifactoryIntegration == nil {
		return nil
	}

	for i := range runResults {
		result := &runResults[i]

		jobRun, err := workflow_v2.LoadRunJobByID(ctx, db, result.WorkflowRunJobID)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to load run job by ID %s: %v", result.WorkflowRunJobID, err)
			continue
		}

		if result.ArtifactManagerIntegrationName == nil {
			continue
		}
		if result.DataSync == nil {
			result.DataSync = new(sdk.WorkflowRunResultSync)
		}

		latestPromotion := result.DataSync.LatestPromotionOrRelease()
		currentMaturity := result.ArtifactManagerMetadata.Get("maturity")
		if latestPromotion != nil && currentMaturity != latestPromotion.ToMaturity {
			return sdk.Errorf("desynchronized maturity and promotion on run result %s", result.ID)
		}

		// Set the properties
		virtualRepository := result.ArtifactManagerMetadata.Get("repository")
		localRepository := result.ArtifactManagerMetadata.Get("localRepository")
		name := result.ArtifactManagerMetadata.Get("name")

		repoDetails, err := artifactClient.GetRepository(localRepository)
		if err != nil {
			log.Error(ctx, "unable to get repository %q fror result %s: %v", localRepository, result.ID, err)
			continue
		}

		// To get FileInfo for a docker image, we have to check the manifest file
		filePath := result.ArtifactManagerMetadata.Get("path")
		if repoDetails.PackageType == "docker" && !strings.HasSuffix(filePath, "manifest.json") {
			filePath = path.Join(filePath, "manifest.json")
		}

		fi, err := artifactClient.GetFileInfo(localRepository, filePath)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact info from result %s: %v", result.ID, err)
			continue
		}

		existingProperties, err := artifactClient.GetProperties(localRepository, filePath)
		if err != nil && !strings.Contains(err.Error(), "404") {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact properties from result %s: %v", result.ID, err)
			continue
		}

		if sdk.MapHasKeys(existingProperties, "cds.signature") {
			log.Debug(ctx, "artifact is already signed by cds")
			continue
		}

		// Push git properties as artifact properties
		props := utils.NewProperties()
		signedProps := make(ArtifactSignature)

		props.AddProperty("cds.project", run.ProjectKey)
		signedProps["cds.project"] = run.ProjectKey
		props.AddProperty("cds.workflow", run.WorkflowName)
		signedProps["cds.workflow"] = run.WorkflowName
		props.AddProperty("cds.version", run.Contexts.CDS.Version)
		signedProps["cds.version"] = run.Contexts.CDS.Version
		props.AddProperty("cds.run", strconv.FormatInt(run.RunNumber, 10))
		signedProps["cds.run"] = strconv.FormatInt(run.RunNumber, 10)
		props.AddProperty("git.url", run.Contexts.Git.RepositoryURL)
		signedProps["git.url"] = run.Contexts.Git.RepositoryURL
		props.AddProperty("git.hash", run.Contexts.Git.Sha)
		signedProps["git.hash"] = run.Contexts.Git.Sha
		props.AddProperty("git.ref", run.Contexts.Git.Ref)
		signedProps["git.ref"] = run.Contexts.Git.Ref
		props.AddProperty("cds.run_id", runID)
		signedProps["cds.run_id"] = runID
		signedProps["cds.region"] = jobRun.Region
		signedProps["cds.worker"] = jobRun.WorkerName
		signedProps["cds.hatchery"] = jobRun.HatcheryName

		// Prepare artifact signature
		signedProps["repository"] = virtualRepository
		signedProps["type"] = repoDetails.PackageType
		signedProps["path"] = fi.Path
		signedProps["name"] = name

		if fi.Checksums == nil {
			log.Error(ctx, "unable to get checksums for artifact %s %s", name, fi.Path)
		} else {
			signedProps["md5"] = fi.Checksums.Md5
			signedProps["sha1"] = fi.Checksums.Sha1
			signedProps["sha256"] = fi.Checksums.Sha256
		}

		// Sign the properties with main CDS authentication key pair
		signature, err := authentication.SignJWS(signedProps, time.Now(), 0)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact properties from result %s: %v", result.ID, err)
			continue
		}
		props.AddProperty("cds.signature", signature)

		pathToApplySet := fi.Path
		// If dir property exist (for artifact manifest.json or list.manifest.json), we'll use it to SetProperties
		if result.ArtifactManagerMetadata.Get("dir") != "" {
			pathToApplySet = result.ArtifactManagerMetadata.Get("dir")
		}

		log.Info(ctx, "setProperties artifact %s%s signature: %s", localRepository, pathToApplySet, signature)
		if err := artifactClient.SetProperties(localRepository, pathToApplySet, props); err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to set artifact properties from result %s: %v", result.ID, err)
			continue
		}
	}

	// Set the Buildinfo
	buildInfoRequest, err := art.PrepareBuildInfo(ctx, artifactClient, art.BuildInfoRequest{
		BuildInfoPrefix:          artifactoryIntegration.Config[sdk.ArtifactoryConfigBuildInfoPrefix].Value,
		ProjectKey:               run.ProjectKey,
		VCS:                      run.Contexts.CDS.WorkflowVCSServer,
		Repository:               run.Contexts.CDS.WorkflowRepository,
		WorkflowName:             run.WorkflowName,
		Version:                  run.Contexts.CDS.Version,
		AgentName:                "cds-api",
		TokenName:                rtTokenName,
		RunURL:                   fmt.Sprintf("%s/project/%s/run/%s", api.Config.URL.UI, run.ProjectKey, runID),
		GitBranch:                run.Contexts.Git.Ref,
		GitURL:                   run.Contexts.Git.RepositoryURL,
		GitHash:                  run.Contexts.Git.Sha,
		RunResultsV2:             runResults,
		DefaultLowMaturitySuffix: artifactoryIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value,
	})
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return err
	}

	log.Info(ctx, "Creating Artifactory Build %s %s on project %s...", buildInfoRequest.Name, buildInfoRequest.Number, artifactoryProjectKey)

	if err := artifactClient.DeleteBuild(artifactoryProjectKey, buildInfoRequest.Name, buildInfoRequest.Number); err != nil {
		log.ErrorWithStackTrace(ctx, err)
	}

	var nbAttempts int
	for {
		nbAttempts++
		err := artifactClient.PublishBuildInfo(artifactoryProjectKey, buildInfoRequest)
		if err == nil {
			break
		} else if nbAttempts >= 3 {
			log.ErrorWithStackTrace(ctx, err)
			break
		} else {
			log.Error(ctx, "error while pushing buildinfo %s %s. Retrying...", buildInfoRequest.Name, buildInfoRequest.Number)
		}
	}

	return nil
}

func computeRunJobsInterpolation(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, run *sdk.V2WorkflowRun, rj *sdk.V2WorkflowRunJob, defaultRegion string, regionPermCache map[string]*sdk.V2WorkflowRunJobInfo, jobContext sdk.WorkflowRunJobsContext, wrEnqueue sdk.V2WorkflowRunEnqueue) (*sdk.V2WorkflowRunJobInfo, bool) {
	runUpdated := false
	if rj.Status.IsTerminated() {
		return nil, false
	}

	bts, _ := json.Marshal(jobContext)
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		rj.Status = sdk.V2WorkflowRunJobStatusFail
		return &sdk.V2WorkflowRunJobInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			IssuedAt:      time.Now(),
			Message:       fmt.Sprintf("Job %s: unable to build context to compute worker model: %v", rj.JobID, err),
		}, false
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)

	for i, integ := range rj.Job.Integrations {
		if strings.Contains(integ, "${{") {
			integName, err := ap.InterpolateToString(ctx, integ)
			if err != nil {
				rj.Status = sdk.V2WorkflowRunJobStatusFail
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    run.ID,
					Level:            sdk.WorkflowRunInfoLevelError,
					WorkflowRunJobID: rj.ID,
					IssuedAt:         time.Now(),
					Message:          fmt.Sprintf("Job %s: unable to interpolate integration %s into a string: %v", rj.JobID, integ, err),
				}, false
			}
			rj.Job.Integrations[i] = integName
			foundInteg := false
		integLoop:
			for _, pi := range wref.project.Integrations {
				if pi.Name != integName {
					continue
				}
				foundInteg = true
				for _, v := range pi.Config {
					if v.Type == sdk.IntegrationConfigTypeRegion {
						rj.Job.Region = v.Value
						rj.Region = v.Value
						break integLoop
					}
				}
			}
			if !foundInteg {
				rj.Status = sdk.V2WorkflowRunJobStatusFail
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    run.ID,
					Level:            sdk.WorkflowRunInfoLevelError,
					WorkflowRunJobID: rj.ID,
					IssuedAt:         time.Now(),
					Message:          fmt.Sprintf("Job %s: unable to find integration %s: %v", rj.JobID, integName, err),
				}, false
			}
		}
	}

	if strings.Contains(rj.Job.Name, "${{") {
		jobName, err := ap.InterpolateToString(ctx, rj.Job.Name)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate job name %s into a string: %v", rj.JobID, rj.Job.Name, err),
			}, false
		}
		rj.Job.Name = jobName
	}

	if strings.HasPrefix(rj.Job.Region, "${{") {
		reg, err := ap.InterpolateToString(ctx, rj.Job.Region)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate region %s into a string: %v", rj.JobID, rj.Job.Region, err),
			}, false
		}
		rj.Region = reg
		rj.Job.Region = reg
	}

	// Check user region right.
	if rj.Region == "" {
		rj.Job.Region = defaultRegion
		rj.Region = defaultRegion
	}
	jobInfoMsg, has := regionPermCache[rj.Region]
	if !has {
		var err error
		jobInfoMsg, err = checkUserRegionRight(ctx, db, rj, wrEnqueue, rj.Region)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("job %s: unable to check right for user %s: %v", rj.JobID, rj.Initiator.Username(), err),
			}, false
		}
		regionPermCache[rj.Region] = jobInfoMsg
	}

	if jobInfoMsg != nil {
		rj.Status = sdk.V2WorkflowRunJobStatusSkipped
		return jobInfoMsg, false
	}

	if strings.Contains(rj.Job.RunsOn.Model, "${{") {
		model, err := ap.InterpolateToString(ctx, rj.Job.RunsOn.Model)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s into a string: %v", rj.JobID, rj.Job.RunsOn.Model, err),
			}, false
		}
		completeName, msg, err := wref.checkWorkerModel(ctx, db, store, rj.JobID, model, rj.Region, "")
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("%v", err),
			}, false
		}
		if msg != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          msg.Message,
			}, false
		}
		if strings.HasPrefix(model, ".cds/worker-models/") {
			rj.ModelType = wref.ef.localWorkerModelCache[model].Model.Type
			rj.ModelOSArch = wref.ef.localWorkerModelCache[model].Model.OSArch
		} else {
			rj.ModelType = wref.ef.workerModelCache[completeName].Model.Type
			rj.ModelOSArch = wref.ef.workerModelCache[completeName].Model.OSArch
		}
		rj.Job.RunsOn.Model = completeName
	}
	if strings.HasPrefix(rj.Job.RunsOn.Flavor, "${{") {
		flavor, err := ap.InterpolateToString(ctx, rj.Job.RunsOn.Flavor)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s into a string: %v", rj.JobID, rj.Job.RunsOn.Flavor, err),
			}, false
		}
		rj.Job.RunsOn.Flavor = flavor
	}
	if strings.HasPrefix(rj.Job.RunsOn.Memory, "${{") {
		mem, err := ap.InterpolateToString(ctx, rj.Job.RunsOn.Memory)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s into a string: %v", rj.JobID, rj.Job.RunsOn.Memory, err),
			}, false
		}
		rj.Job.RunsOn.Memory = mem
	}

	for _, def := range wref.ef.localWorkerModelCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.ef.currentVCS.Name, wref.ef.currentRepo.Name, def.Model.Name, wref.run.WorkflowRef)
		if _, has := run.WorkflowData.WorkerModels[completeName]; !has {
			runUpdated = true
			run.WorkflowData.WorkerModels[completeName] = def.Model
		}
	}
	for name, def := range wref.ef.workerModelCache {
		if _, has := run.WorkflowData.WorkerModels[name]; !has {
			runUpdated = true
			run.WorkflowData.WorkerModels[name] = def.Model
		}
	}

	return nil, runUpdated
}

func prepareRunJobs(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wref *WorkflowRunEntityFinder, run *sdk.V2WorkflowRun, existingRunJobs []sdk.V2WorkflowRunJob, runVarsetCtx map[string]interface{}, wrEnqueue sdk.V2WorkflowRunEnqueue, jobsToQueue map[string]JobToTrigger, runJobsContexts sdk.JobsResultContext, defaultRegion string) ([]sdk.V2WorkflowRunJob, map[string]sdk.V2WorkflowRunJobInfo, []sdk.V2WorkflowRunInfo, bool, error) {
	runJobs := make([]sdk.V2WorkflowRunJob, 0)
	runJobsInfo := make(map[string]sdk.V2WorkflowRunJobInfo)
	hasToUpdateRun := false

	regionPermCache := make(map[string]*sdk.V2WorkflowRunJobInfo)

	// Browse job to queue and compute data ( matrix / region / model etc..... )
	for jobID, jobToTrigger := range jobsToQueue {
		jobDef := jobToTrigger.Job

		// If no step && no template: rj is success
		if (jobToTrigger.Status.IsTerminated() && jobToTrigger.Job.From != "") || (len(jobDef.Steps) == 0 && jobDef.From == "") {
			runJob := sdk.V2WorkflowRunJob{
				WorkflowRunID:      run.ID,
				Status:             sdk.V2WorkflowRunJobStatusSuccess,
				JobID:              jobID,
				Job:                jobDef,
				DeprecatedUserID:   wrEnqueue.Initiator.UserID,
				DeprecatedUsername: wrEnqueue.Initiator.Username(),
				DeprecatedAdminMFA: wrEnqueue.Initiator.IsAdminWithMFA,
				ProjectKey:         run.ProjectKey,
				VCSServer:          run.VCSServer,
				Repository:         run.Repository,
				Region:             jobDef.Region,
				WorkflowName:       run.WorkflowName,
				RunNumber:          run.RunNumber,
				RunAttempt:         run.RunAttempt,
				Initiator:          wrEnqueue.Initiator,
			}
			if jobToTrigger.Status.IsTerminated() {
				runJob.Status = jobToTrigger.Status
			}
			runJobs = append(runJobs, runJob)
			continue
		}

		runJobContext := sdk.WorkflowRunJobsContext{
			WorkflowRunContext: sdk.WorkflowRunContext{
				CDS: run.Contexts.CDS,
				Git: run.Contexts.Git,
				Env: run.Contexts.Env,
			},
			Jobs:  runJobsContexts,
			Vars:  make(map[string]interface{}),
			Needs: sdk.NeedsContext{},
		}

		// Add vars context
		vss := make([]sdk.ProjectVariableSet, 0)
		for _, vs := range jobDef.VariableSets {
			if _, has := runVarsetCtx[vs]; !has {
				vsDB, err := project.LoadVariableSetByName(ctx, db, run.ProjectKey, vs)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, nil, nil, hasToUpdateRun, err
				}
				// If not found stop the run
				if err != nil {
					msg := sdk.V2WorkflowRunInfo{
						WorkflowRunID: run.ID,
						IssuedAt:      time.Now(),
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("variable set %s not found on project", vs),
					}
					return nil, nil, []sdk.V2WorkflowRunInfo{msg}, hasToUpdateRun, nil
				}
				vsDB.Items, err = project.LoadVariableSetAllItem(ctx, db, vsDB.ID)
				if err != nil {
					return nil, nil, nil, hasToUpdateRun, err
				}
				vss = append(vss, *vsDB)
			}
		}
		jobVarsCtx, _, err := buildVarsContext(ctx, vss)
		if err != nil {
			return nil, nil, nil, false, err
		}
		for k, v := range runVarsetCtx {
			jobVarsCtx[k] = v
		}
		runJobContext.Vars = jobVarsCtx

		// Add need context
		for _, n := range jobDef.Needs {
			if jobCtxData, has := runJobsContexts[n]; has {
				runJobContext.Needs[n] = sdk.NeedContext{
					Result:  jobCtxData.Result,
					Outputs: jobCtxData.Outputs,
				}
			}
		}

		// Compute job matrix strategy
		matrixPermutation, msInfo := generateMatrixPermutation(ctx, runJobContext, run, jobDef)
		if msInfo != nil {
			return nil, nil, []sdk.V2WorkflowRunInfo{*msInfo}, false, err
		}

		if len(matrixPermutation) == 0 {
			runJob := sdk.V2WorkflowRunJob{
				ID:                 sdk.UUID(),
				WorkflowRunID:      run.ID,
				Status:             jobToTrigger.Status,
				JobID:              jobID,
				Job:                jobDef,
				DeprecatedUserID:   wrEnqueue.Initiator.UserID,
				DeprecatedUsername: wrEnqueue.Initiator.Username(),
				DeprecatedAdminMFA: wrEnqueue.Initiator.IsAdminWithMFA,
				ProjectKey:         run.ProjectKey,
				VCSServer:          run.VCSServer,
				Repository:         run.Repository,
				Region:             jobDef.Region,
				WorkflowName:       run.WorkflowName,
				RunNumber:          run.RunNumber,
				RunAttempt:         run.RunAttempt,
				Initiator:          wrEnqueue.Initiator,
			}
			// If the current job was a matrix, skip it
			if jobDef.Strategy != nil && len(jobDef.Strategy.Matrix) > 0 {
				runJob.Status = sdk.V2WorkflowRunJobStatusSkipped
				runJobsInfo[runJob.ID] = sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    runJob.WorkflowRunID,
					WorkflowRunJobID: runJob.ID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelWarning,
					Message:          "found an empty matrix, skipping the job",
				}
			} else {
				if jobDef.RunsOn.Model != "" {
					runJob.ModelType = run.WorkflowData.WorkerModels[jobDef.RunsOn.Model].Type
					runJob.ModelOSArch = run.WorkflowData.WorkerModels[jobDef.RunsOn.Model].OSArch
				}
				for _, jobEvent := range run.RunJobEvent {
					if jobEvent.RunAttempt != run.RunAttempt {
						continue
					}
					if jobEvent.JobID != runJob.JobID {
						continue
					}
					runJob.GateInputs = jobEvent.Inputs
				}
				runJobContext.Gate = runJob.GateInputs
				runJobInfo, runUpdated := computeRunJobsInterpolation(ctx, db, store, wref, run, &runJob, defaultRegion, regionPermCache, runJobContext, wrEnqueue)
				if runJobInfo != nil {
					runJobsInfo[runJob.ID] = *runJobInfo
				}

				if runUpdated {
					hasToUpdateRun = runUpdated
				}
			}
			// Manage concurrency
			runJobInfo, err := manageJobConcurrency(ctx, db, *run, jobID, &runJob)
			if err != nil {
				return nil, nil, nil, hasToUpdateRun, err
			}
			if runJobInfo != nil {
				runJobsInfo[runJob.ID] = *runJobInfo
				runJob.Status = sdk.V2WorkflowRunJobStatusBlocked
			}
			runJobs = append(runJobs, runJob)
		} else {
			allVariableSets, err := project.LoadVariableSetsByProject(ctx, db, proj.Key)
			if err != nil {
				return nil, nil, nil, false, err
			}

			jobData := prepareJobData{
				wrEnqueue:       wrEnqueue,
				runJobContext:   runJobContext,
				existingRunJobs: existingRunJobs,
				jobID:           jobID,
				jobToTrigger: JobToTrigger{
					Status: jobToTrigger.Status,
					Job:    jobDef,
				},
				defaultRegion:   defaultRegion,
				regionPermCache: regionPermCache,
				allVariableSets: allVariableSets,
			}
			if jobDef.From == "" {
				// ///////
				// TODO - manage concurrency on matrixed job
				// ///////
				jobs, runUpdated := createMatrixedRunJobs(ctx, db, store, wref, matrixPermutation, runJobsInfo, run, jobData)
				runJobs = append(runJobs, jobs...)
				if runUpdated {
					hasToUpdateRun = true
				}
			} else {
				// For templated matrixed job, we only create new jobs on the parent worklow
				// With hasToUpdateRun the workflow run will be saved to update his definition
				// With previous computed flag  `hasTemplatedMatrixedJob` the workflow will be retriggered in the workflow engine
				msgInfo := createTemplatedMatrixedJobs(ctx, db, store, wref, matrixPermutation, run, jobData)
				hasToUpdateRun = true
				if len(msgInfo) > 0 {
					return nil, nil, msgInfo, false, nil
				}
			}
		}
	}

	// Browse blocked job release then if we can
	for _, rj := range existingRunJobs {
		if rj.Status != sdk.StatusBlocked {
			continue
		}
		rjToUnblocked, err := retrieveRunJobToUnblocked(ctx, db, rj)
		if err != nil {
			return nil, nil, nil, false, err
		}
		if rjToUnblocked != nil && rjToUnblocked.ID == rj.ID {

			runJobsInfo[rj.ID] = sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    rj.WorkflowRunID,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelInfo,
				Message:          "Job has been unlocked",
			}

			rj.Queued = time.Now()
			rj.Status = sdk.V2WorkflowRunJobStatusWaiting
			runJobs = append(runJobs, rj)
		}
	}

	return runJobs, runJobsInfo, nil, hasToUpdateRun, nil
}

func createTemplatedMatrixedJobs(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, matrixPermutation []map[string]string, run *sdk.V2WorkflowRun, data prepareJobData) []sdk.V2WorkflowRunInfo {
	newJobs := make(map[string]sdk.V2Job)
	newStages := make(map[string]sdk.WorkflowStage)
	newGates := make(map[string]sdk.V2JobGate)
	newAnnotations := make(map[string]string)
	var entityTemplateWithObj *sdk.EntityWithObject
	for _, m := range matrixPermutation {
		data.runJobContext.Matrix = make(map[string]string)
		for k, v := range m {
			data.runJobContext.Matrix[k] = v
		}
		bts, _ := json.Marshal(data.runJobContext)
		var mapContexts map[string]interface{}
		if err := json.Unmarshal(bts, &mapContexts); err != nil {
			return []sdk.V2WorkflowRunInfo{{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("Job %s: unable to build context to compute job with permutation %v: %v", data.jobID, m, err),
			}}
		}

		ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)

		// Interpolate template parameters in case of matrix variable usage
		interpolatedParams := make(map[string]string)
		for k, p := range data.jobToTrigger.Job.Parameters {
			value, err := ap.InterpolateToString(ctx, p)
			if err != nil {
				return []sdk.V2WorkflowRunInfo{{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					IssuedAt:      time.Now(),
					Message:       fmt.Sprintf("Job %s: unable to interpolate into a string job parameter %s: %v", data.jobID, p, err),
				}}
			}
			interpolatedParams[k] = value
		}

		entityTemplate, tmpWorkflow, msgs, err := checkJobTemplate(ctx, db, store, wref, data.jobToTrigger.Job, run, interpolatedParams)
		if err != nil {
			return []sdk.V2WorkflowRunInfo{{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("Job %s: unable to build job from template: %v", data.jobID, err),
			}}
		}
		if len(msgs) > 0 {
			return msgs
		}
		entityTemplateWithObj = entityTemplate

		for k, v := range tmpWorkflow.Jobs {
			if _, has := newJobs[k]; has {
				return []sdk.V2WorkflowRunInfo{{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					IssuedAt:      time.Now(),
					Message:       fmt.Sprintf("Job %s: there is more than one job with this name", data.jobID),
				}}
			}
			newJobs[k] = v
		}
		for k, v := range tmpWorkflow.Stages {
			if _, has := newStages[k]; !has {
				newStages[k] = v
			}
		}
		for k, v := range tmpWorkflow.Gates {
			if _, has := newGates[k]; !has {
				newGates[k] = v
			}
		}
		for k, v := range tmpWorkflow.Annotations {
			if _, has := newAnnotations[k]; !has {
				newAnnotations[k] = v
			}
		}
	}

	msgs, err := handleTemplatedJobInWorkflow(ctx, db, store, wref, entityTemplateWithObj, run, newJobs, newStages, newGates, newAnnotations, data.jobID, data.jobToTrigger.Job, data.allVariableSets, data.defaultRegion)
	if err != nil {

	}
	if len(msgs) > 0 {
		return msgs
	}

	// Remove templated job
	delete(run.WorkflowData.Workflow.Jobs, data.jobID)

	// Check usage of current job stage
	if data.jobToTrigger.Job.Stage != "" {
		stageUsed := false
		for _, j := range run.WorkflowData.Workflow.Jobs {
			if j.Stage == data.jobToTrigger.Job.Stage {
				stageUsed = true
			}
		}
		if !stageUsed {
			for _, stage := range run.WorkflowData.Workflow.Stages {
				if slices.Contains(stage.Needs, data.jobToTrigger.Job.Stage) {
					stageUsed = true
				}
			}
		}
		if !stageUsed {
			delete(run.WorkflowData.Workflow.Stages, data.jobToTrigger.Job.Stage)
		}
	}

	msgsLint := make([]sdk.V2WorkflowRunInfo, 0)
	errs := run.WorkflowData.Workflow.Lint()
	for _, e := range errs {
		msgsLint = append(msgsLint, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			IssuedAt:      time.Now(),
			Message:       e.Error(),
		})
	}

	return msgsLint
}

func createMatrixedRunJobs(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, matrixPermutation []map[string]string, runJobsInfo map[string]sdk.V2WorkflowRunJobInfo, run *sdk.V2WorkflowRun, data prepareJobData) ([]sdk.V2WorkflowRunJob, bool) {
	runJobs := make([]sdk.V2WorkflowRunJob, 0)
	hasToUpdateRun := false

	// Check permutation to trigger
	permutations := searchPermutationToTrigger(ctx, matrixPermutation, data.existingRunJobs, data.jobID)
	for _, m := range permutations {
		permJobDef := data.jobToTrigger.Job.Copy()
		runJob := sdk.V2WorkflowRunJob{
			ID:                 sdk.UUID(),
			WorkflowRunID:      run.ID,
			Status:             data.jobToTrigger.Status,
			JobID:              data.jobID,
			Job:                permJobDef,
			DeprecatedUserID:   data.wrEnqueue.Initiator.UserID,
			DeprecatedUsername: data.wrEnqueue.Initiator.Username(),
			DeprecatedAdminMFA: data.wrEnqueue.Initiator.IsAdminWithMFA,
			ProjectKey:         run.ProjectKey,
			VCSServer:          run.VCSServer,
			Repository:         run.Repository,
			Region:             permJobDef.Region,
			WorkflowName:       run.WorkflowName,
			RunNumber:          run.RunNumber,
			RunAttempt:         run.RunAttempt,
			Matrix:             sdk.JobMatrix{},
			Initiator:          data.wrEnqueue.Initiator,
		}
		for k, v := range m {
			runJob.Matrix[k] = v
		}
		if permJobDef.RunsOn.Model != "" {
			runJob.ModelType = run.WorkflowData.WorkerModels[permJobDef.RunsOn.Model].Type
			runJob.ModelOSArch = run.WorkflowData.WorkerModels[permJobDef.RunsOn.Model].OSArch
		}
		for _, jobEvent := range run.RunJobEvent {
			if jobEvent.RunAttempt != run.RunAttempt {
				continue
			}
			if jobEvent.JobID != runJob.JobID {
				continue
			}
			runJob.GateInputs = jobEvent.Inputs
		}
		data.runJobContext.Gate = runJob.GateInputs
		data.runJobContext.Matrix = runJob.Matrix
		runJobInfo, runUpdated := computeRunJobsInterpolation(ctx, db, store, wref, run, &runJob, data.defaultRegion, data.regionPermCache, data.runJobContext, data.wrEnqueue)
		if runJobInfo != nil {
			runJobsInfo[runJob.ID] = *runJobInfo
		}
		if runUpdated {
			hasToUpdateRun = runUpdated
		}
		runJobs = append(runJobs, runJob)
	}
	return runJobs, hasToUpdateRun
}

func searchPermutationToTrigger(ctx context.Context, permutations []map[string]string, runJobs []sdk.V2WorkflowRunJob, jobID string) []map[string]string {
	runJobsForJobID := make([]sdk.V2WorkflowRunJob, 0)

	for _, rj := range runJobs {
		if rj.JobID == jobID {
			runJobsForJobID = append(runJobsForJobID, rj)
		}
	}

	if len(runJobsForJobID) == 0 {
		return permutations
	}

	permutationToTrigger := make([]map[string]string, 0)
	// Browse all permutation
	for _, perm := range permutations {
		runJobFound := false
		// Search if it has been already trigger
	runJobLoop:
		for _, rj := range runJobsForJobID {
			for k, v := range perm {
				// If not the same permutation, check next run job
				if rj.Matrix[k] != v {
					continue runJobLoop
				}
			}
			runJobFound = true
			break
		}
		if !runJobFound {
			permutationToTrigger = append(permutationToTrigger, perm)
		}
	}
	return permutationToTrigger
}

func generateMatrixPermutation(ctx context.Context, rootJobContext sdk.WorkflowRunJobsContext, run *sdk.V2WorkflowRun, jobDef sdk.V2Job) ([]map[string]string, *sdk.V2WorkflowRunInfo) {
	keys := make([]string, 0)
	interpolatedMatrix := make(map[string][]string)
	if jobDef.Strategy != nil && len(jobDef.Strategy.Matrix) > 0 {
		bts, _ := json.Marshal(rootJobContext)
		var mapContexts map[string]interface{}
		_ = json.Unmarshal(bts, &mapContexts) // error cannot happen here

		ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)

		for k, v := range jobDef.Strategy.Matrix {
			keys = append(keys, k)

			matrixValues := make([]string, 0)
			if slice, ok := v.([]interface{}); ok {
				for _, sliceValue := range slice {
					valueString, ok := sliceValue.(string)
					if !ok {
						msg := &sdk.V2WorkflowRunInfo{
							WorkflowRunID: run.ID,
							IssuedAt:      time.Now(),
							Level:         sdk.WorkflowRunInfoLevelError,
							Message:       fmt.Sprintf("matrix value %v is not a string", sliceValue),
						}
						return nil, msg
					}

					interpolatedValue, err := ap.InterpolateToString(ctx, valueString)
					if err != nil {
						msg := &sdk.V2WorkflowRunInfo{
							WorkflowRunID: run.ID,
							IssuedAt:      time.Now(),
							Level:         sdk.WorkflowRunInfoLevelError,
							Message:       fmt.Sprintf("unable to interpolate matrix value %s: %v", valueString, err),
						}
						return nil, msg
					}
					matrixValues = append(matrixValues, interpolatedValue)
				}
			} else if valueString, ok := v.(string); ok {
				interpolatedValue, err := ap.Interpolate(ctx, valueString)
				if err != nil {
					msg := &sdk.V2WorkflowRunInfo{
						WorkflowRunID: run.ID,
						IssuedAt:      time.Now(),
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("unable to interpolate %s: %v", valueString, err),
					}
					return nil, msg
				}
				interpoaltedSlice, ok := interpolatedValue.([]interface{})
				if !ok {
					msg := &sdk.V2WorkflowRunInfo{
						WorkflowRunID: run.ID,
						IssuedAt:      time.Now(),
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("interpolated matrix is not a string slice, got %T", interpolatedValue),
					}
					return nil, msg
				}
				stringsValue := make([]string, 0, len(interpoaltedSlice))
				for _, vle := range interpoaltedSlice {
					stringsValue = append(stringsValue, fmt.Sprintf("%v", vle))
				}
				matrixValues = stringsValue
			} else {
				msg := &sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("unable to use matrix key %s of type %T", k, v),
				}
				return nil, msg
			}
			interpolatedMatrix[k] = matrixValues

		}
	}

	alls := make([]map[string]string, 0)
	if jobDef.Strategy != nil && len(interpolatedMatrix) > 0 {
		generateMatrix(interpolatedMatrix, keys, 0, make(map[string]string), &alls)
		for k := range interpolatedMatrix {
			jobDef.Strategy.Matrix[k] = interpolatedMatrix[k]
		}
	}

	return alls, nil
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

type JobToTrigger struct {
	Status      sdk.V2WorkflowRunJobStatus
	Job         sdk.V2Job
	Concurrency *sdk.V2RunJobConcurrency
}

// Return jobToQueue, skippedJob, runInfos, error
func retrieveJobToQueue(ctx context.Context, db *gorp.DbMap, wrEnqueue sdk.V2WorkflowRunEnqueue, run *sdk.V2WorkflowRun, runJobs []sdk.V2WorkflowRunJob, allrunJobsMap map[string]sdk.V2WorkflowRunJob, runJobsContexts sdk.JobsResultContext, defaultRegion string) (map[string]JobToTrigger, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "retrieveJobToQueue")
	defer next()
	runInfos := make([]sdk.V2WorkflowRunInfo, 0)
	jobToQueue := make(map[string]JobToTrigger)

	// Select jobs to check ( all workflow or list of jobs from enqueue request )
	jobsToCheck := make(map[string]sdk.V2Job)

	for jobID, jobDef := range run.WorkflowData.Workflow.Jobs {
		// Do not enqueue jobs that have already a run
		runJobMapItem, has := allrunJobsMap[jobID]

		if !has {
			jobsToCheck[jobID] = jobDef
		} else {
			// If job with matrix, check if we have to rerun a permmutation
			if runJobMapItem.Job.Strategy != nil && len(runJobMapItem.Job.Strategy.Matrix) > 0 {

				// If runjob has a status && a template, ignore it. A matrix job can be run it template has been resolved
				if runJobMapItem.Job.From != "" {
					continue
				}

				nbPermutations := 1
				for _, v := range runJobMapItem.Job.Strategy.Matrix {
					if vString, ok := v.([]string); ok {
						nbPermutations *= len(vString)
					} else if vInterface, ok := v.([]interface{}); ok {
						nbPermutations *= len(vInterface)
					}
				}
				runPermutations := 0
				for _, rj := range runJobs {
					if rj.JobID == runJobMapItem.JobID {
						runPermutations++
					}
				}
				// If there is still permutation to run
				if nbPermutations > runPermutations {
					jobsToCheck[jobID] = jobDef
				}
			}

		}
	}

	// Check jobs : Needs / Condition / User Right
	stages := run.GetStages()
	if len(stages) > 0 {
		for k, j := range allrunJobsMap {
			stageName := run.WorkflowData.Workflow.Jobs[k].Stage
			jobInStage := stages[stageName].Jobs[k]
			jobInStage.Status = j.Status
			stages[stageName].Jobs[k] = jobInStage
		}
		stages.ComputeStatus()
	}

	for jobID, jobDef := range jobsToCheck {

		// Skip the job in stage that cannot be run
		if len(stages) > 0 && !stages[jobDef.Stage].CanBeRun {
			continue
		}

		// Check jobs needs
		if ok := checkJobNeeds(runJobsContexts, jobDef); !ok {
			continue
		}

		// Build job context
		jobContext := buildContextForJob(ctx, run.WorkflowData.Workflow, runJobsContexts, run.Contexts, stages, jobID)

		canBeQueued, infos, err := checkJob(ctx, db, wrEnqueue, *run, jobID, &jobDef, jobContext)
		runInfos = append(runInfos, infos...)
		if err != nil {
			jobToQueue[jobID] = JobToTrigger{
				Status: sdk.V2WorkflowRunJobStatusSkipped,
				Job:    jobDef,
			}
			return nil, runInfos, err
		}
		if !canBeQueued {
			jobToQueue[jobID] = JobToTrigger{
				Status: sdk.V2WorkflowRunJobStatusSkipped,
				Job:    jobDef,
			}
			continue
		}

		jobToQueue[jobID] = JobToTrigger{
			Status: sdk.V2WorkflowRunJobStatusWaiting,
			Job:    jobDef,
		}
	}

	return jobToQueue, runInfos, nil
}

func checkJob(ctx context.Context, db gorp.SqlExecutor, wrEnqueue sdk.V2WorkflowRunEnqueue, run sdk.V2WorkflowRun, jobID string, jobDef *sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext) (bool, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "checkJob", trace.StringAttribute(telemetry.TagJob, jobID))
	defer next()

	runInfos := make([]sdk.V2WorkflowRunInfo, 0)

	// check varset right
	if !wrEnqueue.Initiator.IsAdminWithMFA && !wrEnqueue.DeprecatedIsAdminWithMFA {
		varsets := append(run.WorkflowData.Workflow.VariableSets, jobDef.VariableSets...)

		if wrEnqueue.Initiator.IsUser() {
			has, vInError, err := rbac.HasRoleOnVariableSetsAndUserID(ctx, db, sdk.VariableSetRoleUse, wrEnqueue.Initiator.UserID, run.ProjectKey, varsets)
			if err != nil {
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("job %s: unable to check right for user %s on varset %v: %v", jobID, wrEnqueue.Initiator.Username(), varsets, err),
				})
				return false, runInfos, nil
			}
			if !has {
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelWarning,
					Message:       fmt.Sprintf("job %s: user %s does not have enough right on varset %s", jobID, wrEnqueue.Initiator.Username(), vInError),
				})
				return false, runInfos, nil
			}
		} else {
			has, vInError, err := rbac.HasRoleOnVariableSetsAndVCSUser(ctx, db, sdk.VariableSetRoleUse, sdk.RBACVCSUser{VCSServer: wrEnqueue.Initiator.VCS, VCSUsername: wrEnqueue.Initiator.VCSUsername}, run.ProjectKey, varsets)
			if err != nil {
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("job %s: unable to check right for user %s on varset %v: %v", jobID, wrEnqueue.Initiator.Username(), varsets, err),
				})
				return false, runInfos, nil
			}
			if !has {
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelWarning,
					Message:       fmt.Sprintf("job %s: user %s does not have enough right on varset %s", jobID, wrEnqueue.Initiator.Username(), vInError),
				})
				return false, runInfos, nil
			}
		}
	}

	// Retrieve inputs from an event if exists
	var inputs map[string]interface{}
	for _, je := range run.RunJobEvent {
		if je.RunAttempt == run.RunAttempt && je.JobID == jobID {
			inputs = je.Inputs
			break
		}
	}

	// check job condition
	canRun, err := checkCanRunJob(ctx, db, run, inputs, *jobDef, currentJobContext, wrEnqueue.Initiator)
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
			Message:       fmt.Sprintf("Job %q: The condition is not satisfied", jobID),
		})
	}
	return canRun, runInfos, nil
}

func computeRunStatusFromJobsStatus(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempt int64, nbJob int) (sdk.V2WorkflowRunStatus, error) {
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, db, runID, runAttempt)
	if err != nil {
		return "", err
	}

	finalStatus := sdk.V2WorkflowRunStatusSuccess
	allJobID := make(map[string]struct{})

	for _, rj := range runJobs {
		if rj.Status == sdk.V2WorkflowRunJobStatusFail && finalStatus != sdk.V2WorkflowRunStatusStopped && finalStatus.IsTerminated() && !rj.Job.ContinueOnError {
			finalStatus = sdk.V2WorkflowRunStatusFail
		}
		if rj.Status == sdk.V2WorkflowRunJobStatusStopped && finalStatus.IsTerminated() {
			finalStatus = sdk.V2WorkflowRunStatusStopped
		}
		if !rj.Status.IsTerminated() {
			finalStatus = sdk.V2WorkflowRunStatusBuilding
		}
		allJobID[rj.JobID] = struct{}{}
	}

	if len(allJobID) < nbJob && finalStatus == sdk.V2WorkflowRunStatusSuccess {
		finalStatus = sdk.V2WorkflowRunStatusBuilding
	}
	return finalStatus, nil
}

// Check and set default region on job
func checkUserRegionRight(ctx context.Context, db gorp.SqlExecutor, rj *sdk.V2WorkflowRunJob, wrEnqueue sdk.V2WorkflowRunEnqueue, regionName string) (*sdk.V2WorkflowRunJobInfo, error) {
	ctx, next := telemetry.Span(ctx, "checkUserRegionRight")
	defer next()

	wantedRegion, err := region.LoadRegionByName(ctx, db, regionName)
	if err != nil {
		return nil, err
	}

	// Check if project has the right to execute job on region
	projectHasPerm, err := rbac.HasRoleOnRegionProject(ctx, db, sdk.RegionRoleExecute, wantedRegion.ID, rj.ProjectKey)
	if err != nil {
		return nil, err
	}
	if !projectHasPerm {
		return &sdk.V2WorkflowRunJobInfo{
			WorkflowRunID:    rj.WorkflowRunID,
			Level:            sdk.WorkflowRunInfoLevelError,
			WorkflowRunJobID: rj.ID,
			IssuedAt:         time.Now(),
			Message:          fmt.Sprintf("job %s: project %s is not allowed to start job on region %q", rj.JobID, rj.ProjectKey, rj.Region),
		}, nil
	}

	if !wrEnqueue.Initiator.IsAdminWithMFA {
		if wrEnqueue.Initiator.IsUser() {
			allowedRegions, err := rbac.LoadRegionIDsByRoleAndUserID(ctx, db, sdk.RegionRoleExecute, wrEnqueue.Initiator.UserID)
			if err != nil {
				next()
				return nil, err
			}
			next()
			for _, r := range allowedRegions {
				if r.RegionID == wantedRegion.ID {
					return nil, nil
				}
			}
		} else {
			allowedRegions, err := rbac.LoadRegionIDsByRoleAndVCSUSer(ctx, db, sdk.RegionRoleExecute, sdk.RBACVCSUser{VCSServer: wrEnqueue.Initiator.VCS, VCSUsername: wrEnqueue.Initiator.VCSUsername})
			if err != nil {
				next()
				return nil, err
			}
			next()
			for _, r := range allowedRegions {
				if r.RegionID == wantedRegion.ID {
					return nil, nil
				}
			}
		}
	} else {
		return nil, nil
	}

	return &sdk.V2WorkflowRunJobInfo{
		WorkflowRunID:    rj.WorkflowRunID,
		Level:            sdk.WorkflowRunInfoLevelError,
		WorkflowRunJobID: rj.ID,
		IssuedAt:         time.Now(),
		Message:          fmt.Sprintf("job %s: user %s does not have enough right on region %q", rj.JobID, wrEnqueue.Initiator.Username(), rj.Region),
	}, nil
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

func checkCanRunJob(ctx context.Context, db gorp.SqlExecutor, run sdk.V2WorkflowRun, jobInputs map[string]interface{}, jobDef sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext, initiator sdk.V2Initiator) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkJobCondition")
	defer next()

	// Check job Gate
	if jobDef.Gate != "" {
		gate := run.WorkflowData.Workflow.Gates[jobDef.Gate]

		// Check reviewers
		reviewersChecked := len(gate.Reviewers.Users) == 0 && len(gate.Reviewers.Groups) == 0
		if len(gate.Reviewers.Users) > 0 {
			reviewersChecked = sdk.IsInArray(initiator.Username(), gate.Reviewers.Users)
		}
		if !reviewersChecked {
			for _, g := range gate.Reviewers.Groups {
				grp, err := group.LoadByName(ctx, db, g, group.LoadOptions.WithMembers)
				if err != nil {
					return false, err
				}
				reviewersChecked = sdk.IsInArray(initiator.UserID, grp.Members.UserIDs())
				if reviewersChecked {
					break
				}
			}
		}
		if !reviewersChecked && !initiator.IsAdminWithMFA {
			return false, nil
		}

		// Create empty input context to be able to interpolate gate condition.
		currentJobContext.Gate = make(map[string]interface{})
		for k, v := range gate.Inputs {
			if v.Default != nil {
				currentJobContext.Gate[k] = v.Default
			} else {
				if v.Options != nil {
					currentJobContext.Gate[k] = make([]interface{}, 0)
				} else {
					switch v.Type {
					case "boolean":
						currentJobContext.Gate[k] = false
					case "number":
						currentJobContext.Gate[k] = 0
					default:
						currentJobContext.Gate[k] = ""
					}
				}

			}
		}

		// Override with value sent by user
		for k, v := range jobInputs {
			if _, has := currentJobContext.Gate[k]; has {

				// Check user gate inputs
				gateDefInput := gate.Inputs[k]
				if gateDefInput.Options != nil {
					if !gateDefInput.Options.Multiple {
						valueFound := false
						for _, possibleValue := range gateDefInput.Options.Values {
							if possibleValue == v {
								valueFound = true
								break
							}
						}
						if !valueFound {
							return false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "gate input %s with value %v doesn't match %v", k, v, gateDefInput.Options.Values)
						}
					} else {
						sliceValues, ok := v.([]interface{})
						if !ok {
							return false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "gate input %s with value %v is not an array, got %T", k, v, v)
						}
						for _, sv := range sliceValues {
							valueFound := false
							for _, inputValue := range gateDefInput.Options.Values {
								if inputValue == sv {
									valueFound = true
									break
								}
							}
							if !valueFound {
								return false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "gate input %s with value %v doesn't match %v", k, v, gateDefInput.Options.Values)
							}
						}
					}
				}

				currentJobContext.Gate[k] = v
			}
		}

		gateConditionResult, err := checkCondition(ctx, gate.If, currentJobContext)
		if err != nil {
			return false, err
		}
		if !gateConditionResult {
			return false, nil
		}
		if jobDef.If == "" {
			return true, nil
		}
	}

	// Check Job IF
	jobIfResult, err := checkCondition(ctx, jobDef.If, currentJobContext)
	if err != nil {
		return false, err
	}
	return jobIfResult, nil
}

func checkCondition(ctx context.Context, condition string, currentJobContext sdk.WorkflowRunJobsContext) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkCondition")
	defer next()

	if condition == "" {
		condition = "${{success()}}"
	}
	if !strings.HasPrefix(condition, "${{") {
		condition = fmt.Sprintf("${{ %s }}", condition)
	}

	bts, err := json.Marshal(currentJobContext)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return false, sdk.WithStack(err)
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	booleanResult, err := ap.InterpolateToBool(ctx, condition)
	if err != nil {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to parse statement %s into a boolean: %v", condition, err)
	}
	return booleanResult, nil
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
	if wr.Status != sdk.V2WorkflowRunStatusBuilding {
		return nil
	}

	// Search last userID that trigger a job in this run
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
	if err != nil {
		return err
	}
	var lastJob sdk.V2WorkflowRunJob
	runningWorkflow := false
	hasBlockedRunJob := false
	for _, rj := range runJobs {
		if !rj.Status.IsTerminated() && rj.Status != sdk.V2WorkflowRunJobStatusBlocked {
			runningWorkflow = true
			continue
		}
		if rj.Status == sdk.V2WorkflowRunJobStatusBlocked {
			hasBlockedRunJob = true
			continue
		}
		if sdk.TimeSafe(rj.Started).After(sdk.TimeSafe(lastJob.Started)) {
			lastJob = rj
		}
	}

	if runningWorkflow && !hasBlockedRunJob {
		return nil
	}

	initiator := &lastJob.Initiator
	// No job have been triggered
	if initiator.Username() == "" {
		initiator = wr.Initiator
	}

	api.EnqueueWorkflowRun(ctx, wr.ID, *initiator, wr.WorkflowName, wr.RunNumber)
	return nil
}

func (api *API) EnqueueWorkflowRun(ctx context.Context, runID string, initiator sdk.V2Initiator, workflowName string, runNumber int64) {
	// Continue workflow
	enqueueRequest := sdk.V2WorkflowRunEnqueue{
		RunID:                    runID,
		Initiator:                initiator,
		DeprecatedUserID:         initiator.UserID,
		DeprecatedIsAdminWithMFA: initiator.IsAdminWithMFA,
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
