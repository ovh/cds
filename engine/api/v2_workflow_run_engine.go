package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
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
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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
		if err := api.workflowRunV2Trigger(ctx, wrEnqueue); err != nil {
			log.ErrorWithStackTrace(ctx, err)
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

	u, err := user.LoadByID(ctx, api.mustDB(), wrEnqueue.UserID)
	if err != nil {
		return err
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
	for _, rj := range allRunJobs {
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

	jobsToQueue, runMsgs, errRetrieve := retrieveJobToQueue(ctx, api.mustDB(), wrEnqueue, run, allRunJobs, allrunJobsMap, runJobsContexts, u, api.Config.Workflow.JobDefaultRegion)

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
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, allrunJobsMap, runResults, *u)
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
			}, allrunJobsMap, runResults, u)
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
		}, allrunJobsMap, runResults, u)
	}

	// Compute worker model / region on runJobs if needed
	wref := NewWorkflowRunEntityFinder(*proj, *run, *repo, *vcsServer, *u, wrEnqueue.IsAdminWithMFA, api.Config.WorkflowV2.LibraryProjectKey)
	wref.ef.repoCache[vcsServer.Name+"/"+repo.Name] = *repo
	wref.ef.vcsServerCache[vcsServer.Name] = *vcsServer

	// Enqueue JOB
	runJobs, runJobsInfos, errorMsg, runUpdated, err := prepareRunJobs(ctx, api.mustDB(), api.Cache, wref, run, allRunJobs, variableSetCtx, wrEnqueue, jobsToQueue, runJobsContexts, *u, api.Config.Workflow.JobDefaultRegion)
	if err != nil {
		return err
	}
	if errorMsg != nil {
		return failRunWithMessage(ctx, api.mustDB(), api.Cache, run, []sdk.V2WorkflowRunInfo{*errorMsg}, allrunJobsMap, runResults, u)
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
		if err := workflow_v2.InsertRunJob(ctx, tx, rj); err != nil {
			return err
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
				Message:          u.GetFullname() + " triggers manually this job",
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
		finalStatus, err := computeRunStatusFromJobsStatus(ctx, tx, run.ID, run.RunAttempt)
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
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, allrunJobsMap, runResults, *u)

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
					CDS:        run.Contexts.CDS,
					Git:        run.Contexts.Git,
					UserID:     run.UserID,
					UserName:   run.Username,
					Conclusion: string(run.Status),
					CreatedAt:  run.Started,
					Jobs:       make(map[string]sdk.HookWorkflowRunEventJob),
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

	if hasSkippedOrFailedJob || hasNoStepsJobs {
		// Re enqueue workflow to trigger job after
		api.EnqueueWorkflowRun(ctx, run.ID, wrEnqueue.UserID, run.WorkflowName, run.RunNumber, wrEnqueue.IsAdminWithMFA)
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

func failRunWithMessage(ctx context.Context, db *gorp.DbMap, cache cache.Store, run *sdk.V2WorkflowRun, msgs []sdk.V2WorkflowRunInfo, jobRunMap map[string]sdk.V2WorkflowRunJob, runResult []sdk.V2WorkflowRunResult, u *sdk.AuthentifiedUser) error {
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
	event_v2.PublishRunEvent(ctx, cache, sdk.EventRunEnded, *run, jobRunMap, runResult, *u)
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
		props.AddProperty("cds.version", run.Contexts.Git.SemverCurrent)
		signedProps["cds.version"] = run.Contexts.Git.SemverCurrent
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
		Version:                  run.Contexts.Git.SemverCurrent,
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

func computeRunJobsInterpolation(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, run *sdk.V2WorkflowRun, rj *sdk.V2WorkflowRunJob, defaultRegion string, regionPermCache map[string]*sdk.V2WorkflowRunJobInfo, wrEnqueue sdk.V2WorkflowRunEnqueue, u sdk.AuthentifiedUser) (*sdk.V2WorkflowRunJobInfo, bool) {
	runUpdated := false
	if rj.Status.IsTerminated() {
		return nil, false
	}

	computeModelCtx := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: run.Contexts,
		Matrix:             rj.Matrix,
		Gate:               rj.GateInputs,
	}
	bts, _ := json.Marshal(computeModelCtx)
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

	if strings.Contains(rj.Job.Name, "${{") {
		jobName, err := ap.InterpolateToString(ctx, rj.Job.Name)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s into a string: %v", rj.JobID, rj.Job.Name, err),
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
				Message:          fmt.Sprintf("Job %s: unable to interpolate %s into a string: %v", rj.JobID, rj.Job.Region, err),
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
		jobInfoMsg, err = checkUserRegionRight(ctx, db, rj, wrEnqueue, rj.Region, u)
		if err != nil {
			rj.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				Level:            sdk.WorkflowRunInfoLevelError,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("job %s: unable to check right for user %s: %v", rj.JobID, u.Username, err),
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
		} else {
			rj.ModelType = wref.ef.workerModelCache[completeName].Model.Type
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

func prepareRunJobs(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, run *sdk.V2WorkflowRun, existingRunJobs []sdk.V2WorkflowRunJob, runVarsetCtx map[string]interface{}, wrEnqueue sdk.V2WorkflowRunEnqueue, jobsToQueue map[string]JobToTrigger, runJobsContexts sdk.JobsResultContext, u sdk.AuthentifiedUser, defaultRegion string) ([]sdk.V2WorkflowRunJob, map[string]sdk.V2WorkflowRunJobInfo, *sdk.V2WorkflowRunInfo, bool, error) {
	runJobs := make([]sdk.V2WorkflowRunJob, 0)
	runJobsInfo := make(map[string]sdk.V2WorkflowRunJobInfo)
	hasToUpdateRun := false

	regionPermCache := make(map[string]*sdk.V2WorkflowRunJobInfo)

	rootJobContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: sdk.WorkflowRunContext{
			CDS: run.Contexts.CDS,
			Git: run.Contexts.Git,
			Env: run.Contexts.Env,
		},
		Jobs: runJobsContexts,
	}

	// Compute Matrix
	for jobID, jobToTrigger := range jobsToQueue {
		jobDef := jobToTrigger.Job
		if len(jobDef.Steps) == 0 {
			runJob := sdk.V2WorkflowRunJob{
				WorkflowRunID: run.ID,
				Status:        sdk.V2WorkflowRunJobStatusSuccess,
				JobID:         jobID,
				Job:           jobDef,
				UserID:        wrEnqueue.UserID,
				Username:      u.Username,
				AdminMFA:      wrEnqueue.IsAdminWithMFA,
				ProjectKey:    run.ProjectKey,
				Region:        jobDef.Region,
				WorkflowName:  run.WorkflowName,
				RunNumber:     run.RunNumber,
				RunAttempt:    run.RunAttempt,
			}
			runJobs = append(runJobs, runJob)
			continue
		}

		// Compute job matrix strategy
		keys := make([]string, 0)
		interpolatedMatrix := make(map[string][]string)
		if jobDef.Strategy != nil && len(jobDef.Strategy.Matrix) > 0 {
			vss := make([]sdk.ProjectVariableSet, 0)
			for _, vs := range jobDef.VariableSets {
				if _, has := runVarsetCtx[vs]; !has {
					vsDB, err := project.LoadVariableSetByName(ctx, db, run.ProjectKey, vs)
					if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
						return nil, nil, nil, hasToUpdateRun, err
					}
					// If not found stop the run
					if err != nil {
						msg := &sdk.V2WorkflowRunInfo{
							WorkflowRunID: run.ID,
							IssuedAt:      time.Now(),
							Level:         sdk.WorkflowRunInfoLevelError,
							Message:       fmt.Sprintf("variable set %s not found on project", vs),
						}
						return nil, nil, msg, hasToUpdateRun, nil
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
				return nil, nil, nil, hasToUpdateRun, err
			}
			for k, v := range runVarsetCtx {
				jobVarsCtx[k] = v
			}
			rootJobContext.Vars = jobVarsCtx
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
							return nil, nil, msg, hasToUpdateRun, nil
						}

						interpolatedValue, err := ap.InterpolateToString(ctx, valueString)
						if err != nil {
							msg := &sdk.V2WorkflowRunInfo{
								WorkflowRunID: run.ID,
								IssuedAt:      time.Now(),
								Level:         sdk.WorkflowRunInfoLevelError,
								Message:       fmt.Sprintf("unable to interpolate matrix value %s: %v", valueString, err),
							}
							return nil, nil, msg, hasToUpdateRun, nil
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
						return nil, nil, msg, hasToUpdateRun, nil
					}
					interpoaltedSlice, ok := interpolatedValue.([]interface{})
					if !ok {
						msg := &sdk.V2WorkflowRunInfo{
							WorkflowRunID: run.ID,
							IssuedAt:      time.Now(),
							Level:         sdk.WorkflowRunInfoLevelError,
							Message:       fmt.Sprintf("interpolated matrix is not a string slice, got %T", interpolatedValue),
						}
						return nil, nil, msg, hasToUpdateRun, nil

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
					return nil, nil, msg, hasToUpdateRun, nil
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

		if len(alls) == 0 {
			runJob := sdk.V2WorkflowRunJob{
				ID:            sdk.UUID(),
				WorkflowRunID: run.ID,
				Status:        jobToTrigger.Status,
				JobID:         jobID,
				Job:           jobDef,
				UserID:        wrEnqueue.UserID,
				Username:      u.Username,
				AdminMFA:      wrEnqueue.IsAdminWithMFA,
				ProjectKey:    run.ProjectKey,
				Region:        jobDef.Region,
				WorkflowName:  run.WorkflowName,
				RunNumber:     run.RunNumber,
				RunAttempt:    run.RunAttempt,
			}
			if jobDef.RunsOn.Model != "" {
				runJob.ModelType = run.WorkflowData.WorkerModels[jobDef.RunsOn.Model].Type
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
			runJobInfo, runUpdated := computeRunJobsInterpolation(ctx, db, store, wref, run, &runJob, defaultRegion, regionPermCache, wrEnqueue, u)
			if runJobInfo != nil {
				runJobsInfo[runJob.ID] = *runJobInfo
			}

			if runUpdated {
				hasToUpdateRun = runUpdated
			}

			runJobs = append(runJobs, runJob)
		} else {
			// Check permutation to trigger
			permutations := searchPermutationToTrigger(ctx, alls, existingRunJobs, jobID)
			for _, m := range permutations {
				runJob := sdk.V2WorkflowRunJob{
					ID:            sdk.UUID(),
					WorkflowRunID: run.ID,
					Status:        jobToTrigger.Status,
					JobID:         jobID,
					Job:           jobDef,
					UserID:        wrEnqueue.UserID,
					Username:      u.Username,
					AdminMFA:      wrEnqueue.IsAdminWithMFA,
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
				if jobDef.RunsOn.Model != "" {
					runJob.ModelType = run.WorkflowData.WorkerModels[jobDef.RunsOn.Model].Type
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
				runJobInfo, runUpdated := computeRunJobsInterpolation(ctx, db, store, wref, run, &runJob, defaultRegion, regionPermCache, wrEnqueue, u)
				if runJobInfo != nil {
					runJobsInfo[runJob.ID] = *runJobInfo
				}
				if runUpdated {
					hasToUpdateRun = runUpdated
				}
				runJobs = append(runJobs, runJob)
			}
		}
	}

	return runJobs, runJobsInfo, nil, hasToUpdateRun, nil
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
	Status sdk.V2WorkflowRunJobStatus
	Job    sdk.V2Job
}

// Return jobToQueue, skippedJob, runInfos, error
func retrieveJobToQueue(ctx context.Context, db *gorp.DbMap, wrEnqueue sdk.V2WorkflowRunEnqueue, run *sdk.V2WorkflowRun, runJobs []sdk.V2WorkflowRunJob, allrunJobsMap map[string]sdk.V2WorkflowRunJob, runJobsContexts sdk.JobsResultContext, u *sdk.AuthentifiedUser, defaultRegion string) (map[string]JobToTrigger, []sdk.V2WorkflowRunInfo, error) {
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
				if nbPermutations != runPermutations {
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

		canBeQueued, infos, err := checkJob(ctx, db, wrEnqueue, *u, *run, jobID, &jobDef, jobContext, defaultRegion)
		runInfos = append(runInfos, infos...)
		if err != nil {
			jobToQueue[jobID] = JobToTrigger{
				Status: sdk.V2WorkflowRunJobStatusSkipped,
				Job:    jobDef,
			}
			return nil, runInfos, err
		}

		if canBeQueued {
			jobToQueue[jobID] = JobToTrigger{
				Status: sdk.V2WorkflowRunJobStatusWaiting,
				Job:    jobDef,
			}
		} else {
			jobToQueue[jobID] = JobToTrigger{
				Status: sdk.V2WorkflowRunJobStatusSkipped,
				Job:    jobDef,
			}

		}
	}

	return jobToQueue, runInfos, nil
}

func checkJob(ctx context.Context, db gorp.SqlExecutor, wrEnqueue sdk.V2WorkflowRunEnqueue, u sdk.AuthentifiedUser, run sdk.V2WorkflowRun, jobID string, jobDef *sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext, defaultRegion string) (bool, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "checkJob", trace.StringAttribute(telemetry.TagJob, jobID))
	defer next()

	runInfos := make([]sdk.V2WorkflowRunInfo, 0)

	// check varset right
	if !wrEnqueue.IsAdminWithMFA {
		varsets := append(run.WorkflowData.Workflow.VariableSets, jobDef.VariableSets...)
		has, vInError, err := rbac.HasRoleOnVariableSetsAndUserID(ctx, db, sdk.VariableSetRoleUse, u.ID, run.ProjectKey, varsets)
		if err != nil {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: unable to check right for user %s on varset %v: %v", jobID, u.Username, varsets, err),
			})
			return false, runInfos, nil
		}
		if !has {
			runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelWarning,
				Message:       fmt.Sprintf("job %s: user %s does not have enough right on varset %s", jobID, u.Username, vInError),
			})
			return false, runInfos, nil
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
	canRun, err := checkJobCondition(ctx, db, run, inputs, *jobDef, currentJobContext, u, wrEnqueue.IsAdminWithMFA)
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

func computeRunStatusFromJobsStatus(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempt int64) (sdk.V2WorkflowRunStatus, error) {
	runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, db, runID, runAttempt)
	if err != nil {
		return "", err
	}

	finalStatus := sdk.V2WorkflowRunStatusSuccess
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
	}
	return finalStatus, nil
}

// Check and set default region on job
func checkUserRegionRight(ctx context.Context, db gorp.SqlExecutor, rj *sdk.V2WorkflowRunJob, wrEnqueue sdk.V2WorkflowRunEnqueue, regionName string, u sdk.AuthentifiedUser) (*sdk.V2WorkflowRunJobInfo, error) {
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

	if !wrEnqueue.IsAdminWithMFA {
		allowedRegions, err := rbac.LoadRegionIDsByRoleAndUserID(ctx, db, sdk.RegionRoleExecute, u.ID)
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
		return nil, nil
	}

	return &sdk.V2WorkflowRunJobInfo{
		WorkflowRunID:    rj.WorkflowRunID,
		Level:            sdk.WorkflowRunInfoLevelError,
		WorkflowRunJobID: rj.ID,
		IssuedAt:         time.Now(),
		Message:          fmt.Sprintf("job %s: user %s does not have enough right on region %q", rj.JobID, u.Username, rj.Region),
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

func checkJobCondition(ctx context.Context, db gorp.SqlExecutor, run sdk.V2WorkflowRun, jobInputs map[string]interface{}, jobDef sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext, u sdk.AuthentifiedUser, isAdminWithMFA bool) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkJobCondition")
	defer next()

	// On keep ancestor of the current job
	var jobCondition string

	if jobDef.Gate != "" {
		gate := run.WorkflowData.Workflow.Gates[jobDef.Gate]

		// Check reviewers
		reviewersChecked := len(gate.Reviewers.Users) == 0 && len(gate.Reviewers.Groups) == 0
		if len(gate.Reviewers.Users) > 0 {
			reviewersChecked = sdk.IsInArray(u.GetUsername(), gate.Reviewers.Users)
		}
		if !reviewersChecked {
			for _, g := range gate.Reviewers.Groups {
				grp, err := group.LoadByName(ctx, db, g, group.LoadOptions.WithMembers)
				if err != nil {
					return false, err
				}
				reviewersChecked = sdk.IsInArray(u.ID, grp.Members.UserIDs())
				if reviewersChecked {
					break
				}
			}
		}
		if !reviewersChecked && !isAdminWithMFA {
			return false, nil
		}

		// Create empty input context to be able to interpolate gate condition.
		currentJobContext.Gate = make(map[string]interface{})
		for k, v := range gate.Inputs {
			if v.Default != nil {
				currentJobContext.Gate[k] = v.Default
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

		jobCondition = gate.If

		// Override with value sent by user
		for k, v := range jobInputs {
			if _, has := currentJobContext.Gate[k]; has {
				currentJobContext.Gate[k] = v
			}
		}
	} else {
		jobCondition = jobDef.If
	}
	if jobCondition == "" {
		jobCondition = "${{success()}}"
	}
	if !strings.HasPrefix(jobCondition, "${{") {
		jobCondition = fmt.Sprintf("${{ %s }}", jobCondition)
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
	booleanResult, err := ap.InterpolateToBool(ctx, jobCondition)
	if err != nil {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to parse if statement %s into a boolean: %v", jobCondition, err)
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
	var lastJobs sdk.V2WorkflowRunJob
	for _, rj := range runJobs {
		if !rj.Status.IsTerminated() {
			return nil
		}
		if sdk.TimeSafe(rj.Started).After(sdk.TimeSafe(lastJobs.Started)) {
			lastJobs = rj
		}
	}

	userID := lastJobs.UserID
	adminMFA := lastJobs.AdminMFA
	// No job have been triggered
	if userID == "" {
		userID = wr.UserID
		adminMFA = wr.AdminMFA
	}

	api.EnqueueWorkflowRun(ctx, wr.ID, userID, wr.WorkflowName, wr.RunNumber, adminMFA)
	return nil
}

func (api *API) EnqueueWorkflowRun(ctx context.Context, runID string, userID string, workflowName string, runNumber int64, adminMFA bool) {
	// Continue workflow
	enqueueRequest := sdk.V2WorkflowRunEnqueue{
		RunID:          runID,
		UserID:         userID,
		IsAdminWithMFA: adminMFA,
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
