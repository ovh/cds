package workflow

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
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

func execute(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, n *sdk.WorkflowNodeRun, chanEvent chan<- interface{}) (errExecute error) {
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

	//Browse stages
	for stageIndex := range n.Stages {
		stage := &n.Stages[stageIndex]
		log.Debug("workflow.execute> checking stage %s (status=%s)", stage.Name, stage.Status)
		//Initialize stage status at waiting
		if stage.Status.String() == "" {
			if stageIndex == 0 {
				newStatus = sdk.StatusWaiting.String()
			}

			stage.Status = sdk.StatusWaiting
			//Add job to Queue
			//Insert data in workflow_node_run_job
			log.Debug("workflow.execute> stage %s call addJobsToQueue", stage.Name)
			if err := addJobsToQueue(db, stage, n, chanEvent); err != nil {
				return err
			}
			if stage.Status == sdk.StatusSkipped || stage.Status == sdk.StatusDisabled {
				continue
			}
			break
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
					break
				}
				if stage.Status == sdk.StatusStopped {
					n.Done = time.Now()
					newStatus = sdk.StatusStopped.String()
					break
				}
				if stageIndex == len(n.Stages)-1 {
					n.Done = time.Now()
					newStatus = sdk.StatusSuccess.String()
					break
				}
				if stageIndex != len(n.Stages)-1 {
					continue
				}
			}
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
			if _, err := processWorkflowRun(db, store, p, updatedWorkflowRun, nil, nil, nil, chanEvent); err != nil {
				return sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
			}
		}

		//Delete the line in workflow_node_run_job
		if err := DeleteNodeJobRuns(db, n.ID); err != nil {
			return sdk.WrapError(err, "workflow.execute> Unable to delete node %d job runs ", n.ID)
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

		//Create the job run
		wjob := sdk.WorkflowNodeJobRun{
			WorkflowNodeRunID: run.ID,
			Start:             time.Time{},
			Queued:            time.Now(),
			Status:            sdk.StatusWaiting.String(),
			Parameters:        jobParams,
			Job: sdk.ExecutedJob{
				Job: *job,
			},
		}

		if !stage.Enabled || !wjob.Job.Enabled {
			wjob.Status = sdk.StatusDisabled.String()
		} else if !conditionsOK {
			wjob.Status = sdk.StatusSkipped.String()
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

	return nil
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

//NodeBuildParameters returns build_parameters for a node given its id
func NodeBuildParameters(proj *sdk.Project, wf *sdk.Workflow, wr *sdk.WorkflowRun, id int64, u *sdk.User) ([]sdk.Parameter, error) {
	refNode := wf.GetNode(id)
	if refNode == nil {
		return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to load workflow node")
	}

	res := []sdk.Parameter{}

	if wr != nil {
		for nodeID, nodeRuns := range wr.WorkflowNodeRuns {
			oldNode := wr.Workflow.GetNode(nodeID)
			if oldNode == nil {
				log.Warning("getWorkflowTriggerConditionHandler> Unable to find last run")
				break
			}
			if oldNode.EqualsTo(refNode) {
				for _, p := range nodeRuns[0].BuildParameters {
					sdk.AddParameter(&res, p.Name, p.Type, p.Value)
				}
				break
			}
		}
	}

	if len(res) == 0 {
		var err error
		res, err = GetNodeBuildParameters(proj, wf, refNode, refNode.Context.DefaultPipelineParameters, refNode.Context.DefaultPayload)
		if err != nil {
			return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to get workflow node parameters: %v", err)
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
			runj.Status = sdk.StatusStopped.String()
			for iStep := range runj.Job.StepStatus {
				stepStat := &runj.Job.StepStatus[iStep]
				stepStat.Status = sdk.StatusStopped.String()
			}
		}
		stag.Status = sdk.StatusStopped
	}

	nodeRun.Status = sdk.StatusStopped.String()
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
		if err := UpdateNodeJobRunStatus(tx, store, proj, njr, sdk.StatusStopped, chanNodeJobRun); err != nil {
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
