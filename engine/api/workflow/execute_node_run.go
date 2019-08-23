package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
			rj.ContainsService = j.ContainsService
			rj.Job = j.Job
			rj.Header = j.Header
			rj.Parameters = j.Parameters
		}
	}
}

func syncTakeJobInNodeRun(ctx context.Context, db gorp.SqlExecutor, n *sdk.WorkflowNodeRun, j *sdk.WorkflowNodeJobRun, stageIndex int) (*ProcessorReport, error) {
	_, end := observability.Span(ctx, "workflow.syncTakeJobInNodeRun")
	defer end()

	log.Debug("workflow.syncTakeJobInNodeRun> job parameters= %+v", j.Parameters)

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
		newStatus = sdk.StatusSuccess
		nr.Done = time.Now()
	}

	stagesTerminated := 0
	//Browse stages
	for stageIndex := range nr.Stages {
		stage := &nr.Stages[stageIndex]
		log.Debug("workflow.execute> checking stage %s (status=%s)", stage.Name, stage.Status)
		//Initialize stage status at waiting
		if stage.Status == "" {
			stage.Status = sdk.StatusWaiting

			if stageIndex == 0 {
				newStatus = sdk.StatusWaiting
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
				log.Debug("workflow.execute> stage %s status after call to addJobsToQueue %s", stage.Name, stage.Status)
			}

			// check for failure caused by action not usable or requirements problem
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
			log.Debug("workflow.execute> stage %s status:%s - nothing to do", stage.Name, stage.Status)
			break
		}

		if stage.Status == sdk.StatusBuilding {
			newStatus = sdk.StatusBuilding
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
					newStatus = sdk.StatusFail
					stagesTerminated++
					break
				}
				if stage.Status == sdk.StatusStopped {
					nr.Done = time.Now()
					newStatus = sdk.StatusStopped
					stagesTerminated++
					break
				}

				if sdk.StatusIsTerminated(stage.Status) {
					stagesTerminated++
					nr.Done = time.Now()
				}

				if stageIndex == len(nr.Stages)-1 {
					nr.Done = time.Now()
					newStatus = sdk.StatusSuccess
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
				computeRunStatus(stage.Status, &counterStatus)
			}
			newStatus = getRunStatus(counterStatus)
		}
	}

	nr.Status = newStatus

	if sdk.StatusIsTerminated(nr.Status) && nr.Status != sdk.StatusNeverBuilt {
		nr.Done = time.Now()
	}

	// Save the node run in database
	if err := updateNodeRunStatusAndStage(db, nr); err != nil {
		return nil, sdk.WrapError(err, "unable to update node id=%d at status %s", nr.ID, nr.Status)
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(db, nr.WorkflowRunID, LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to reload workflow run id=%d", nr.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	//Delete jobs only when node is over
	if sdk.StatusIsTerminated(nr.Status) {
		if nr.Status != sdk.StatusStopped {
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

	_, next := observability.Span(ctx, "checkCondition")
	conditionsOK := checkCondition(wr, stage.Conditions, run.BuildParameters)
	next()
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
	groups, err := getJobExecutablesGroups(wr, runContext)
	if err != nil {
		return report, sdk.WrapError(err, "error getting job executables groups")
	}
	next()

	skippedOrDisabledJobs := 0
	failedJobs := 0
	//Browse the jobs
	for j := range stage.Jobs {
		job := &stage.Jobs[j]

		// errors generated in the loop will be added to job run spawn info
		spawnErrs := sdk.MultiError{}

		//Process variables for the jobs
		_, next = observability.Span(ctx, "workflow..getNodeJobRunParameters")
		jobParams, err := getNodeJobRunParameters(db, *job, run, stage)
		next()
		if err != nil {
			spawnErrs.Join(*err)
		}

		_, next = observability.Span(ctx, "workflow.processNodeJobRunRequirements")
		jobRequirements, containsService, wm, err := processNodeJobRunRequirements(ctx, db, *job, run, sdk.Groups(groups).ToIDs(), integrationPluginBinaries)
		next()
		if err != nil {
			spawnErrs.Join(*err)
		}

		// check that children actions used by job can be used by the project
		if err := action.CheckChildrenForGroupIDsWithLoop(ctx, db, &job.Action, sdk.Groups(groups).ToIDs()); err != nil {
			spawnErrs.Append(err)
		}

		// add requirements in job parameters, to use them as {{.job.requirement...}} in job
		_, next = observability.Span(ctx, "workflow.prepareRequirementsToNodeJobRunParameters")
		jobParams = append(jobParams, prepareRequirementsToNodeJobRunParameters(jobRequirements)...)
		next()

		//Create the job run
		wjob := sdk.WorkflowNodeJobRun{
			ProjectID:                 wr.ProjectID,
			WorkflowNodeRunID:         run.ID,
			Start:                     time.Time{},
			Queued:                    time.Now(),
			Status:                    sdk.StatusWaiting,
			Parameters:                jobParams,
			ExecGroups:                groups,
			IntegrationPluginBinaries: integrationPluginBinaries,
			Job: sdk.ExecutedJob{
				Job: *job,
			},
			Header:          run.Header,
			ContainsService: containsService,
		}
		if wm != nil {
			wjob.ModelType = wm.Type
		}
		wjob.Job.Job.Action.Requirements = jobRequirements // Set the interpolated requirements on the job run only

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
				msg := sdk.SpawnMsg{
					ID: sdk.MsgSpawnInfoJobError.ID,
				}
				msg.Args = []interface{}{sdk.Cause(e).Error()}
				wjob.SpawnInfos = append(wjob.SpawnInfos, sdk.SpawnInfo{
					APITime:    time.Now(),
					Message:    msg,
					RemoteTime: time.Now(),
				})
			}
		} else {
			wjob.SpawnInfos = []sdk.SpawnInfo{{
				APITime:    time.Now(),
				Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobInQueue.ID},
				RemoteTime: time.Now(),
			}}
		}

		// insert in database
		_, next = observability.Span(ctx, "workflow.insertWorkflowNodeJobRun")
		if err := insertWorkflowNodeJobRun(db, &wjob); err != nil {
			next()
			return report, sdk.WrapError(err, "unable to insert in table workflow_node_run_job")
		}
		next()

		if err := AddSpawnInfosNodeJobRun(db, wjob.ID, PrepareSpawnInfos(wjob.SpawnInfos)); err != nil {
			return nil, sdk.WrapError(err, "cannot save spawn info job %d", wjob.ID)
		}

		//Put the job run in database
		stage.RunJobs = append(stage.RunJobs, wjob)

		report.Add(wjob)
	}

	if skippedOrDisabledJobs == len(stage.Jobs) {
		stage.Status = sdk.StatusSkipped
	}

	if failedJobs > 0 {
		stage.Status = sdk.StatusFail
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

func syncStage(db gorp.SqlExecutor, store cache.Store, stage *sdk.Stage) (bool, error) {
	stageEnd := true
	finalStatus := sdk.StatusBuilding

	// browse all running jobs
	for indexJob := range stage.RunJobs {
		runJob := &stage.RunJobs[indexJob]

		// If job is runnning, sync it
		if runJob.Status == sdk.StatusBuilding || runJob.Status == sdk.StatusWaiting {
			runJobDB, errJob := LoadNodeJobRun(db, store, runJob.ID)
			if errJob != nil {
				return stageEnd, errJob
			}

			if runJobDB.Status == sdk.StatusBuilding || runJobDB.Status == sdk.StatusWaiting {
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
		res, err = GetBuildParameterFromNodeContext(proj, wf, runContext, refNode.Context.DefaultPipelineParameters, refNode.Context.DefaultPayload, nil)
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

	nodeRun.Status = sdk.StatusStopped
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
	nodeRun.Callback.Status = sdk.StatusStopped
	nodeRun.Status = nodeRun.Callback.Status

	srvs, err := services.LoadAllByType(ctx, db, services.TypeHooks)
	if err != nil {
		return fmt.Errorf("unable to get hooks services: %v", err)
	}

	if nodeRun.HookExecutionID != "" {
		path := fmt.Sprintf("/task/%s/execution/%d/stop", nodeRun.HookExecutionID, nodeRun.HookExecutionTimeStamp)
		if _, _, err := services.DoJSONRequest(ctx, db, srvs, "POST", path, nil, nil); err != nil {
			return fmt.Errorf("unable to stop task execution: %v", err)
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

func getVCSInfos(ctx context.Context, db gorp.SqlExecutor, store cache.Store, projectKey string, vcsServer *sdk.ProjectVCSServer, gitValues map[string]string, applicationName, applicationVCSServer, applicationRepositoryFullname string) (*vcsInfos, error) {
	var vcsInfos vcsInfos
	vcsInfos.Repository = gitValues[tagGitRepository]
	vcsInfos.Branch = gitValues[tagGitBranch]
	vcsInfos.Tag = gitValues[tagGitTag]
	vcsInfos.Hash = gitValues[tagGitHash]
	vcsInfos.Author = gitValues[tagGitAuthor]
	vcsInfos.Message = gitValues[tagGitMessage]
	vcsInfos.URL = gitValues[tagGitURL]
	vcsInfos.HTTPUrl = gitValues[tagGitHTTPURL]

	if vcsServer != nil {
		vcsInfos.Server = vcsServer.Name
	}

	if applicationName == "" || applicationVCSServer == "" || vcsServer == nil {
		return &vcsInfos, nil
	}

	// START OBSERVABILITY
	ctx, end := observability.Span(ctx, "workflow.getVCSInfos",
		observability.Tag("application", applicationName),
		observability.Tag("vcs_server", applicationVCSServer),
		observability.Tag("vcs_repo", applicationRepositoryFullname),
	)
	defer end()

	// Try to get the data from cache
	cacheKey := cache.Key("api:workflow:getVCSInfos:", applicationVCSServer, applicationRepositoryFullname, vcsInfos.String())
	if store.Get(cacheKey, &vcsInfos) {
		log.Debug("completeVCSInfos> load from cache: %s", cacheKey)
		return &vcsInfos, nil
	}

	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, projectKey, vcsServer)
	if errclient != nil {
		return nil, sdk.WrapError(errclient, "cannot get client")
	}

	// Check repository value
	if vcsInfos.Repository == "" {
		vcsInfos.Repository = applicationRepositoryFullname
	} else if strings.ToLower(vcsInfos.Repository) != strings.ToLower(applicationRepositoryFullname) {
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
			return nil, sdk.NewError(sdk.ErrNotFound, fmt.Errorf("repository %s not found", vcsInfos.Repository))
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
		branch, errB := client.Branch(ctx, vcsInfos.Repository, vcsInfos.Branch)
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
			log.Warning("unable to ")
			return nil, sdk.WrapError(errCm, "cannot get commit infos for %s %s", vcsInfos.Repository, vcsInfos.Hash)
		}
		vcsInfos.Author = commit.Author.Name
		vcsInfos.Message = commit.Message
	}

	store.Set(cacheKey, vcsInfos)
	return &vcsInfos, nil
}
