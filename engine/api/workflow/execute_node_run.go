package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

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
	if n.Status != sdk.StatusWaiting.String() && n.Status != sdk.StatusBuilding.String() {
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
	//If the node is over, push the status in the build parameter, so it would be availabe in children build parameters
	if n.Status == sdk.StatusSuccess.String() || n.Status == sdk.StatusFail.String() {
		sdk.AddParameter(&n.BuildParameters, "cds.status", sdk.StringParameter, n.Status)
	}

	// Save the node run in database
	if err := UpdateNodeRun(db, n); err != nil {
		return sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s. err:%s", n.ID, n.Status, err), "workflow.execute> Unable to execute node")
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(db, n.WorkflowRunID)
	if err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to reload workflow run id=%d", n.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	//Delete jobs only when node is over
	if n.Status == sdk.StatusSuccess.String() || n.Status == sdk.StatusFail.String() {
		// push node run event
		if chanEvent != nil {
			chanEvent <- *n
		}
		if _, err := processWorkflowRun(db, store, p, updatedWorkflowRun, nil, nil, nil, chanEvent); err != nil {
			return sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
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

	log.Debug("syncStage> work on stage %s", stage.Name)
	// browse all running jobs
	for indexJob := range stage.RunJobs {
		pbJob := &stage.RunJobs[indexJob]
		// If job is runnning, sync it
		if pbJob.Status == sdk.StatusBuilding.String() || pbJob.Status == sdk.StatusWaiting.String() {
			pbJobDB, errJob := LoadNodeJobRun(db, store, pbJob.ID)
			if errJob != nil {
				return stageEnd, errJob
			}

			if pbJobDB.Status == sdk.StatusBuilding.String() || pbJobDB.Status == sdk.StatusWaiting.String() {
				stageEnd = false
			}

			pbJob.SpawnInfos = pbJobDB.SpawnInfos

			// If same status, sync step status
			if pbJobDB.Status == pbJob.Status {
				pbJob.Job.StepStatus = pbJobDB.Job.StepStatus
			} else {
				pbJob.Status = pbJobDB.Status
				pbJob.Start = pbJobDB.Start
				pbJob.Done = pbJobDB.Done
				pbJob.Model = pbJobDB.Model
				pbJob.Job = pbJobDB.Job
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
func StopWorkflowNodeRun(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, nodeRun sdk.WorkflowNodeRun, stopInfos sdk.SpawnInfo, chanEvent chan<- interface{}) error {
	// Load node job run ID
	ids, errIDS := LoadNodeJobRunIDByNodeRunID(db, nodeRun.ID)
	if errIDS != nil {
		return sdk.WrapError(errIDS, "stopWorkflowNodeRunHandler> Cannot load node job run id")
	}

	for _, nrjID := range ids {
		njr, errNRJ := LoadAndLockNodeJobRunWait(db, store, nrjID)
		if errNRJ != nil {
			return sdk.WrapError(errNRJ, "StopWorkflowNodeRun> Cannot load node job run id")
		}
		njr.SpawnInfos = append(njr.SpawnInfos, stopInfos)
		if err := UpdateNodeJobRunStatus(db, store, proj, njr, sdk.StatusStopped, chanEvent); err != nil {
			return sdk.WrapError(err, "StopWorkflowNodeRun> Cannot update node job run")
		}
	}

	if err := updateNodeRunStatus(db, nodeRun.ID, sdk.StatusStopped.String()); err != nil {
		return sdk.WrapError(err, "StopWorkflowNodeRun> Cannot update node run status")
	}

	return nil
}
