package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func syncJobInNodeRun(n *sdk.WorkflowNodeRun, j *sdk.WorkflowNodeJobRun, stageIndex int) {
	stage := &n.Stages[stageIndex]
	for i := range stage.RunJobs {
		rj := &stage.RunJobs[i]
		if rj.ID == j.ID {
			rj.Status = j.Status
			rj.Start = j.Start
			rj.Done = j.Done
			rj.Model = j.Model
			rj.ModelType = j.ModelType
			rj.Region = j.Region
			rj.ContainsService = j.ContainsService
			rj.Job = j.Job
			rj.Header = j.Header
			rj.Parameters = j.Parameters
			rj.SpawnInfos = j.SpawnInfos
		}
	}
}

func syncTakeJobInNodeRun(ctx context.Context, db gorp.SqlExecutor, n *sdk.WorkflowNodeRun, j *sdk.WorkflowNodeJobRun, stageIndex int) (*ProcessorReport, error) {
	_, end := telemetry.Span(ctx, "workflow.syncTakeJobInNodeRun")
	defer end()

	report := new(ProcessorReport)

	//If status is not waiting neither build: nothing to do
	if sdk.StatusIsTerminated(n.Status) {
		return nil, nil
	}

	nodeUpdated := false
	//Browse stages
	stage := &n.Stages[stageIndex]
	if stage.Status == sdk.StatusWaiting {
		stage.Status = sdk.StatusBuilding
		nodeUpdated = true
	}
	isStopped := true
	for i := range stage.RunJobs {
		rj := &stage.RunJobs[i]
		if rj.ID == j.ID {
			rj.Status = j.Status
			rj.Start = j.Start
			rj.Done = j.Done
			rj.Model = j.Model
			rj.ModelType = j.ModelType
			rj.Region = j.Region
			rj.ContainsService = j.ContainsService
			rj.WorkerName = j.WorkerName
			rj.HatcheryName = j.HatcheryName
			rj.Job = j.Job
			rj.Header = j.Header
			rj.Parameters = j.Parameters
		}
		if rj.Status != sdk.StatusStopped {
			isStopped = false
		}
	}
	if isStopped {
		nodeUpdated = true
		stage.Status = sdk.StatusStopped
	}

	if n.Status == sdk.StatusWaiting {
		nodeUpdated = true
		n.Status = sdk.StatusBuilding
	}

	if nodeUpdated {
		report.Add(ctx, *n)
	}

	// Save the node run in database
	if err := UpdateNodeRunStatusAndStage(db, n); err != nil {
		return nil, sdk.WrapError(err, "unable to update node id=%d at status %s", n.ID, n.Status)
	}
	return report, nil
}

func executeNodeRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, workflowNodeRun *sdk.WorkflowNodeRun) (*ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.executeNodeRun",
		telemetry.Tag(telemetry.TagProjectKey, proj.Key),
		telemetry.Tag(telemetry.TagWorkflowRun, workflowNodeRun.Number),
		telemetry.Tag(telemetry.TagWorkflowNodeRun, workflowNodeRun.ID),
		telemetry.Tag("workflow_node_run_status", workflowNodeRun.Status),
	)
	defer end()

	wr, err := LoadRunByID(ctx, db, workflowNodeRun.WorkflowRunID, LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load workflow run with id %d", workflowNodeRun.WorkflowRunID)
	}

	n := wr.Workflow.WorkflowData.NodeByID(workflowNodeRun.WorkflowNodeID)

	report := new(ProcessorReport)
	defer func(wnr *sdk.WorkflowNodeRun) {
		report.Add(ctx, *wnr)
	}(workflowNodeRun)

	// If status is not waiting neither build: nothing to do
	if sdk.StatusIsTerminated(workflowNodeRun.Status) {
		return nil, nil
	}

	var newStatus = workflowNodeRun.Status

	// For join and fork, reuse the status from parent
	if n.Type == sdk.NodeTypeJoin || n.Type == sdk.NodeTypeFork {
		parentStatus := sdk.ParameterFind(workflowNodeRun.BuildParameters, "cds.status")
		newStatus = parentStatus.Value
	} else if len(workflowNodeRun.Stages) == 0 { // If no stages ==> success
		newStatus = sdk.StatusSuccess
	}

	stagesTerminated := 0
	var previousNodeRun *sdk.WorkflowNodeRun
	if workflowNodeRun.Manual != nil && workflowNodeRun.Manual.OnlyFailedJobs {
		previousNodeRun, err = checkRunOnlyFailedJobs(wr, workflowNodeRun)
		if err != nil {
			return report, err
		}
	}

	// Browse stages
	for stageIndex := range workflowNodeRun.Stages {
		stage := &workflowNodeRun.Stages[stageIndex]
		// Initialize stage status at waiting
		if stage.Status == "" {
			var previousStage sdk.Stage
			// Find previous stage
			if previousNodeRun != nil {
				for i := range previousNodeRun.Stages {
					if previousNodeRun.Stages[i].ID == stage.ID {
						previousStage = previousNodeRun.Stages[i]
						break
					}
				}
			}

			if previousNodeRun == nil || previousStage.Status == sdk.StatusFail || !sdk.StatusIsTerminated(previousStage.Status) {
				stage.Status = sdk.StatusWaiting
				if stageIndex == 0 {
					newStatus = sdk.StatusWaiting
				}
			} else if sdk.StatusIsTerminated(previousStage.Status) {
				// If stage terminated, recopy it
				workflowNodeRun.Stages[stageIndex] = previousStage
				stagesTerminated++
				continue
			}

			if len(stage.Jobs) == 0 {
				stage.Status = sdk.StatusSuccess
			} else {
				// Add job to Queue
				// Insert data in workflow_node_run_job
				log.Debug(ctx, "workflow.executeNodeRun> stage %s call addJobsToQueue", stage.Name)
				r, err := addJobsToQueue(ctx, store, db, proj, stage, wr, workflowNodeRun, &previousStage)
				report.Merge(ctx, r)
				if err != nil {
					return report, err
				}
				log.Debug(ctx, "workflow.executeNodeRun> stage %s status after call to addJobsToQueue %s", stage.Name, stage.Status)
			}

			// Check for failure caused by action not usable or requirements problem
			if sdk.StatusFail == stage.Status {
				newStatus = sdk.StatusFail
				break
			}

			if sdk.StatusIsTerminated(stage.Status) {
				stagesTerminated++
				continue
			}
			break
		}

		// check for failure caused by action not usable or requirements problem
		if sdk.StatusFail == stage.Status {
			newStatus = sdk.StatusFail
			break
		}

		if sdk.StatusIsTerminated(stage.Status) {
			stagesTerminated++
		}

		//If stage is waiting, nothing to do
		if stage.Status == sdk.StatusWaiting {
			log.Debug(ctx, "workflow.executeNodeRun> stage %s status:%s - nothing to do", stage.Name, stage.Status)
			break
		}

		if stage.Status == sdk.StatusBuilding {
			newStatus = sdk.StatusBuilding
			var end bool

			_, next := telemetry.Span(ctx, "workflow.syncStage")
			end, errSync := syncStage(ctx, db, store, stage)
			next()
			if errSync != nil {
				return report, errSync
			}
			if !end {
				break
			} else {
				//The stage is over
				if stage.Status == sdk.StatusFail {
					workflowNodeRun.Done = time.Now()
					newStatus = sdk.StatusFail
					stagesTerminated++
					break
				}
				if stage.Status == sdk.StatusStopped {
					workflowNodeRun.Done = time.Now()
					newStatus = sdk.StatusStopped
					stagesTerminated++
					break
				}

				if sdk.StatusIsTerminated(stage.Status) {
					stagesTerminated++
					workflowNodeRun.Done = time.Now()
				}

				if stageIndex == len(workflowNodeRun.Stages)-1 {
					workflowNodeRun.Done = time.Now()
					newStatus = sdk.StatusSuccess
					stagesTerminated++
					break
				}
				if stageIndex != len(workflowNodeRun.Stages)-1 {
					continue
				}
			}
		}
	}

	if stagesTerminated >= len(workflowNodeRun.Stages) || (stagesTerminated >= len(workflowNodeRun.Stages)-1 &&
		(workflowNodeRun.Stages[len(workflowNodeRun.Stages)-1].Status == sdk.StatusDisabled || workflowNodeRun.Stages[len(workflowNodeRun.Stages)-1].Status == sdk.StatusSkipped)) {
		var counterStatus statusCounter
		if len(workflowNodeRun.Stages) > 0 {
			for _, stage := range workflowNodeRun.Stages {
				computeRunStatus(stage.Status, &counterStatus)
			}
			newStatus = getRunStatus(counterStatus)
		}
	}

	workflowNodeRun.Status = newStatus

	if sdk.StatusIsTerminated(workflowNodeRun.Status) && workflowNodeRun.Status != sdk.StatusNeverBuilt {
		workflowNodeRun.Done = time.Now()
	}

	// Save the node run in database
	if err := UpdateNodeRunStatusAndStage(db, workflowNodeRun); err != nil {
		return nil, sdk.WrapError(err, "unable to update node id=%d at status %s", workflowNodeRun.ID, workflowNodeRun.Status)
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(ctx, db, workflowNodeRun.WorkflowRunID, LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to reload workflow run id=%d", workflowNodeRun.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	// Delete jobs only when node is over
	if sdk.StatusIsTerminated(workflowNodeRun.Status) {
		if workflowNodeRun.Status != sdk.StatusStopped {
			r1, _, err := processWorkflowDataRun(ctx, db, store, proj, updatedWorkflowRun, nil, nil, nil)
			if err != nil {
				return nil, sdk.WrapError(err, "unable to reprocess workflow")
			}
			report.Merge(ctx, r1)
		}

		// Delete the line in workflow_node_run_job
		if err := DeleteNodeJobRuns(db, workflowNodeRun.ID); err != nil {
			// Checking the error:
			// pq: update or delete on table "workflow_node_run_job" violates foreign key constraint "fk_worker_workflow_node_run_job" on table "worker")
			type WorkerInfo struct {
				WorkerID             string `db:"worker_id"`
				WorkerName           string `db:"worker_name"`
				WorkflowNodeRunJobID string `db:"workflow_node_run_job_id"`
			}

			var workers []WorkerInfo
			if _, errSelect := db.Select(&workers, `
        SELECT worker.id as worker_id, worker.name as worker_name, workflow_node_run_job.id as workflow_node_run_job_id FROM worker
        JOIN workflow_node_run_job ON workflow_node_run_job.worker_id = worker.id
        WHERE workflow_node_run_job.workflow_node_run_id = $1
      `, workflowNodeRun.ID); errSelect != nil {
				log.ErrorWithStackTrace(ctx, sdk.WrapError(errSelect, "unable to get worker list for node run with id %d", workflowNodeRun.ID))
			} else {
				buf, _ := json.Marshal(workers)
				log.Error(ctx, "list of workers for node run %d that block jobs deletion (len:%d): %q", workflowNodeRun.ID, len(workers), string(buf))
			}

			return nil, sdk.WrapError(err, "unable to delete node %d job runs", workflowNodeRun.ID)
		}

		// If current node has a mutex, we want to trigger another node run that can be waiting for the mutex
		node := updatedWorkflowRun.Workflow.WorkflowData.NodeByID(workflowNodeRun.WorkflowNodeID)
		hasMutex := node != nil && node.Context != nil && node.Context.Mutex
		if hasMutex {
			r, err := releaseMutex(ctx, db, store, proj, updatedWorkflowRun.WorkflowID, workflowNodeRun.WorkflowNodeName)
			report.Merge(ctx, r)
			if err != nil {
				return report, err
			}
		}
	}
	return report, nil
}

func releaseMutex(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, workflowID int64, nodeName string) (*ProcessorReport, error) {
	ctx, end := telemetry.Span(ctx, "workflow.releaseMutex")
	defer end()

	mutexQuery := `
    SELECT workflow_node_run.id
    FROM workflow_node_run
    JOIN workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
    JOIN workflow on workflow.id = workflow_run.workflow_id
    WHERE workflow.id = $1
      AND workflow_node_run.workflow_node_name = $2
      AND workflow_node_run.status = $3
    ORDER BY workflow_run.num ASC
    LIMIT 1
  `
	waitingRunID, err := db.SelectInt(mutexQuery, workflowID, nodeName, sdk.StatusWaiting)
	if err != nil && err != sql.ErrNoRows {
		err = sdk.WrapError(err, "unable to load mutex-locked workflow node run id")
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "%v", err)
		return nil, nil
	}
	if waitingRunID == 0 {
		return nil, nil
	}

	// Load the workflow node run that is waiting for the mutex
	waitingRun, errRun := LoadNodeRunByID(ctx, db, waitingRunID, LoadRunOptions{})
	if errRun != nil && sdk.Cause(errRun) != sql.ErrNoRows {
		err = sdk.WrapError(err, "unable to load mutex-locked workflow node run")
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, err.Error())
		return nil, nil
	}
	if waitingRun == nil {
		return nil, nil
	}

	// Load the workflow run that is waiting for the mutex
	workflowRun, err := LoadRunByID(ctx, db, waitingRun.WorkflowRunID, LoadRunOptions{})
	if err != nil {
		err = sdk.WrapError(err, "unable to load mutex-locked workflow run")
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, err.Error())
		return nil, nil
	}

	// Add a spawn info on the workflow run
	AddWorkflowRunInfo(workflowRun, sdk.SpawnMsgNew(*sdk.MsgWorkflowNodeMutexRelease, waitingRun.WorkflowNodeName))
	if err := UpdateWorkflowRun(ctx, db, workflowRun); err != nil {
		return nil, sdk.WrapError(err, "unable to update workflow run %d after mutex release", workflowRun.ID)
	}

	log.Debug(ctx, "workflow.execute> process the node run %d because mutex has been released", waitingRun.ID)
	r, err := executeNodeRun(ctx, db, store, proj, waitingRun)
	if err != nil {
		return r, sdk.WrapError(err, "unable to reprocess workflow")
	}

	return r, nil
}

func checkRunOnlyFailedJobs(wr *sdk.WorkflowRun, nr *sdk.WorkflowNodeRun) (*sdk.WorkflowNodeRun, error) {
	var previousNR *sdk.WorkflowNodeRun
	nrs, ok := wr.WorkflowNodeRuns[nr.WorkflowNodeID]
	if !ok {
		return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "node %d not found in workflow run %d", nr.WorkflowNodeID, wr.ID)
	}
	for i := range nrs {
		if nrs[i].SubNumber < nr.SubNumber {
			previousNR = &nrs[i]
			break
		}
	}

	if previousNR == nil {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find a previous execution of this pipeline")
	}

	if len(previousNR.Stages) != len(nr.Stages) {
		return nil, sdk.NewErrorFrom(sdk.ErrForbidden, "you cannot rerun a pipeline that have a different number of stages")
	}

	for i, s := range nr.Stages {
		if len(s.Jobs) != len(previousNR.Stages[i].Jobs) {
			return nil, sdk.NewErrorFrom(sdk.ErrForbidden, "you cannot rerun a pipeline that have a different number of jobs")
		}
	}
	return previousNR, nil
}

func addJobsToQueue(ctx context.Context, store cache.Store, db gorpmapper.SqlExecutorWithTx, proj sdk.Project, stage *sdk.Stage, wr *sdk.WorkflowRun, nr *sdk.WorkflowNodeRun, previousStage *sdk.Stage) (*ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.addJobsToQueue")
	defer end()

	report := new(ProcessorReport)

	_, next := telemetry.Span(ctx, "checkCondition")
	conditionsOK := checkCondition(ctx, wr, stage.Conditions, nr.BuildParameters)
	next()
	if !conditionsOK {
		stage.Status = sdk.StatusSkipped
	}
	if !stage.Enabled {
		stage.Status = sdk.StatusDisabled
	}

	_, next = telemetry.Span(ctx, "workflow.getIntegrationPlugins")
	integrationConfigs, integrationPlugins, err := getIntegrationPlugins(ctx, db, wr, nr)
	if err != nil {
		return report, sdk.WrapError(err, "unable to get integration plugins requirement")
	}
	next()

	_, next = telemetry.Span(ctx, "workflow.getJobExecutablesGroups")
	groups, err := getExecutablesGroups(wr, nr)
	if err != nil {
		return report, sdk.WrapError(err, "error getting job executables groups")
	}
	next()

	skippedOrDisabledJobs := 0
	failedJobs := 0
	//Browse the jobs
jobLoop:
	for j := range stage.Jobs {
		job := stage.Jobs[j]

		if previousStage != nil {
			for _, rj := range previousStage.RunJobs {
				if rj.Job.PipelineActionID == job.PipelineActionID && rj.Status != sdk.StatusFail && sdk.StatusIsTerminated(rj.Status) {
					stage.RunJobs = append(stage.RunJobs, rj)
					continue jobLoop
				}
			}
		}

		// errors generated in the loop will be added to job run spawn info
		spawnErrs := sdk.MultiError{}

		// Copy context from noderun
		jobFullContext := sdk.JobRunContext{}
		jobFullContext.Vars = nr.Contexts.Vars
		jobFullContext.Git = nr.Contexts.Git

		//Process variables for the jobs
		_, next = telemetry.Span(ctx, "workflow..getNodeJobRunParameters")
		jobParams, err := getNodeJobRunParameters(job, nr, stage)
		next()
		if err != nil {
			spawnErrs.Join(*err)
		}

		_, next = telemetry.Span(ctx, "workflow.processNodeJobRunRequirements")
		jobRequirements, containsService, modelType, err := processNodeJobRunRequirements(ctx, store, db, proj.Key, *wr, job, nr, sdk.Groups(groups).ToIDs(), integrationPlugins, integrationConfigs, jobParams)
		next()
		if err != nil {
			spawnErrs.Join(*err)
		}

		// Retrieve service requirement
		jobContext := sdk.JobContext{Services: make(map[string]sdk.JobContextService)}
		jobContext.Status = strings.ToLower(sdk.StatusSuccess)
		jobFullContext.Job = jobContext

		if exist := featureflipping.Exists(ctx, gorpmapping.Mapper, db, sdk.FeatureRegion); exist {
			if err := checkJobRegion(ctx, db, proj.Key, proj.Organization, wr.Workflow.Name, jobRequirements); err != nil {
				spawnErrs.Append(err)
			}
		}

		// check that children actions used by job can be used by the project
		if err := action.CheckChildrenForGroupIDsWithLoop(ctx, db, &job.Action, sdk.Groups(groups).ToIDs()); err != nil {
			spawnErrs.Append(err)
		}

		// add requirements in job parameters, to use them as {{.job.requirement...}} in job
		_, next = telemetry.Span(ctx, "workflow.prepareRequirementsToNodeJobRunParameters")
		jobParams = append(jobParams, prepareRequirementsToNodeJobRunParameters(jobRequirements)...)
		next()

		//Create the job run
		wjob := sdk.WorkflowNodeJobRun{
			ProjectID:          wr.ProjectID,
			WorkflowNodeRunID:  nr.ID,
			Start:              time.Time{},
			Queued:             time.Now(),
			Status:             sdk.StatusWaiting,
			Parameters:         jobParams,
			ExecGroups:         groups,
			IntegrationPlugins: integrationPlugins,
			Job: sdk.ExecutedJob{
				Job: job,
			},
			Header:          nr.Header,
			ContainsService: containsService,
			Contexts:        jobFullContext,
		}
		wjob.ModelType = modelType
		wjob.Job.Job.Action.Requirements = jobRequirements // Set the interpolated requirements on the job run only

		// Set region from requirement on job run if exists
		for i := range jobRequirements {
			if jobRequirements[i].Type == sdk.RegionRequirement {
				wjob.Region = &jobRequirements[i].Value
				break
			}
		}

		if !stage.Enabled || !wjob.Job.Enabled {
			wjob.Status = sdk.StatusDisabled
			skippedOrDisabledJobs++
		} else if !conditionsOK {
			wjob.Status = sdk.StatusSkipped
			skippedOrDisabledJobs++
		}

		// If there is any error in the previous operation, mark the job as failed
		if !spawnErrs.IsEmpty() {
			failedJobs++
			wjob.Status = sdk.StatusFail
			for _, e := range spawnErrs {
				log.ErrorWithStackTrace(ctx, e)
				wjob.SpawnInfos = append(wjob.SpawnInfos, sdk.SpawnInfo{
					Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobError.ID, Args: []interface{}{sdk.ExtractHTTPError(e).Error()}},
				})
			}
		} else {
			if wjob.Status == sdk.StatusDisabled {
				wjob.SpawnInfos = []sdk.SpawnInfo{{
					Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobDisabled.ID},
				}}
			} else {
				wjob.SpawnInfos = []sdk.SpawnInfo{{
					Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobInQueue.ID},
				}}
			}

		}

		// insert in database
		_, next = telemetry.Span(ctx, "workflow.insertWorkflowNodeJobRun")
		if err := insertWorkflowNodeJobRun(db, &wjob); err != nil {
			next()
			return report, sdk.WrapError(err, "unable to insert in table workflow_node_run_job")
		}
		next()

		if err := AddSpawnInfosNodeJobRun(db, wjob.WorkflowNodeRunID, wjob.ID, wjob.SpawnInfos); err != nil {
			return nil, sdk.WrapError(err, "cannot save spawn info job %d", wjob.ID)
		}

		//Put the job run in database
		stage.RunJobs = append(stage.RunJobs, wjob)

		report.Add(ctx, wjob)
	}

	if skippedOrDisabledJobs == len(stage.Jobs) {
		stage.Status = sdk.StatusSkipped
	}

	if failedJobs > 0 {
		stage.Status = sdk.StatusFail
	}

	return report, nil
}

func getIntegrationPlugins(ctx context.Context, db gorp.SqlExecutor, wr *sdk.WorkflowRun, nr *sdk.WorkflowNodeRun) ([]sdk.IntegrationConfig, []sdk.GRPCPlugin, error) {
	plugins := make([]sdk.GRPCPlugin, 0)
	mapConfig := make([]sdk.IntegrationConfig, 0)

	var projectIntegration *sdk.ProjectIntegration
	node := wr.Workflow.WorkflowData.NodeByID(nr.WorkflowNodeID)
	if node != nil && node.Context != nil {
		if node.Context.ProjectIntegrationID != 0 {
			pp, has := wr.Workflow.ProjectIntegrations[node.Context.ProjectIntegrationID]
			if has {
				projectIntegration = &pp
			}
		}
	}

	if projectIntegration != nil && projectIntegration.Model.ID > 0 {
		mapConfig = append(mapConfig, projectIntegration.Config)
		plg, err := plugin.LoadByIntegrationModelIDAndType(ctx, db, projectIntegration.Model.ID, sdk.GRPCPluginDeploymentIntegration)
		if err != nil {
			return nil, nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNotFound, "cannot find plugin for integration model %q", projectIntegration.Model.Name))
		}
		plugins = append(plugins, *plg)
	}

	var artifactManagerInteg *sdk.WorkflowProjectIntegration
	for i := range wr.Workflow.Integrations {
		if wr.Workflow.Integrations[i].ProjectIntegration.Model.ArtifactManager {
			artifactManagerInteg = &wr.Workflow.Integrations[i]
		}
	}
	if artifactManagerInteg != nil {
		mapConfig = append(mapConfig, artifactManagerInteg.Config)
		plgs, err := plugin.LoadAllByIntegrationModelID(ctx, db, artifactManagerInteg.ProjectIntegration.Model.ID)
		if err != nil {
			return nil, nil, sdk.NewErrorFrom(sdk.ErrNotFound, "Cannot find plugin for integration model id %d, %v", artifactManagerInteg.ProjectIntegration.Model.ID, err)
		}
		platform := artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPlatform]
		for _, plg := range plgs {
			if strings.HasPrefix(plg.Name, fmt.Sprintf("%s-", platform.Value)) {
				plugins = append(plugins, plg)
			}
		}
	}

	return mapConfig, plugins, nil
}

func getExecutablesGroups(wr *sdk.WorkflowRun, nr *sdk.WorkflowNodeRun) ([]sdk.Group, error) {
	var node = wr.Workflow.WorkflowData.NodeByID(nr.WorkflowNodeID)
	var groups []sdk.Group

	if len(node.Groups) > 0 {
		for _, gp := range node.Groups {
			if gp.Permission >= sdk.PermissionReadExecute {
				groups = append(groups, gp.Group)
			}
		}
	} else {
		for _, gp := range wr.Workflow.Groups {
			if gp.Permission >= sdk.PermissionReadExecute {
				groups = append(groups, gp.Group)
			}
		}
	}
	groups = append(groups, *group.SharedInfraGroup)

	return groups, nil
}

func syncStage(ctx context.Context, db gorp.SqlExecutor, store cache.Store, stage *sdk.Stage) (bool, error) {
	stageEnd := true
	finalStatus := sdk.StatusBuilding

	// browse all running jobs
	for indexJob := range stage.RunJobs {
		runJob := &stage.RunJobs[indexJob]

		// If job is runnning, sync it
		if runJob.Status == sdk.StatusBuilding || runJob.Status == sdk.StatusWaiting {
			runJobDB, errJob := LoadNodeJobRun(ctx, db, store, runJob.ID)
			if errJob != nil {
				return stageEnd, errJob
			}

			if runJobDB.Status == sdk.StatusBuilding || runJobDB.Status == sdk.StatusWaiting {
				stageEnd = false
			}
			spawnInfos, err := LoadNodeRunJobInfo(ctx, db, runJob.WorkflowNodeRunID, runJob.ID)
			if err != nil {
				return false, sdk.WrapError(err, "unable to load spawn infos for runJob: %d", runJob.ID)
			}
			runJob.SpawnInfos = spawnInfos

			// If same status, sync step status
			if runJobDB.Status == runJob.Status {
				runJob.Job.StepStatus = runJobDB.Job.StepStatus
			} else {
				runJob.Status = runJobDB.Status
				runJob.Start = runJobDB.Start
				runJob.Done = runJobDB.Done
				runJob.Model = runJobDB.Model
				runJob.ModelType = runJobDB.ModelType
				runJob.Region = runJobDB.Region
				runJob.ContainsService = runJobDB.ContainsService
				runJob.Job = runJobDB.Job
				runJob.WorkerName = runJobDB.WorkerName
				runJob.HatcheryName = runJobDB.HatcheryName
			}
		}
	}
	log.Debug(ctx, "syncStage> stage %s stageEnd:%t len(stage.RunJobs):%d", stage.Name, stageEnd, len(stage.RunJobs))

	if stageEnd || len(stage.RunJobs) == 0 {
		finalStatus = sdk.StatusSuccess
		stageEnd = true
		// Determine final stage status
	finalStageLoop:
		for _, runJob := range stage.RunJobs {
			switch runJob.Status {
			case sdk.StatusDisabled:
				if finalStatus == sdk.StatusBuilding {
					finalStatus = sdk.StatusDisabled
				}
			case sdk.StatusSkipped:
				if finalStatus == sdk.StatusBuilding || finalStatus == sdk.StatusDisabled {
					finalStatus = sdk.StatusSkipped
				}
			case sdk.StatusFail:
				finalStatus = sdk.StatusFail
				break finalStageLoop
			case sdk.StatusSuccess:
				if finalStatus != sdk.StatusFail {
					finalStatus = sdk.StatusSuccess
				}
			case sdk.StatusStopped:
				if finalStatus != sdk.StatusFail {
					finalStatus = sdk.StatusStopped
				}
			}
		}
	}
	log.Debug(ctx, "syncStage> set stage %s from %s to %s", stage.Name, stage.Status, finalStatus)
	stage.Status = finalStatus
	return stageEnd, nil
}

// NodeBuildParametersFromRun return build parameters from previous workflow run
func NodeBuildParametersFromRun(wr sdk.WorkflowRun, id int64) ([]sdk.Parameter, error) {
	params := make([]sdk.Parameter, 0)

	nodesRun, ok := wr.WorkflowNodeRuns[id]
	if !ok || len(nodesRun) == 0 {
		return params, nil
	}

	for _, p := range nodesRun[0].BuildParameters {
		sdk.AddParameter(&params, p.Name, p.Type, p.Value)
	}

	return params, nil
}

// NodeBuildParametersFromWorkflow returns build_parameters for a node given its id
func NodeBuildParametersFromWorkflow(proj sdk.Project, wf *sdk.Workflow, refNode *sdk.Node, ancestorsIds []int64) ([]sdk.Parameter, error) {
	runContext := nodeRunContext{
		WorkflowProjectIntegrations: wf.Integrations,
		ProjectIntegrations:         make([]sdk.ProjectIntegration, 0),
	}
	res := make([]sdk.Parameter, 0)
	if refNode != nil && refNode.Context != nil {
		if refNode.Context.PipelineID != 0 && wf.Pipelines != nil {
			pip, has := wf.Pipelines[refNode.Context.PipelineID]
			if has {
				runContext.Pipeline = pip
			}
		}
		if refNode.Context.ApplicationID != 0 && wf.Applications != nil {
			app, has := wf.Applications[refNode.Context.ApplicationID]
			if has {
				runContext.Application = app
			}
		}
		if refNode.Context.EnvironmentID != 0 && wf.Environments != nil {
			env, has := wf.Environments[refNode.Context.EnvironmentID]
			if has {
				runContext.Environment = env
			}
		}
		if refNode.Context.ProjectIntegrationID != 0 && wf.ProjectIntegrations != nil {
			pp, has := wf.ProjectIntegrations[refNode.Context.ProjectIntegrationID]
			if has {
				runContext.ProjectIntegrations = append(runContext.ProjectIntegrations, pp)
			}
		}
		runContext.NodeGroups = refNode.Groups

		var err error
		res, _, err = getBuildParameterFromNodeContext(proj, *wf, runContext, refNode.Context.DefaultPipelineParameters, refNode.Context.DefaultPayload, nil)
		if err != nil {
			return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to get workflow node parameters: %v", err)
		}
	}

	// Process ancestor
	for _, aID := range ancestorsIds {
		ancestor := wf.WorkflowData.NodeByID(aID)
		if ancestor == nil {
			continue
		}
		sdk.AddParameter(&res, "workflow."+ancestor.Name+".status", "string", "")
	}

	// Add payload from root
	if wf.WorkflowData.Node.Context.DefaultPayload != nil {
		e := dump.NewDefaultEncoder()
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false

		tempParams := sdk.ParametersToMap(res)
		m1, errm1 := e.ToStringMap(wf.WorkflowData.Node.Context.DefaultPayload)
		if errm1 == nil {
			mergedParameters := sdk.ParametersMapMerge(tempParams, m1, sdk.MapMergeOptions.ExcludeGitParams)
			res = sdk.ParametersFromMap(mergedParameters)
		}
	}

	return res, nil
}

type stopNodeJobRunResult struct {
	report *ProcessorReport
	err    error
}

func stopWorkflowNodePipeline(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj sdk.Project, nodeRun *sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo) (*ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.stopWorkflowNodePipeline")
	defer end()

	report := new(ProcessorReport)

	const stopWorkflowNodeRunNBWorker = 5
	var wg sync.WaitGroup

	ids, err := LoadNodeJobRunIDByNodeRunID(dbFunc(), nodeRun.ID)
	if err != nil {
		return report, sdk.WrapError(err, "cannot load node jobs run ids ")
	}

	chanStopID := make(chan int64, stopWorkflowNodeRunNBWorker)
	chanStopResult := make(chan stopNodeJobRunResult, stopWorkflowNodeRunNBWorker)
	for i := 0; i < stopWorkflowNodeRunNBWorker && i < len(ids); i++ {
		go func() {
			stopWorkflowNodeJobRun(ctx, dbFunc, store, proj, stopInfos, chanStopID, chanStopResult)
		}()
	}

	wg.Add(len(ids))
	for _, njrID := range ids {
		chanStopID <- njrID
	}
	close(chanStopID)

	for i := 0; i < len(ids); i++ {
		r := <-chanStopResult
		wg.Done()
		report.Merge(ctx, r.report)
		if r.err != nil {
			return report, err
		}
	}
	wg.Wait()

	tx, err := dbFunc().Begin()
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create transaction")
	}
	defer tx.Rollback() //nolint

	// Update stages from node run
	stopWorkflowNodeRunStages(ctx, tx, nodeRun)

	nodeRun.Status = sdk.StatusStopped
	nodeRun.Done = time.Now()

	if err := UpdateNodeRun(tx, nodeRun); err != nil {
		return report, sdk.WrapError(err, "cannot update node run")
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "cannot commit transaction")
	}

	return report, nil
}

func stopWorkflowNodeOutGoingHook(ctx context.Context, dbFunc func() *gorp.DbMap, nodeRun *sdk.WorkflowNodeRun) error {
	db := dbFunc()
	if nodeRun.Callback == nil {
		nodeRun.Callback = new(sdk.WorkflowNodeOutgoingHookRunCallback)
	}
	nodeRun.Callback.Done = time.Now()
	nodeRun.Callback.Log += "\nStopped"
	nodeRun.Callback.Status = sdk.StatusStopped
	nodeRun.Status = nodeRun.Callback.Status

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return sdk.WrapError(err, "unable to get hooks services")
	}

	if nodeRun.HookExecutionID != "" {
		path := fmt.Sprintf("/task/%s/execution/%d/stop", nodeRun.HookExecutionID, nodeRun.HookExecutionTimeStamp)
		if _, _, err := services.NewClient(db, srvs).DoJSONRequest(ctx, "POST", path, nil, nil); err != nil {
			return sdk.WrapError(err, "unable to stop task execution")
		}
	}

	nodeRun.Status = sdk.StatusStopped
	nodeRun.Done = time.Now()
	if errU := UpdateNodeRun(dbFunc(), nodeRun); errU != nil {
		return sdk.WrapError(errU, "stopWorkflowNodePipeline> Cannot update node run")
	}
	return nil
}

// StopWorkflowNodeRun to stop a workflow node run with a specific spawn info
func StopWorkflowNodeRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj sdk.Project, workflowRun sdk.WorkflowRun, workflowNodeRun sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo) (*ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.StopWorkflowNodeRun")
	defer end()

	report := new(ProcessorReport)

	if workflowNodeRun.Stages != nil && len(workflowNodeRun.Stages) > 0 {
		r, err := stopWorkflowNodePipeline(ctx, dbFunc, store, proj, &workflowNodeRun, stopInfos)
		report.Merge(ctx, r)
		if err != nil {
			return report, sdk.WrapError(err, "unable to stop workflow node run")
		}
	}
	if workflowNodeRun.OutgoingHook != nil {
		if err := stopWorkflowNodeOutGoingHook(ctx, dbFunc, &workflowNodeRun); err != nil {
			return report, sdk.WrapError(err, "unable to stop workflow node run")
		}
	}
	report.Add(ctx, workflowNodeRun)

	// If current node has a mutex, we want to trigger another node run that can be waiting for the mutex
	workflowNode := workflowRun.Workflow.WorkflowData.NodeByID(workflowNodeRun.WorkflowNodeID)
	hasMutex := workflowNode != nil && workflowNode.Context != nil && workflowNode.Context.Mutex
	if hasMutex {
		tx, err := dbFunc().Begin()
		if err != nil {
			return report, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		r, err := releaseMutex(ctx, tx, store, proj, workflowNodeRun.WorkflowID, workflowNodeRun.WorkflowNodeName)
		report.Merge(ctx, r)
		if err != nil {
			return report, err
		}

		if err := tx.Commit(); err != nil {
			return report, err
		}
	}

	return report, nil
}

// stopWorkflowNodeRunStages mark to stop all stages and step status in struct
func stopWorkflowNodeRunStages(ctx context.Context, db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun) {
	// Update stages from node run
	for iS := range nodeRun.Stages {
		stag := &nodeRun.Stages[iS]
		for iR := range stag.RunJobs {
			runj := &stag.RunJobs[iR]
			spawnInfos, err := LoadNodeRunJobInfo(ctx, db, nodeRun.ID, runj.ID)
			if err != nil {
				log.Warn(ctx, "unable to load spawn infos for runj ID: %d", runj.ID)
			} else {
				runj.SpawnInfos = spawnInfos
			}

			if !sdk.StatusIsTerminated(runj.Status) {
				runj.Status = sdk.StatusStopped
				runj.Done = time.Now()
			}
			for iStep := range runj.Job.StepStatus {
				stepStat := &runj.Job.StepStatus[iStep]
				if !sdk.StatusIsTerminated(stepStat.Status) {
					stepStat.Status = sdk.StatusStopped
					stepStat.Done = time.Now()
				}
			}
		}
		if !sdk.StatusIsTerminated(stag.Status) {
			stag.Status = sdk.StatusStopped
		}
	}
}

func stopWorkflowNodeJobRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj sdk.Project, stopInfos sdk.SpawnInfo, chanNjrID <-chan int64, chanResult chan<- stopNodeJobRunResult) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.stopWorkflowNodeJobRun")
	defer end()

	for njrID := range chanNjrID {
		tx, err := dbFunc().Begin()
		if err != nil {
			chanResult <- stopNodeJobRunResult{err: sdk.WrapError(err, "cannot create transaction")}
			continue
		}

		njr, err := LoadAndLockNodeJobRunWait(ctx, tx, store, njrID)
		if err != nil {
			chanResult <- stopNodeJobRunResult{err: sdk.WrapError(err, "cannot load node job run id")}
			tx.Rollback() // nolint
			continue
		}

		if err := AddSpawnInfosNodeJobRun(tx, njr.WorkflowNodeRunID, njr.ID, []sdk.SpawnInfo{stopInfos}); err != nil {
			chanResult <- stopNodeJobRunResult{err: sdk.WrapError(err, "cannot save spawn info job %d", njr.ID)}
			tx.Rollback() // nolint
			continue
		}

		var res stopNodeJobRunResult

		njr.SpawnInfos = append(njr.SpawnInfos, stopInfos)
		r, err := UpdateNodeJobRunStatus(ctx, tx, store, proj, njr, sdk.StatusStopped)
		res.report = r
		if err != nil {
			res.err = sdk.WrapError(err, "cannot update node job run")
			chanResult <- res
			tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			res.err = sdk.WithStack(err)
			chanResult <- res
			tx.Rollback() // nolint
			continue
		}

		chanResult <- res
	}
}

// SyncNodeRunRunJob sync step status and spawnInfos in a specific run job
func SyncNodeRunRunJob(ctx context.Context, db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun, nodeJobRun sdk.WorkflowNodeJobRun) (bool, error) {
	var end func()
	_, end = telemetry.Span(ctx, "workflow.SyncNodeRunRunJob")
	defer end()

	found := false
	for i := range nodeRun.Stages {
		s := &nodeRun.Stages[i]
		for j := range s.RunJobs {
			runJob := &s.RunJobs[j]
			if runJob.ID == nodeJobRun.ID {
				spawnInfos, err := LoadNodeRunJobInfo(ctx, db, nodeRun.ID, runJob.ID)
				if err != nil {
					return false, sdk.WrapError(err, "unable to load spawn infos for runJobID: %d", runJob.ID)
				}
				runJob.SpawnInfos = spawnInfos
				runJob.Job.StepStatus = nodeJobRun.Job.StepStatus
				found = true
				break
			}
		}
	}

	return found, nil
}

type vcsInfos struct {
	Repository string
	Tag        string
	Branch     string
	Hash       string
	Author     string
	Message    string
	URL        string
	HTTPUrl    string
	Server     string
}

func (i vcsInfos) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", i.Server, i.Repository, i.Branch, i.Hash)
}

func getVCSInfos(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, projectKey string, gitValues map[string]string, applicationName, applicationVCSServer, applicationRepositoryFullname string) (*vcsInfos, error) {
	var vcsInfos vcsInfos
	vcsInfos.Repository = gitValues[tagGitRepository]
	vcsInfos.Branch = gitValues[tagGitBranch]
	vcsInfos.Tag = gitValues[tagGitTag]
	vcsInfos.Hash = gitValues[tagGitHash]
	vcsInfos.Author = gitValues[tagGitAuthor]
	vcsInfos.Message = gitValues[tagGitMessage]
	vcsInfos.URL = gitValues[tagGitURL]
	vcsInfos.HTTPUrl = gitValues[tagGitHTTPURL]
	vcsInfos.Server = applicationVCSServer

	if applicationName == "" || applicationVCSServer == "" {
		return &vcsInfos, nil
	}

	// START OBSERVABILITY
	ctx, end := telemetry.Span(ctx, "workflow.getVCSInfos",
		telemetry.Tag("application", applicationName),
		telemetry.Tag("vcs_server", applicationVCSServer),
		telemetry.Tag("vcs_repo", applicationRepositoryFullname),
	)
	defer end()

	// Try to get the data from cache
	cacheKey := cache.Key("api:workflow:getVCSInfos:", applicationVCSServer, applicationRepositoryFullname, vcsInfos.String())
	find, err := store.Get(cacheKey, &vcsInfos)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
	}
	if find && vcsInfos.Branch != "" && vcsInfos.Hash != "" {
		log.Debug(ctx, "completeVCSInfos> load from cache: %s", cacheKey)
		return &vcsInfos, nil
	}

	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, projectKey, applicationVCSServer)
	if errclient != nil {
		return nil, sdk.WrapError(errclient, "cannot get client")
	}

	// Check repository value
	if vcsInfos.Repository == "" {
		vcsInfos.Repository = applicationRepositoryFullname
	} else if !strings.EqualFold(vcsInfos.Repository, applicationRepositoryFullname) {
		//The input repository is not the same as the application, we have to check if it is a fork
		forks, err := client.ListForks(ctx, applicationRepositoryFullname)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot get forks for %s", applicationRepositoryFullname)
		}
		var forkFound bool
		for _, fork := range forks {
			if vcsInfos.Repository == fork.Fullname {
				forkFound = true
				break
			}
		}

		//If it's not a fork; reset this value to the application repository
		if !forkFound {
			return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "repository %s not found", vcsInfos.Repository)
		}
	}

	// RETRIEVE URL
	repo, err := client.RepoByFullname(ctx, vcsInfos.Repository)
	if err != nil {
		return nil, sdk.NewError(sdk.ErrNotFound, err)
	}
	vcsInfos.URL = repo.SSHCloneURL
	vcsInfos.HTTPUrl = repo.HTTPCloneURL

	switch {
	case vcsInfos.Branch == "" && vcsInfos.Hash == "":
		// Get default branch
		defaultB, errD := repositoriesmanager.DefaultBranch(ctx, client, vcsInfos.Repository)
		if errD != nil {
			return nil, errD
		}
		vcsInfos.Branch = defaultB.DisplayID
		vcsInfos.Hash = defaultB.LatestCommit
	case vcsInfos.Hash == "" && vcsInfos.Branch != "":
		// GET COMMIT INFO
		branch, errB := client.Branch(ctx, vcsInfos.Repository, sdk.VCSBranchFilters{BranchName: vcsInfos.Branch})
		if errB != nil {
			// Try default branch
			b, errD := repositoriesmanager.DefaultBranch(ctx, client, vcsInfos.Repository)
			if errD != nil {
				return nil, errD
			}
			branch = &b
			vcsInfos.Branch = branch.DisplayID
		}
		vcsInfos.Hash = branch.LatestCommit
	}

	// Get commit info if needed
	if vcsInfos.Hash != "" && (vcsInfos.Author == "" || vcsInfos.Message == "") {
		commit, errCm := client.Commit(ctx, vcsInfos.Repository, vcsInfos.Hash)
		if errCm != nil {
			return nil, sdk.WrapError(errCm, "cannot get commit infos for %s %s", vcsInfos.Repository, vcsInfos.Hash)
		}
		vcsInfos.Author = commit.Author.Name
		vcsInfos.Message = commit.Message
	}

	if err := store.Set(cacheKey, vcsInfos); err != nil {
		log.Error(ctx, "unable to cache set %v: %v", cacheKey, err)
	}
	return &vcsInfos, nil
}
