package api

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/repository"
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
	run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), wrEnqueue.RunID, workflow_v2.WithRunResults)
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
		}
		run.Status = sdk.StatusFail
		if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
			return err
		}
		if errTx := tx.Commit(); errTx != nil {
			return sdk.WithStack(errTx)
		}
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, *u)
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
		return sdk.WithStack(errTx)
	}
	defer tx.Rollback() // nolint

	for i := range runJobs {
		rj := &runJobs[i]
		// Check gate inputs
		for _, jobEvent := range run.RunJobEvent {
			if jobEvent.RunAttempt != run.RunAttempt {
				continue
			}
			if jobEvent.JobID != rj.JobID {
				continue
			}
			rj.GateInputs = jobEvent.Inputs
		}

		if err := workflow_v2.InsertRunJob(ctx, tx, rj); err != nil {
			return err
		}
		if info, has := runJobsInfos[rj.JobID]; has {
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

	// Synchronize run result in a separate transaction
	api.GoRoutines.Run(ctx, "api.synchronizeRunResults", func(ctx context.Context) {
		if err := api.synchronizeRunResults(ctx, api.mustDBWithCtx(ctx), run.ID); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	})

	if sdk.StatusIsTerminated(run.Status) {
		event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *run, *u)
	}

	if len(skippedJobs) > 0 {
		// Re enqueue workflow to trigger job after
		api.EnqueueWorkflowRun(ctx, run.ID, run.UserID, run.WorkflowName, run.RunNumber)
	}

	// Send to websocket
	for _, rj := range runJobs {
		switch rj.Status {
		case sdk.StatusFail:
			event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, run.Contexts.Git.Server, run.Contexts.Git.Repository, rj)
		default:
			event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnqueued, run.Contexts.Git.Server, run.Contexts.Git.Repository, rj)
		}
	}
	return nil
}

type ArtifactSignature map[string]string

func (api *API) synchronizeRunResults(ctx context.Context, db gorp.SqlExecutor, runID string) error {
	run, err := workflow_v2.LoadRunByID(ctx, db, runID)
	if err != nil {
		return err
	}

	// Synchronize workflow runs
	runResults, err := workflow_v2.LoadRunResults(ctx, db, runID, &integration.LoadProjectIntegrationByIDsWithClearPassword)
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

	for i := range run.WorkflowData.Integrations {
		integs, err := integration.LoadProjectIntegrationByIDsWithClearPassword(ctx, db, run.WorkflowData.Integrations[i].ID)
		if err != nil {
			return err
		}

		integ := integs[run.WorkflowData.Integrations[i].ID]

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

	for i := range runResults {
		result := &runResults[i]
		if result.ArtifactManagerIntegrationID == nil {
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
		} else {
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

			log.Info(ctx, "artifact %s%s signature: %s", localRepository, fi.Path, signature)

			props.AddProperty("cds.signature", signature)
			if err := artifactClient.SetProperties(localRepository, fi.Path, props); err != nil {
				ctx := log.ContextWithStackTrace(ctx, err)
				log.Error(ctx, "unable to set artifact properties from result %s: %v", result.ID, err)
				continue
			}
		}
	}

	if artifactClient != nil && artifactoryIntegration != nil {
		// Set the Buildinfo
		buildInfoRequest, err := art.PrepareBuildInfo(ctx, artifactClient, art.BuildInfoRequest{
			BuildInfoPrefix: artifactoryIntegration.Config[sdk.ArtifactoryConfigBuildInfoPrefix].Value,
			ProjectKey:      run.ProjectKey,
			WorkflowName:    run.WorkflowName,
			Version:         run.Contexts.Git.SemverCurrent,
			AgentName:       "cds-api",
			TokenName:       rtTokenName,
			//RunURL:                   "TODO", // TODO Run UI URL
			GitBranch:                run.Contexts.Git.Ref,
			GitURL:                   run.Contexts.Git.RepositoryURL,
			GitHash:                  run.Contexts.Git.Sha,
			RunResultsV2:             runResults,
			DefaultLowMaturitySuffix: artifactoryIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value,
		})
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}

		log.Info(ctx, "Creating Artifactory Build %s %s on project %s...\n", buildInfoRequest.Name, buildInfoRequest.Number, artifactoryProjectKey)

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
				log.Error(ctx, "error while pushing buildinfo %s %s. Retrying...\n", buildInfoRequest.Name, buildInfoRequest.Number)
			}
		}

	}

	return nil
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

func prepareRunJobs(_ context.Context, proj sdk.Project, run sdk.V2WorkflowRun, wrEnqueue sdk.V2WorkflowRunEnqueue, jobsToQueue map[string]sdk.V2Job, jobStatus string, u sdk.AuthentifiedUser) []sdk.V2WorkflowRunJob {
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
	runJobsContexts := computeExistingRunJobContexts(*run, runJobs)

	// temp map of run jobs
	allrunJobsMap := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runJobs {
		// addition check for matrix job, only keep not terminated one if present
		runjob, has := allrunJobsMap[rj.JobID]
		if !has || sdk.StatusIsTerminated(runjob.Status) {
			allrunJobsMap[rj.JobID] = rj
		}
	}

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

		// Skip the job in stage that cannot be run
		if len(stages) > 0 && !stages[jobDef.Stage].CanBeRun {
			continue
		}

		// Check jobs needs
		if ok := checkJobNeeds(runJobsContexts, jobDef); !ok {
			continue
		}

		// Build job context
		jobContext := buildContextForJob(ctx, run.WorkflowData.Workflow.Jobs, runJobsContexts, run.Contexts, jobID)

		canBeQueued, infos, err := checkJob(ctx, db, *u, *run, jobID, &jobDef, jobContext, defaultRegion)
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

func checkJob(ctx context.Context, db gorp.SqlExecutor, u sdk.AuthentifiedUser, run sdk.V2WorkflowRun, jobID string, jobDef *sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext, defaultRegion string) (bool, []sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "checkJob", trace.StringAttribute(telemetry.TagJob, jobID))
	defer next()

	runInfos := make([]sdk.V2WorkflowRunInfo, 0)

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
	canRun, err := checkJobCondition(ctx, run, jobID, jobDef, currentJobContext)
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

func checkJobCondition(ctx context.Context, run sdk.V2WorkflowRun, jobID string, jobDef *sdk.V2Job, currentJobContext sdk.WorkflowRunJobsContext) (bool, error) {
	ctx, next := telemetry.Span(ctx, "checkJobCondition")
	defer next()

	// On keep ancestor of the current job
	var jobCondition string

	if jobDef.Gate != "" {
		jobCondition = run.WorkflowData.Workflow.Gates[jobDef.Gate].If
		// Create empty input context to be able to interpolate gate condition.
		currentJobContext.Gate = make(map[string]interface{})
		for k, v := range run.WorkflowData.Workflow.Gates[jobDef.Gate].Inputs {
			switch v.Type {
			case "boolean":
				currentJobContext.Gate[k] = false
			case "number":
				currentJobContext.Gate[k] = 0
			default:
				currentJobContext.Gate[k] = ""
			}
		}

		// Check if there is an event
		for _, je := range run.RunJobEvent {
			if je.RunAttempt != run.RunAttempt {
				continue
			}
			if je.JobID != jobID {
				continue
			}

			// Ovveride with value sent by user
			for k, v := range je.Inputs {
				if _, has := currentJobContext.Gate[k]; has {
					currentJobContext.Gate[k] = v
				}
			}
		}
	} else {
		if jobDef.If == "" {
			jobDef.If = "${{success()}}"
		}
		if !strings.HasPrefix(jobDef.If, "${{") {
			jobDef.If = fmt.Sprintf("${{ %s }}", jobDef.If)
		}
		jobCondition = jobDef.If
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
	interpolatedInput, err := ap.Interpolate(ctx, jobCondition)
	if err != nil {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "job %s: unable to parse if statement %s: %v", jobID, jobCondition, err)
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
