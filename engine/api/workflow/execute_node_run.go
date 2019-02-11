package workflow

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func syncTakeJobInNodeRun(ctx context.Context, db gorp.SqlExecutor, n *sdk.WorkflowNodeRun, j *sdk.WorkflowNodeJobRun, stageIndex int) (*ProcessorReport, error) {
	_, end := observability.Span(ctx, "workflow.syncTakeJobInNodeRun")
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
			rj.ContainsService = j.ContainsService
			rj.Job = j.Job
			rj.Header = j.Header
		}
		if rj.Status != sdk.StatusStopped.String() {
			isStopped = false
		}
	}
	if isStopped {
		nodeUpdated = true
		stage.Status = sdk.StatusStopped
	}

	if n.Status == sdk.StatusWaiting.String() {
		nodeUpdated = true
		n.Status = sdk.StatusBuilding.String()
	}

	if nodeUpdated {
		report.Add(*n)
	}

	// Save the node run in database
	if err := updateNodeRunStatusAndStage(db, n); err != nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s. err:%s", n.ID, n.Status, err), "workflow.syncTakeJobInNodeRun> Unable to execute node")
	}
	return report, nil
}

func execute(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, nr *sdk.WorkflowNodeRun, runContext nodeRunContext) (*ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.execute",
		observability.Tag(observability.TagWorkflowRun, nr.Number),
		observability.Tag(observability.TagWorkflowNodeRun, nr.ID),
		observability.Tag("workflow_node_run_status", nr.Status),
	)
	defer end()
	wr, errWr := LoadRunByID(db, nr.WorkflowRunID, LoadRunOptions{})
	if errWr != nil {
		return nil, sdk.WrapError(errWr, "workflow.execute> unable to load workflow run ID %d", nr.WorkflowRunID)
	}

	report := new(ProcessorReport)
	defer func(wNr *sdk.WorkflowNodeRun) {
		report.Add(*wNr)
	}(nr)

	//If status is not waiting neither build: nothing to do
	if sdk.StatusIsTerminated(nr.Status) {
		return nil, nil
	}

	var newStatus = nr.Status

	//If no stages ==> success
	if len(nr.Stages) == 0 {
		newStatus = sdk.StatusSuccess.String()
		nr.Done = time.Now()
	}

	stagesTerminated := 0
	//Browse stages
	for stageIndex := range nr.Stages {
		stage := &nr.Stages[stageIndex]
		log.Debug("workflow.execute> checking stage %s (status=%s)", stage.Name, stage.Status)
		//Initialize stage status at waiting
		if stage.Status.String() == "" {
			stage.Status = sdk.StatusWaiting

			if stageIndex == 0 {
				newStatus = sdk.StatusWaiting.String()
			}

			if len(stage.Jobs) == 0 {
				stage.Status = sdk.StatusSuccess
			} else {
				//Add job to Queue
				//Insert data in workflow_node_run_job
				log.Debug("workflow.execute> stage %s call addJobsToQueue", stage.Name)
				var err error
				report, err = report.Merge(addJobsToQueue(ctx, db, stage, wr, nr, runContext))
				if err != nil {
					return report, err
				}
			}

			if sdk.StatusIsTerminated(stage.Status.String()) {
				stagesTerminated++
				continue
			}
			break
		}

		if sdk.StatusIsTerminated(stage.Status.String()) {
			stagesTerminated++
		}

		//If stage is waiting, nothing to do
		if stage.Status == sdk.StatusWaiting {
			log.Debug("workflow.execute> stage %s status:%s - nothing to do", stage.Name, stage.Status)
			break
		}

		if stage.Status == sdk.StatusBuilding {
			newStatus = sdk.StatusBuilding.String()
			var end bool

			_, next := observability.Span(ctx, "workflow.syncStage")
			end, errSync := syncStage(db, store, stage)
			next()
			if errSync != nil {
				return report, errSync
			}
			if !end {
				break
			} else {
				//The stage is over
				if stage.Status == sdk.StatusFail {
					nr.Done = time.Now()
					newStatus = sdk.StatusFail.String()
					stagesTerminated++
					break
				}
				if stage.Status == sdk.StatusStopped {
					nr.Done = time.Now()
					newStatus = sdk.StatusStopped.String()
					stagesTerminated++
					break
				}

				if sdk.StatusIsTerminated(stage.Status.String()) {
					stagesTerminated++
					nr.Done = time.Now()
				}

				if stageIndex == len(nr.Stages)-1 {
					nr.Done = time.Now()
					newStatus = sdk.StatusSuccess.String()
					stagesTerminated++
					break
				}
				if stageIndex != len(nr.Stages)-1 {
					continue
				}
			}
		}
	}

	if stagesTerminated >= len(nr.Stages) || (stagesTerminated >= len(nr.Stages)-1 && (nr.Stages[len(nr.Stages)-1].Status == sdk.StatusDisabled || nr.Stages[len(nr.Stages)-1].Status == sdk.StatusSkipped)) {
		var counterStatus statusCounter
		if len(nr.Stages) > 0 {
			for _, stage := range nr.Stages {
				computeRunStatus(stage.Status.String(), &counterStatus)
			}
			newStatus = getRunStatus(counterStatus)
		}
	}

	nr.Status = newStatus

	if sdk.StatusIsTerminated(nr.Status) && nr.Status != sdk.StatusNeverBuilt.String() {
		nr.Done = time.Now()
	}

	// Save the node run in database
	if err := updateNodeRunStatusAndStage(db, nr); err != nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s. err:%s", nr.ID, nr.Status, err), "workflow.execute> Unable to execute node")
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(db, nr.WorkflowRunID, LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to reload workflow run id=%d", nr.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	//Delete jobs only when node is over
	if sdk.StatusIsTerminated(nr.Status) {
		if nr.Status != sdk.StatusStopped.String() {
			r1, _, err := processWorkflowDataRun(ctx, db, store, proj, updatedWorkflowRun, nil, nil, nil)
			if err != nil {
				return nil, sdk.WrapError(err, "Unable to reprocess workflow !")
			}
			report, _ = report.Merge(r1, nil)

		}

		//Delete the line in workflow_node_run_job
		if err := DeleteNodeJobRuns(db, nr.ID); err != nil {
			return nil, sdk.WrapError(err, "Unable to delete node %d job runs ", nr.ID)
		}

		var hasMutex bool
		var nodeName string

		node := updatedWorkflowRun.Workflow.WorkflowData.NodeByID(nr.WorkflowNodeID)
		if node != nil && node.Context != nil && node.Context.Mutex {
			hasMutex = node.Context.Mutex
			nodeName = node.Name
		}

		//Do we release a mutex ?
		//Try to find one node run of the same node from the same workflow at status Waiting
		if hasMutex {
			_, next := observability.Span(ctx, "workflow.releaseMutex")

			mutexQuery := `select workflow_node_run.id
			from workflow_node_run
			join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
			join workflow on workflow.id = workflow_run.workflow_id
			where workflow.id = $1
			and workflow_node_run.workflow_node_name = $2
			and workflow_node_run.status = $3
			order by workflow_node_run.start asc
			limit 1`
			waitingRunID, errID := db.SelectInt(mutexQuery, updatedWorkflowRun.WorkflowID, nodeName, string(sdk.StatusWaiting))
			if errID != nil && errID != sql.ErrNoRows {
				log.Error("workflow.execute> Unable to load mutex-locked workflow node run ID: %v", errID)
				return report, nil
			}
			//If not more run is found, stop the loop
			if waitingRunID == 0 {
				return report, nil
			}
			waitingRun, errRun := LoadNodeRunByID(db, waitingRunID, LoadRunOptions{})
			if errRun != nil && sdk.Cause(errRun) != sql.ErrNoRows {
				log.Error("workflow.execute> Unable to load mutex-locked workflow rnode un: %v", errRun)
				return report, nil
			}
			//If not more run is found, stop the loop
			if waitingRun == nil {
				return report, nil
			}

			//Here we are loading another workflow run
			workflowRun, errWRun := LoadRunByID(db, waitingRun.WorkflowRunID, LoadRunOptions{})
			if errWRun != nil {
				log.Error("workflow.execute> Unable to load mutex-locked workflow rnode un: %v", errWRun)
				return report, nil
			}
			AddWorkflowRunInfo(workflowRun, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutexRelease.ID,
				Args: []interface{}{waitingRun.WorkflowNodeName},
			})

			if err := UpdateWorkflowRun(ctx, db, workflowRun); err != nil {
				return nil, sdk.WrapError(err, "Unable to update workflow run %d after mutex release", workflowRun.ID)
			}

			log.Debug("workflow.execute> process the node run %d because mutex has been released", waitingRun.ID)
			var err error
			report, err = report.Merge(execute(ctx, db, store, proj, waitingRun, runContext))
			if err != nil {
				return nil, sdk.WrapError(err, "Unable to reprocess workflow")
			}

			next()
		}
	}
	return report, nil
}

func addJobsToQueue(ctx context.Context, db gorp.SqlExecutor, stage *sdk.Stage, wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, runContext nodeRunContext) (*ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.addJobsToQueue")
	defer end()

	report := new(ProcessorReport)

	_, next := observability.Span(ctx, "sdk.WorkflowCheckConditions")
	conditionsOK, err := sdk.WorkflowCheckConditions(stage.Conditions(), run.BuildParameters)
	next()
	if err != nil {
		return report, sdk.WrapError(err, "Cannot compute prerequisites on stage %s(%d)", stage.Name, stage.ID)
	}

	if !conditionsOK {
		stage.Status = sdk.StatusSkipped
	}
	if !stage.Enabled {
		stage.Status = sdk.StatusDisabled
	}

	_, next = observability.Span(ctx, "workflow.getIntegrationPluginBinaries")
	integrationPluginBinaries, err := getIntegrationPluginBinaries(db, runContext)
	if err != nil {
		return report, sdk.WrapError(err, "unable to get integration plugins requirement")
	}
	next()

	_, next = observability.Span(ctx, "workflow.getJobExecutablesGroups")
	groups, errGroups := getJobExecutablesGroups(wr, runContext)
	if errGroups != nil {
		return report, sdk.WrapError(errGroups, "addJobsToQueue> error on getJobExecutablesGroups")
	}
	next()

	skippedOrDisabledJobs := 0
	//Browse the jobs
	for j := range stage.Jobs {
		job := &stage.Jobs[j]
		errs := sdk.MultiError{}
		//Process variables for the jobs
		_, next = observability.Span(ctx, "workflow..getNodeJobRunParameters")
		jobParams, errParam := getNodeJobRunParameters(db, *job, run, stage)
		next()

		if errParam != nil {
			errs.Join(*errParam)
		}

		_, next = observability.Span(ctx, "workflow.getNodeJobRunRequirements")
		jobRequirements, containsService, modelType, errReq := getNodeJobRunRequirements(db, *job, run)
		next()

		if errReq != nil {
			errs.Join(*errReq)
		}

		// add requirements in job parameters, to use them as {{.job.requirement...}} in job
		_, next = observability.Span(ctx, "workflow.prepareRequirementsToNodeJobRunParameters")
		jobParams = append(jobParams, prepareRequirementsToNodeJobRunParameters(jobRequirements)...)
		next()

		if errGroups != nil {
			return report, sdk.WrapError(errGroups, "addJobsToQueue> error on getJobExecutablesGroups")
		}

		//Create the job run
		wjob := sdk.WorkflowNodeJobRun{
			ProjectID:                 wr.ProjectID,
			WorkflowNodeRunID:         run.ID,
			Start:                     time.Time{},
			Queued:                    time.Now(),
			Status:                    sdk.StatusWaiting.String(),
			Parameters:                jobParams,
			ExecGroups:                groups,
			IntegrationPluginBinaries: integrationPluginBinaries,
			Job: sdk.ExecutedJob{
				Job: *job,
			},
			Header:          run.Header,
			ContainsService: containsService,
			ModelType:       modelType,
		}
		wjob.Job.Job.Action.Requirements = jobRequirements // Set the interpolated requirements on the job run only

		if !stage.Enabled || !wjob.Job.Enabled {
			wjob.Status = sdk.StatusDisabled.String()
			skippedOrDisabledJobs++
		} else if !conditionsOK {
			wjob.Status = sdk.StatusSkipped.String()
			skippedOrDisabledJobs++
		}

		if errParam != nil {
			wjob.Status = sdk.StatusFail.String()
			spawnInfos := sdk.SpawnMsg{
				ID: sdk.MsgSpawnInfoJobError.ID,
			}

			for _, e := range *errParam {
				spawnInfos.Args = append(spawnInfos.Args, e.Error())
			}

			wjob.SpawnInfos = []sdk.SpawnInfo{sdk.SpawnInfo{
				APITime:    time.Now(),
				Message:    spawnInfos,
				RemoteTime: time.Now(),
			}}
		} else {
			wjob.SpawnInfos = []sdk.SpawnInfo{sdk.SpawnInfo{
				APITime:    time.Now(),
				Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobInQueue.ID},
				RemoteTime: time.Now(),
			}}
		}

		//Insert in database
		_, next = observability.Span(ctx, "workflow.insertWorkflowNodeJobRun")
		if err := insertWorkflowNodeJobRun(db, &wjob); err != nil {
			next()
			return report, sdk.WrapError(err, "Unable to insert in table workflow_node_run_job")
		}
		next()

		if err := AddSpawnInfosNodeJobRun(db, wjob.ID, PrepareSpawnInfos(wjob.SpawnInfos)); err != nil {
			return nil, sdk.WrapError(err, "Cannot save spawn info job %d", wjob.ID)
		}

		//Put the job run in database
		stage.RunJobs = append(stage.RunJobs, wjob)

		report.Add(wjob)
	}

	if skippedOrDisabledJobs == len(stage.Jobs) {
		stage.Status = sdk.StatusSkipped
	}

	return report, nil
}

func getIntegrationPluginBinaries(db gorp.SqlExecutor, runContext nodeRunContext) ([]sdk.GRPCPluginBinary, error) {
	if runContext.ProjectIntegration.Model.ID > 0 {
		plugin, err := plugin.LoadByIntegrationModelIDAndType(db, runContext.ProjectIntegration.Model.ID, sdk.GRPCPluginDeploymentIntegration)
		if err != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "Cannot find plugin for integration model id %d, %v", runContext.ProjectIntegration.Model.ID, err)
		}
		return plugin.Binaries, nil
	}
	return nil, nil
}

func getJobExecutablesGroups(wr *sdk.WorkflowRun, runContext nodeRunContext) ([]sdk.Group, error) {
	var groups []sdk.Group

	if len(runContext.NodeGroups) > 0 {
		for _, gp := range runContext.NodeGroups {
			if gp.Permission >= permission.PermissionReadExecute {
				groups = append(groups, gp.Group)
			}
		}
	} else {
		for _, gp := range wr.Workflow.Groups {
			if gp.Permission >= permission.PermissionReadExecute {
				groups = append(groups, gp.Group)
			}
		}
	}
	groups = append(groups, *group.SharedInfraGroup)

	return groups, nil
}

func syncStage(db gorp.SqlExecutor, store cache.Store, stage *sdk.Stage) (bool, error) {
	stageEnd := true
	finalStatus := sdk.StatusBuilding

	// browse all running jobs
	for indexJob := range stage.RunJobs {
		runJob := &stage.RunJobs[indexJob]
		// If job is runnning, sync it
		if runJob.Status == sdk.StatusBuilding.String() || runJob.Status == sdk.StatusWaiting.String() {
			runJobDB, errJob := LoadNodeJobRun(db, store, runJob.ID)
			if errJob != nil {
				return stageEnd, errJob
			}

			if runJobDB.Status == sdk.StatusBuilding.String() || runJobDB.Status == sdk.StatusWaiting.String() {
				stageEnd = false
			}
			spawnInfos, err := LoadNodeRunJobInfo(db, runJob.ID)
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
				runJob.ContainsService = runJobDB.ContainsService
				runJob.Job = runJobDB.Job
			}
		}
	}
	log.Debug("syncStage> stage %s stageEnd:%t len(stage.RunJobs):%d", stage.Name, stageEnd, len(stage.RunJobs))

	if stageEnd || len(stage.RunJobs) == 0 {
		finalStatus = sdk.StatusSuccess
		stageEnd = true
		// Determine final stage status
	finalStageLoop:
		for _, runJob := range stage.RunJobs {
			switch runJob.Status {
			case sdk.StatusDisabled.String():
				if finalStatus == sdk.StatusBuilding {
					finalStatus = sdk.StatusDisabled
				}
			case sdk.StatusSkipped.String():
				if finalStatus == sdk.StatusBuilding || finalStatus == sdk.StatusDisabled {
					finalStatus = sdk.StatusSkipped
				}
			case sdk.StatusFail.String():
				finalStatus = sdk.StatusFail
				break finalStageLoop
			case sdk.StatusSuccess.String():
				if finalStatus != sdk.StatusFail {
					finalStatus = sdk.StatusSuccess
				}
			case sdk.StatusStopped.String():
				if finalStatus != sdk.StatusFail {
					finalStatus = sdk.StatusStopped
				}
			}
		}
	}
	log.Debug("syncStage> set stage %s from %s to %s", stage.Name, stage.Status, finalStatus)
	stage.Status = finalStatus
	return stageEnd, nil
}

// NodeBuildParametersFromRun return build parameters from previous workflow run
func NodeBuildParametersFromRun(wr sdk.WorkflowRun, id int64) ([]sdk.Parameter, error) {
	params := []sdk.Parameter{}

	nodesRun, ok := wr.WorkflowNodeRuns[id]
	if !ok || len(nodesRun) == 0 {
		return params, nil
	}

	for _, p := range nodesRun[0].BuildParameters {
		sdk.AddParameter(&params, p.Name, p.Type, p.Value)
	}

	return params, nil
}

//NodeBuildParametersFromWorkflow returns build_parameters for a node given its id
func NodeBuildParametersFromWorkflow(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, refNode *sdk.Node, ancestorsIds []int64) ([]sdk.Parameter, error) {
	runContext := nodeRunContext{}
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
				runContext.ProjectIntegration = pp
			}
		}
		runContext.NodeGroups = refNode.Groups
	}

	res := []sdk.Parameter{}
	if len(res) == 0 {
		var err error
		res, err = GetNodeBuildParameters(proj, wf, runContext, refNode.Context.DefaultPipelineParameters, refNode.Context.DefaultPayload, nil)
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
	if wf.Root.Context.DefaultPayload != nil {
		e := dump.NewDefaultEncoder(new(bytes.Buffer))
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false

		tempParams := sdk.ParametersToMap(res)
		m1, errm1 := e.ToStringMap(wf.Root.Context.DefaultPayload)
		if errm1 == nil {
			mergedParameters := sdk.ParametersMapMerge(tempParams, m1, sdk.MapMergeOptions.ExcludeGitParams)
			res = sdk.ParametersFromMap(mergedParameters)
		}
	}

	return res, nil
}

func stopWorkflowNodePipeline(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj *sdk.Project, nodeRun *sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo) (*ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.stopWorkflowNodePipeline")
	defer end()

	report := new(ProcessorReport)

	const stopWorkflowNodeRunNBWorker = 5
	var wg sync.WaitGroup
	// Load node job run ID
	ids, errIDS := LoadNodeJobRunIDByNodeRunID(dbFunc(), nodeRun.ID)
	if errIDS != nil {
		return report, sdk.WrapError(errIDS, "stopWorkflowNodePipeline> Cannot load node jobs run ids ")
	}

	chanNjrID := make(chan int64, stopWorkflowNodeRunNBWorker)
	chanNodeJobRunDone := make(chan bool, stopWorkflowNodeRunNBWorker)
	chanErr := make(chan error, stopWorkflowNodeRunNBWorker)
	for i := 0; i < stopWorkflowNodeRunNBWorker && i < len(ids); i++ {
		go func() {
			//since report is mutable and is a pointer and in this case we can't have any error, we can skip returned values
			_, _ = report.Merge(stopWorkflowNodeJobRun(ctx, dbFunc, store, proj, nodeRun, stopInfos, chanNjrID, chanErr, chanNodeJobRunDone, &wg), nil)
		}()
	}

	wg.Add(len(ids))
	for _, njrID := range ids {
		chanNjrID <- njrID
	}
	close(chanNjrID)

	for i := 0; i < len(ids); i++ {
		select {
		case <-chanNodeJobRunDone:
		case err := <-chanErr:
			return report, err
		}
	}
	wg.Wait()

	tx, errTx := dbFunc().Begin()
	if errTx != nil {
		return nil, sdk.WrapError(errTx, "stopWorkflowNodePipeline> Unable to create transaction")
	}
	defer tx.Rollback() //nolint

	// Update stages from node run
	stopWorkflowNodeRunStages(tx, nodeRun)

	nodeRun.Status = sdk.StatusStopped.String()
	nodeRun.Done = time.Now()
	if errU := UpdateNodeRun(tx, nodeRun); errU != nil {
		return report, sdk.WrapError(errU, "stopWorkflowNodePipeline> Cannot update node run")
	}
	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "stopWorkflowNodePipeline> Cannot commit transaction")
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
	nodeRun.Callback.Status = sdk.StatusStopped.String()
	nodeRun.Status = nodeRun.Callback.Status

	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return fmt.Errorf("unable to get hooks services: %v", err)
	}

	if nodeRun.HookExecutionID != "" {
		path := fmt.Sprintf("/task/%s/execution/%d/stop", nodeRun.HookExecutionID, nodeRun.HookExecutionTimeStamp)
		if _, err := services.DoJSONRequest(ctx, srvs, "POST", path, nil, nil); err != nil {
			return fmt.Errorf("unable to stop task execution: %v", err)
		}
	}

	nodeRun.Status = sdk.StatusStopped.String()
	nodeRun.Done = time.Now()
	if errU := UpdateNodeRun(dbFunc(), nodeRun); errU != nil {
		return sdk.WrapError(errU, "stopWorkflowNodePipeline> Cannot update node run")
	}
	return nil
}

// StopWorkflowNodeRun to stop a workflow node run with a specific spawn info
func StopWorkflowNodeRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj *sdk.Project, nodeRun sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo) (*ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.StopWorkflowNodeRun")
	defer end()

	report := new(ProcessorReport)

	var r1 *ProcessorReport
	var errS error
	if nodeRun.Stages != nil && len(nodeRun.Stages) > 0 {
		r1, errS = stopWorkflowNodePipeline(ctx, dbFunc, store, proj, &nodeRun, stopInfos)
	}
	if nodeRun.OutgoingHook != nil {
		errS = stopWorkflowNodeOutGoingHook(ctx, dbFunc, &nodeRun)
	}

	if errS != nil {
		return report, sdk.WrapError(errS, "Unable to stop workflow node run")
	}

	report.Merge(r1, nil) // nolint
	report.Add(nodeRun)

	return report, nil
}

// stopWorkflowNodeRunStages mark to stop all stages and step status in struct
func stopWorkflowNodeRunStages(db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun) {
	// Update stages from node run
	for iS := range nodeRun.Stages {
		stag := &nodeRun.Stages[iS]
		for iR := range stag.RunJobs {
			runj := &stag.RunJobs[iR]
			spawnInfos, err := LoadNodeRunJobInfo(db, runj.ID)
			if err != nil {
				log.Warning("unable to load spawn infos for runj ID: %d", runj.ID)
			} else {
				runj.SpawnInfos = spawnInfos
			}

			if !sdk.StatusIsTerminated(runj.Status) {
				runj.Status = sdk.StatusStopped.String()
				runj.Done = time.Now()
			}
			for iStep := range runj.Job.StepStatus {
				stepStat := &runj.Job.StepStatus[iStep]
				if !sdk.StatusIsTerminated(stepStat.Status) {
					stepStat.Status = sdk.StatusStopped.String()
					stepStat.Done = time.Now()
				}
			}
		}
		if !sdk.StatusIsTerminated(stag.Status.String()) {
			stag.Status = sdk.StatusStopped
		}
	}
}

func stopWorkflowNodeJobRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, proj *sdk.Project, nodeRun *sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo, chanNjrID <-chan int64, chanErr chan<- error, chanDone chan<- bool, wg *sync.WaitGroup) *ProcessorReport {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.stopWorkflowNodeJobRun")
	defer end()

	report := new(ProcessorReport)

	for njrID := range chanNjrID {
		tx, errTx := dbFunc().Begin()
		if errTx != nil {
			chanErr <- sdk.WrapError(errTx, "StopWorkflowNodeRun> Cannot create transaction")
			wg.Done()
			return report
		}

		njr, errNRJ := LoadAndLockNodeJobRunWait(tx, store, njrID)
		if errNRJ != nil {
			chanErr <- sdk.WrapError(errNRJ, "StopWorkflowNodeRun> Cannot load node job run id")
			tx.Rollback()
			wg.Done()
			return report
		}

		if err := AddSpawnInfosNodeJobRun(tx, njr.ID, []sdk.SpawnInfo{stopInfos}); err != nil {
			chanErr <- sdk.WrapError(err, "Cannot save spawn info job %d", njr.ID)
			tx.Rollback()
			wg.Done()
			return report
		}

		njr.SpawnInfos = append(njr.SpawnInfos, stopInfos)
		if _, err := report.Merge(UpdateNodeJobRunStatus(ctx, dbFunc, tx, store, proj, njr, sdk.StatusStopped)); err != nil {
			chanErr <- sdk.WrapError(err, "Cannot update node job run")
			tx.Rollback()
			wg.Done()
			return report
		}

		if err := tx.Commit(); err != nil {
			chanErr <- sdk.WrapError(err, "Cannot commit transaction")
			tx.Rollback()
			wg.Done()
			return report
		}
		chanDone <- true
		wg.Done()
	}
	return report
}

// SyncNodeRunRunJob sync step status and spawnInfos in a specific run job
func SyncNodeRunRunJob(ctx context.Context, db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun, nodeJobRun sdk.WorkflowNodeJobRun) (bool, error) {
	var end func()
	_, end = observability.Span(ctx, "workflow.SyncNodeRunRunJob")
	defer end()

	found := false
	for i := range nodeRun.Stages {
		s := &nodeRun.Stages[i]
		for j := range s.RunJobs {
			runJob := &s.RunJobs[j]
			if runJob.ID == nodeJobRun.ID {
				spawnInfos, err := LoadNodeRunJobInfo(db, runJob.ID)
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

func getVCSInfos(ctx context.Context, db gorp.SqlExecutor, store cache.Store, vcsServer *sdk.ProjectVCSServer, gitValues map[string]string, applicationName, applicationVCSServer, applicationRepositoryFullname string, isChildNode bool, previousGitRepo string) (i vcsInfos, err error) {
	var vcsInfos vcsInfos
	vcsInfos.Repository = gitValues[tagGitRepository]
	vcsInfos.Branch = gitValues[tagGitBranch]
	vcsInfos.Tag = gitValues[tagGitTag]
	vcsInfos.Hash = gitValues[tagGitHash]
	vcsInfos.Author = gitValues[tagGitAuthor]
	vcsInfos.Message = gitValues[tagGitMessage]
	vcsInfos.URL = gitValues[tagGitURL]
	vcsInfos.HTTPUrl = gitValues[tagGitHTTPURL]

	if applicationName == "" || applicationVCSServer == "" {
		return vcsInfos, nil
	}

	ctx, end := observability.Span(ctx, "workflow.getVCSInfos",
		observability.Tag("application", applicationName),
		observability.Tag("vcs_server", applicationVCSServer),
		observability.Tag("vcs_repo", applicationRepositoryFullname),
	)
	defer end()

	if vcsServer == nil {
		return vcsInfos, nil
	}
	vcsInfos.Server = vcsServer.Name

	// Cache management, kind of memoization form gathered vcsInfos
	cacheKey := cache.Key("api:workflow:getVCSInfos:", applicationVCSServer, applicationRepositoryFullname, vcsInfos.String(), fmt.Sprintf("%v", isChildNode), previousGitRepo)
	// Try to get the data from cache
	if store.Get(cacheKey, &vcsInfos) {
		log.Debug("getVCSInfos> load from cache: %s", cacheKey)
		return vcsInfos, nil
	}
	// Store the result in the cache
	defer func() {
		if err == nil {
			store.Set(cacheKey, &i)
		}
	}()

	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, vcsServer)
	if errclient != nil {
		return vcsInfos, sdk.WrapError(errclient, "computeVCSInfos> Cannot get client")
	}

	// Set default values
	if vcsInfos.Repository == "" {
		vcsInfos.Repository = applicationRepositoryFullname
	} else if vcsInfos.Repository != applicationRepositoryFullname {
		//The input repository is not the same as the application, we have to check if it is a fork
		forks, err := client.ListForks(ctx, applicationRepositoryFullname)
		if err != nil {
			return vcsInfos, sdk.WrapError(err, "Cannot get forks for %s", applicationRepositoryFullname)
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
			if !isChildNode {
				return vcsInfos, sdk.NewError(sdk.ErrNotFound, fmt.Errorf("repository %s not found", vcsInfos.Repository))
			}
			vcsInfos.Hash = ""
			vcsInfos.Repository = applicationRepositoryFullname
			vcsInfos.Tag = ""
		}
	}

	//Get the url and http_url
	repo, err := client.RepoByFullname(ctx, vcsInfos.Repository)
	if err != nil {
		if !isChildNode {
			return vcsInfos, sdk.NewError(sdk.ErrNotFound, err)
		}
		//If we ignore errors
		vcsInfos.Repository = applicationRepositoryFullname
		repo, err = client.RepoByFullname(ctx, applicationRepositoryFullname)
		if err != nil {
			return vcsInfos, sdk.WrapError(err, "Cannot get repo %s", applicationRepositoryFullname)
		}
	}
	vcsInfos.URL = repo.SSHCloneURL
	vcsInfos.HTTPUrl = repo.HTTPCloneURL

	if vcsInfos.Branch == "" && !isChildNode {
		if vcsInfos.Tag != "" {
			return vcsInfos, nil
		}
		return vcsInfos, sdk.WrapError(sdk.ErrBranchNameNotProvided, "computeVCSInfos> should not have an empty branch")
	}

	branch, err := client.Branch(ctx, vcsInfos.Repository, vcsInfos.Branch)
	if err != nil {
		if !isChildNode {
			return vcsInfos, sdk.NewError(sdk.ErrBranchNameNotProvided, err)
		}
	}

	if branch == nil {
		log.Error("computeVCSInfos> unable to get branch %s - repository:%s - app:%s", vcsInfos.Branch, vcsInfos.Repository, applicationName)
		vcsInfos.Branch = ""
	}

	//Get the default branch
	if branch == nil {
		branches, errR := client.Branches(ctx, vcsInfos.Repository)
		if errR != nil {
			return vcsInfos, sdk.WrapError(errR, "computeVCSInfos> cannot get branches infos for %s", vcsInfos.Repository)
		}
		_branch := sdk.GetDefaultBranch(branches)
		branch = &_branch
		vcsInfos.Branch = branch.DisplayID
	}

	//Check if the branch is still valid
	if branch == nil && previousGitRepo != "" && previousGitRepo == applicationRepositoryFullname {
		return vcsInfos, sdk.WrapError(fmt.Errorf("branch has been deleted"), "computeVCSInfos> ")
	}

	if branch != nil && vcsInfos.Hash == "" {
		vcsInfos.Hash = branch.LatestCommit
	}

	//Get the latest commit
	commit, errCm := client.Commit(ctx, vcsInfos.Repository, vcsInfos.Hash)
	if errCm != nil {
		if !isChildNode {
			return vcsInfos, sdk.WrapError(errCm, "computeVCSInfos> cannot get commit infos for %s %s", vcsInfos.Repository, vcsInfos.Hash)
		}
		vcsInfos.Hash = branch.LatestCommit
		commit, errCm = client.Commit(ctx, vcsInfos.Repository, vcsInfos.Hash)
		if errCm != nil {
			return vcsInfos, sdk.WrapError(errCm, "computeVCSInfos> cannot get commit infos for %s %s", vcsInfos.Repository, vcsInfos.Hash)
		}
	}
	vcsInfos.Author = commit.Author.Name
	vcsInfos.Message = commit.Message

	return vcsInfos, nil
}
