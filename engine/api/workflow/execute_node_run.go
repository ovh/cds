package workflow

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func syncTakeJobInNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun, j *sdk.WorkflowNodeJobRun, stageIndex int, chanEvent chan<- interface{}) (errExecute error) {
	t0 := time.Now()
	log.Debug("workflow.syncTakeJobInNodeRun> Begin [#%d.%d] runID=%d (%s)", n.Number, n.SubNumber, n.WorkflowRunID, n.Status)
	defer func() {
		log.Debug("workflow.syncTakeJobInNodeRun> End [#%d.%d] runID=%d (%s) - %.3fs", n.Number, n.SubNumber, n.WorkflowRunID, n.Status, time.Since(t0).Seconds())
		if errExecute != nil {
			log.Error("workflow.syncTakeJobInNodeRun.defer> Unable to execute run %d: %v", n.WorkflowRunID, errExecute)
		}
	}()

	//If status is not waiting neither build: nothing to do
	if sdk.StatusIsTerminated(n.Status) {
		return nil
	}

	//Browse stages
	stage := &n.Stages[stageIndex]
	if stage.Status == sdk.StatusWaiting {
		stage.Status = sdk.StatusBuilding
	}
	isStopped := true
	for i := range stage.RunJobs {
		rj := &stage.RunJobs[i]
		if rj.ID == j.ID {
			rj.Status = j.Status
			rj.Start = j.Start
			rj.Done = j.Done
			rj.Model = j.Model
			rj.Job = j.Job
		}
		if rj.Status != sdk.StatusStopped.String() {
			isStopped = false
		}
	}
	if isStopped {
		stage.Status = sdk.StatusStopped
	}

	if n.Status == sdk.StatusWaiting.String() {
		n.Status = sdk.StatusBuilding.String()
		if chanEvent != nil {
			chanEvent <- *n
		}
	}

	// Save the node run in database
	if err := UpdateNodeRun(db, n); err != nil {
		return sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s. err:%s", n.ID, n.Status, err), "workflow.syncTakeJobInNodeRun> Unable to execute node")
	}
	return nil
}

func execute(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, n *sdk.WorkflowNodeRun, chanEvent chan<- interface{}) (errExecute error) {
	t0 := time.Now()
	log.Debug("workflow.execute> Begin [#%d.%d] runID=%d (%s)", n.Number, n.SubNumber, n.WorkflowRunID, n.Status)
	defer func() {
		log.Debug("workflow.execute> End [#%d.%d] runID=%d (%s) - %.3fs", n.Number, n.SubNumber, n.WorkflowRunID, n.Status, time.Since(t0).Seconds())
		if errExecute != nil {
			log.Error("workflow.execute.defer> Unable to execute run %d: %v", n.WorkflowRunID, errExecute)
		}
	}()

	//If status is not waiting neither build: nothing to do
	if sdk.StatusIsTerminated(n.Status) {
		return nil
	}

	var newStatus = n.Status

	//If no stages ==> success
	if len(n.Stages) == 0 {
		newStatus = sdk.StatusSuccess.String()
	}

	stagesTerminated := 0
	//Browse stages
	for stageIndex := range n.Stages {
		stage := &n.Stages[stageIndex]
		log.Debug("workflow.execute> checking stage %s (status=%s)", stage.Name, stage.Status)
		//Initialize stage status at waiting
		if stage.Status.String() == "" {
			stage.Status = sdk.StatusWaiting

			if stageIndex == 0 {
				newStatus = sdk.StatusWaiting.String()
			}

			if len(stage.Jobs) == 0 {
				newStatus = sdk.StatusSuccess.String()
				stage.Status = sdk.StatusSuccess
			} else {
				//Add job to Queue
				//Insert data in workflow_node_run_job
				log.Debug("workflow.execute> stage %s call addJobsToQueue", stage.Name)
				if err := addJobsToQueue(db, stage, n, chanEvent); err != nil {
					return err
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
			end, errSync := syncStage(db, store, stage)
			if errSync != nil {
				return errSync
			}
			if !end {
				break
			} else {
				//The stage is over
				if stage.Status == sdk.StatusFail {
					n.Done = time.Now()
					newStatus = sdk.StatusFail.String()
					stagesTerminated++
					break
				}
				if stage.Status == sdk.StatusStopped {
					n.Done = time.Now()
					newStatus = sdk.StatusStopped.String()
					stagesTerminated++
					break
				}

				if sdk.StatusIsTerminated(stage.Status.String()) {
					n.Done = time.Now()
				}

				if stageIndex == len(n.Stages)-1 {
					n.Done = time.Now()
					newStatus = sdk.StatusSuccess.String()
					stagesTerminated++
					break
				}
				if stageIndex != len(n.Stages)-1 {
					continue
				}
			}
		}
	}

	if stagesTerminated >= len(n.Stages) || (stagesTerminated >= len(n.Stages)-1 && (n.Stages[len(n.Stages)-1].Status == sdk.StatusDisabled || n.Stages[len(n.Stages)-1].Status == sdk.StatusSkipped)) {
		var success, building, fail, stop, skipped, disabled int
		if len(n.Stages) > 0 {
			for _, stage := range n.Stages {
				computeRunStatus(stage.Status.String(), &success, &building, &fail, &stop, &skipped, &disabled)
			}
			newStatus = getRunStatus(success, building, fail, stop, skipped, disabled)
		}
	}

	log.Debug("workflow.execute> status from %s to %s", n.Status, newStatus)
	n.Status = newStatus

	// Save the node run in database
	if err := UpdateNodeRun(db, n); err != nil {
		return sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s. err:%s", n.ID, n.Status, err), "workflow.execute> Unable to execute node")
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(db, n.WorkflowRunID, false)
	if err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to reload workflow run id=%d", n.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	//Delete jobs only when node is over
	if sdk.StatusIsTerminated(n.Status) {
		// push node run event
		if chanEvent != nil {
			chanEvent <- *n
		}

		if n.Status != sdk.StatusStopped.String() {
			if _, err := processWorkflowRun(dbCopy, db, store, p, updatedWorkflowRun, nil, nil, nil, chanEvent); err != nil {
				return sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
			}
		}

		//Delete the line in workflow_node_run_job
		if err := DeleteNodeJobRuns(db, n.ID); err != nil {
			return sdk.WrapError(err, "workflow.execute> Unable to delete node %d job runs ", n.ID)
		}

		node := updatedWorkflowRun.Workflow.GetNode(n.WorkflowNodeID)
		//Do we release a mutex ?
		//Try to find one node run of the same node from the same workflow at status Waiting
		if node != nil && node.Context != nil && node.Context.Mutex {
			mutexQuery := `select workflow_node_run.id
			from workflow_node_run
			join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
			join workflow on workflow.id = workflow_run.workflow_id
			where workflow.id = $1
			and workflow_node_run.workflow_node_name = $2
			and workflow_node_run.status = $3
			order by workflow_node_run.start asc
			limit 1`
			waitingRunID, errID := db.SelectInt(mutexQuery, updatedWorkflowRun.WorkflowID, node.Name, string(sdk.StatusWaiting))
			if errID != nil && errID != sql.ErrNoRows {
				log.Error("workflow.execute> Unable to load mutex-locked workflow node run ID: %v", errID)
				return nil
			}
			//If not more run is found, stop the loop
			if waitingRunID == 0 {
				return nil
			}
			waitingRun, errRun := LoadNodeRunByID(db, waitingRunID, false)
			if errRun != nil && errRun != sql.ErrNoRows {
				log.Error("workflow.execute> Unable to load mutex-locked workflow rnode un: %v", errRun)
				return nil
			}
			//If not more run is found, stop the loop
			if waitingRun == nil {
				return nil
			}

			workflowRun, errWRun := LoadRunByID(db, waitingRun.WorkflowRunID, false)
			if errWRun != nil {
				log.Error("workflow.execute> Unable to load mutex-locked workflow rnode un: %v", errWRun)
				return nil
			}
			AddWorkflowRunInfo(workflowRun, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutexRelease.ID,
				Args: []interface{}{waitingRun.WorkflowNodeName},
			})

			if err := UpdateWorkflowRun(db, workflowRun); err != nil {
				return sdk.WrapError(err, "workflow.execute> Unable to update workflow run %d after mutex release", workflowRun.ID)
			}

			log.Debug("workflow.execute> process the node run %d because mutex has been released", waitingRun.ID)
			//TODO: how to manage the chanEvent ? Need to discuss about it
			if err := execute(dbCopy, db, store, p, waitingRun, nil); err != nil {
				return sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow")
			}
		}
	}
	return nil
}

func addJobsToQueue(db gorp.SqlExecutor, stage *sdk.Stage, run *sdk.WorkflowNodeRun, chanEvent chan<- interface{}) error {
	log.Debug("addJobsToQueue> add %d in stage %s", run.ID, stage.Name)
	conditionsOK, err := sdk.WorkflowCheckConditions(stage.Conditions(), run.BuildParameters)
	if err != nil {
		return sdk.WrapError(err, "addJobsToQueue> Cannot compute prerequisites on stage %s(%d)", stage.Name, stage.ID)
	}

	if !conditionsOK {
		stage.Status = sdk.StatusSkipped
	}
	if !stage.Enabled {
		stage.Status = sdk.StatusDisabled
	}

	skippedOrDisabledJobs := 0
	//Browse the jobs
	for j := range stage.Jobs {
		job := &stage.Jobs[j]
		errs := sdk.MultiError{}
		//Process variables for the jobs
		jobParams, errParam := getNodeJobRunParameters(db, *job, run, stage)
		if errParam != nil {
			errs.Join(*errParam)
		}
		jobRequirements, errReq := getNodeJobRunRequirements(db, *job, run)
		if errReq != nil {
			errs.Join(*errReq)
		}
		job.Action.Requirements = jobRequirements

		// add requirements in job parameters, to use them as {{.job.requirement...}} in job
		jobParams = append(jobParams, prepareRequirementsToNodeJobRunParameters(jobRequirements)...)

		groups, errGroups := getJobExecutablesGroups(db, run)
		if errGroups != nil {
			return sdk.WrapError(errGroups, "addJobsToQueue> error on getJobExecutablesGroups")
		}

		//Create the job run
		wjob := sdk.WorkflowNodeJobRun{
			WorkflowNodeRunID: run.ID,
			Start:             time.Time{},
			Queued:            time.Now(),
			Status:            sdk.StatusWaiting.String(),
			Parameters:        jobParams,
			ExecGroups:        groups,
			Job: sdk.ExecutedJob{
				Job: *job,
			},
		}

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
		}

		//Insert in database
		if err := insertWorkflowNodeJobRun(db, &wjob); err != nil {
			return sdk.WrapError(err, "addJobsToQueue> Unable to insert in table workflow_node_run_job")
		}

		if chanEvent != nil {
			chanEvent <- wjob
		}

		//Put the job run in database
		stage.RunJobs = append(stage.RunJobs, wjob)
	}

	if skippedOrDisabledJobs == len(stage.Jobs) {
		stage.Status = sdk.StatusSkipped
	}

	return nil
}

func getJobExecutablesGroups(db gorp.SqlExecutor, run *sdk.WorkflowNodeRun) ([]sdk.Group, error) {
	query := `
	SELECT distinct("group".id), "group".name FROM "group"
	LEFT JOIN project_group ON project_group.group_id = "group".id
	LEFT JOIN pipeline_group ON pipeline_group.group_id = "group".id
	LEFT JOIN workflow_group ON workflow_group.group_id = "group".id
	LEFT JOIN workflow_node ON workflow_node.pipeline_id = pipeline_group.pipeline_id
	LEFT JOIN workflow_node_context ON workflow_node_context.workflow_node_id = workflow_node.id
	LEFT OUTER JOIN application_group ON workflow_node_context.application_id = application_group.application_id
	LEFT OUTER JOIN environment_group ON workflow_node_context.environment_id = environment_group.environment_id
	WHERE workflow_node.id = $1
		AND workflow_node_context.workflow_node_id = workflow_node.id
		AND workflow_node.pipeline_id = pipeline_group.pipeline_id
		AND pipeline_group.group_id = "group".id
		AND workflow_group.workflow_id = workflow_node.workflow_id
		AND workflow_group.role >= $2
		AND (workflow_node_context.application_id is null OR application_group.role >= $2)
		AND pipeline_group.role >= $2
		AND (workflow_node_context.environment_id is NULL or environment_group.role >= $2);
	`

	var groups []sdk.Group
	rows, err := db.Query(query, run.WorkflowNodeID, permission.PermissionReadExecute)
	if err != nil {
		return nil, sdk.WrapError(err, "getJobExecutablesGroups> err query")
	}
	defer rows.Close()

	var sharedInfraIn bool
	for rows.Next() {
		var g sdk.Group
		var groupID sql.NullInt64
		var groupName sql.NullString

		if err := rows.Scan(&groupID, &groupName); err != nil {
			return nil, sdk.WrapError(err, "getJobExecutablesGroups> err scan")
		}

		if groupID.Valid {
			g.ID = groupID.Int64
			g.Name = groupName.String
		}
		groups = append(groups, g)
		if g.ID == group.SharedInfraGroup.ID {
			sharedInfraIn = true
		}
	}
	if !sharedInfraIn {
		groups = append(groups, *group.SharedInfraGroup)
	}

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
			spawnInfos, err := loadNodeRunJobInfo(db, runJob.ID)
			if err != nil {
				return false, sdk.WrapError(err, "syncStage> unable to load spawn infos for runJob: %d", runJob.ID)
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
func NodeBuildParametersFromWorkflow(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, refNode *sdk.WorkflowNode, ancestorsIds []int64) ([]sdk.Parameter, error) {

	res := []sdk.Parameter{}
	if len(res) == 0 {
		var err error
		res, err = GetNodeBuildParameters(db, store, proj, wf, refNode, refNode.Context.DefaultPipelineParameters, refNode.Context.DefaultPayload)
		if err != nil {
			return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to get workflow node parameters: %v", err)
		}
	}

	// Process ancestor
	for _, aID := range ancestorsIds {
		ancestor := wf.GetNode(aID)
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
			mergedParameters := sdk.ParametersMapMerge(tempParams, m1)
			res = sdk.ParametersFromMap(mergedParameters)
		}
	}

	return res, nil
}

// StopWorkflowNodeRun to stop a workflow node run with a specific spawn info
func StopWorkflowNodeRun(db *gorp.DbMap, store cache.Store, proj *sdk.Project, nodeRun sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo, chanEvent chan<- interface{}) error {
	const stopWorkflowNodeRunNBWorker = 5
	var wg sync.WaitGroup
	// Load node job run ID
	ids, errIDS := LoadNodeJobRunIDByNodeRunID(db, nodeRun.ID)
	if errIDS != nil {
		return sdk.WrapError(errIDS, "StopWorkflowNodeRun> Cannot load node jobs run ids ")
	}

	chanNjrID := make(chan int64, stopWorkflowNodeRunNBWorker)
	chanNodeJobRunDone := make(chan bool, stopWorkflowNodeRunNBWorker)
	chanErr := make(chan error, stopWorkflowNodeRunNBWorker)
	for i := 0; i < stopWorkflowNodeRunNBWorker && i < len(ids); i++ {
		go stopWorkflowNodeJobRun(db, store, proj, &nodeRun, stopInfos, chanNjrID, chanEvent, chanErr, chanNodeJobRunDone, &wg)
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
			return err
		}
	}
	wg.Wait()

	// Update stages from node run
	for iS := range nodeRun.Stages {
		stag := &nodeRun.Stages[iS]
		for iR := range stag.RunJobs {
			runj := &stag.RunJobs[iR]
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

	nodeRun.Status = sdk.StatusStopped.String()
	nodeRun.Done = time.Now()
	if errU := UpdateNodeRun(db, &nodeRun); errU != nil {
		return sdk.WrapError(errU, "StopWorkflowNodeRun> Cannot update node run")
	}

	return nil
}

func stopWorkflowNodeJobRun(db *gorp.DbMap, store cache.Store, proj *sdk.Project, nodeRun *sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo, chanNjrID <-chan int64, chanNodeJobRun chan<- interface{}, chanErr chan<- error, chanDone chan<- bool, wg *sync.WaitGroup) {
	for njrID := range chanNjrID {
		tx, errTx := db.Begin()
		if errTx != nil {
			chanErr <- sdk.WrapError(errTx, "StopWorkflowNodeRun> Cannot create transaction")
			wg.Done()
			return
		}

		njr, errNRJ := LoadAndLockNodeJobRunWait(tx, store, njrID)
		if errNRJ != nil {
			chanErr <- sdk.WrapError(errNRJ, "StopWorkflowNodeRun> Cannot load node job run id")
			tx.Rollback()
			wg.Done()
			return
		}

		if err := AddSpawnInfosNodeJobRun(tx, store, proj, njr.ID, []sdk.SpawnInfo{stopInfos}); err != nil {
			chanErr <- sdk.WrapError(err, "StopWorkflowNodeRun> Cannot save spawn info job %d", njr.ID)
			tx.Rollback()
			wg.Done()
			return
		}

		njr.SpawnInfos = append(njr.SpawnInfos, stopInfos)
		if err := UpdateNodeJobRunStatus(db, tx, store, proj, njr, sdk.StatusStopped, chanNodeJobRun); err != nil {
			chanErr <- sdk.WrapError(err, "StopWorkflowNodeRun> Cannot update node job run")
			tx.Rollback()
			wg.Done()
			return
		}

		if err := tx.Commit(); err != nil {
			chanErr <- sdk.WrapError(err, "StopWorkflowNodeRun> Cannot commit transaction")
			tx.Rollback()
			wg.Done()
			return
		}
		chanDone <- true
		wg.Done()
	}
}

// SyncNodeRunRunJob sync step status and spawnInfos in a specific run job
func SyncNodeRunRunJob(db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun, nodeJobRun sdk.WorkflowNodeJobRun) (bool, error) {
	found := false
	for i := range nodeRun.Stages {
		s := &nodeRun.Stages[i]
		for j := range s.RunJobs {
			runJob := &s.RunJobs[j]
			if runJob.ID == nodeJobRun.ID {
				spawnInfos, err := loadNodeRunJobInfo(db, runJob.ID)
				if err != nil {
					return false, sdk.WrapError(err, "SyncNodeRunRunJob> unable to load spawn infos for runJobID: %d", runJob.ID)
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
	repository string
	branch     string
	hash       string
	author     string
	message    string
	url        string
	httpurl    string
}

func getVCSInfos(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, gitValues map[string]string, node *sdk.WorkflowNode, nodeRun *sdk.WorkflowNodeRun, ignoreError bool) (vcsInfos, error) {
	var vcsInfos vcsInfos
	vcsInfos.repository = gitValues[tagGitRepository]
	vcsInfos.branch = gitValues[tagGitBranch]
	vcsInfos.hash = gitValues[tagGitHash]
	vcsInfos.author = gitValues[tagGitAuthor]
	vcsInfos.message = gitValues[tagGitMessage]

	if node.Context == nil || node.Context.Application == nil || node.Context.Application.VCSServer == "" {
		return vcsInfos, nil
	}

	vcsServer := repositoriesmanager.GetProjectVCSServer(p, node.Context.Application.VCSServer)
	if vcsServer == nil {
		return vcsInfos, nil
	}

	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(db, store, vcsServer)
	if errclient != nil {
		return vcsInfos, sdk.WrapError(errclient, "computeVCSInfos> Cannot get client")
	}

	// Set default values
	if vcsInfos.repository == "" {
		vcsInfos.repository = node.Context.Application.RepositoryFullname
	} else if vcsInfos.repository != node.Context.Application.RepositoryFullname {
		//The input repository is not the same as the application, we have to check if it is a fork
		forks, err := client.ListForks(node.Context.Application.RepositoryFullname)
		if err != nil {
			return vcsInfos, sdk.WrapError(err, "computeVCSInfos> Cannot get forks for %s", node.Context.Application.RepositoryFullname)
		}
		var forkFound bool
		for _, fork := range forks {
			if vcsInfos.repository == fork.Fullname {
				forkFound = true
				break
			}
		}

		//If it's not a fork; reset this value to the application repository
		if !forkFound {
			if !ignoreError {
				return vcsInfos, sdk.NewError(sdk.ErrNotFound, fmt.Errorf("repository %s not found", vcsInfos.repository))
			}
			vcsInfos.repository = node.Context.Application.RepositoryFullname
		}
	}

	//Get the url and http_url
	repo, err := client.RepoByFullname(vcsInfos.repository)
	if err != nil {
		if !ignoreError {
			return vcsInfos, sdk.NewError(sdk.ErrNotFound, err)
		}
		//If we ignore errors
		vcsInfos.repository = node.Context.Application.RepositoryFullname
		repo, err = client.RepoByFullname(node.Context.Application.RepositoryFullname)
		if err != nil {
			return vcsInfos, sdk.WrapError(err, "computeVCSInfos> Cannot get repo %s", node.Context.Application.RepositoryFullname)
		}
	}
	vcsInfos.url = repo.SSHCloneURL
	vcsInfos.httpurl = repo.HTTPCloneURL

	//Check that the branch exist, if it isn't existing, let the default branch
	var branch *sdk.VCSBranch
	if vcsInfos.branch != "" {
		branch, err = client.Branch(vcsInfos.repository, vcsInfos.branch)
		if err != nil {
			if !ignoreError {
				return vcsInfos, sdk.NewError(sdk.ErrNotFound, err)
			}
		}

		if branch == nil {
			log.Error("computeVCSInfos> unable to get branch %s", vcsInfos.branch)
			vcsInfos.branch = ""
		}
	}

	//Get the default branch
	if branch == nil {
		branches, errR := client.Branches(vcsInfos.repository)
		if errR != nil {
			return vcsInfos, sdk.WrapError(errR, "computeVCSInfos> cannot get branches infos for %s", vcsInfos.repository)
		}
		_branch := sdk.GetDefaultBranch(branches)
		branch = &_branch
		vcsInfos.branch = branch.DisplayID
	}
	if vcsInfos.hash == "" {
		vcsInfos.hash = branch.LatestCommit
	}

	//Get the latest commit
	commit, errCm := client.Commit(vcsInfos.repository, vcsInfos.hash)
	if errCm != nil {
		if !ignoreError {
			return vcsInfos, sdk.WrapError(errCm, "computeVCSInfos> cannot get commit infos for %s %s", vcsInfos.repository, vcsInfos.hash)
		}
		vcsInfos.hash = branch.LatestCommit
		commit, errCm = client.Commit(vcsInfos.repository, vcsInfos.hash)
		if errCm != nil {
			return vcsInfos, sdk.WrapError(errCm, "computeVCSInfos> cannot get commit infos for %s %s", vcsInfos.repository, vcsInfos.hash)
		}
	}
	vcsInfos.author = commit.Author.Name
	vcsInfos.message = commit.Message

	return vcsInfos, nil
}
