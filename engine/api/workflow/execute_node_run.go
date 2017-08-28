package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func lockAndExecute(db *gorp.DbMap, n *sdk.WorkflowNodeRun) error {
	//Start a transaction
	tx, errtx := db.Begin()
	if errtx != nil {
		return errtx
	}
	defer tx.Rollback()

	//Select fobor update on table workflow_run, workflow_node_run
	if _, err := tx.Exec("select workflow_run.* from workflow_run where id = $1 for update nowait", n.WorkflowRunID); err != nil {
		return fmt.Errorf("Unable to take lock on workflow_run ID=%d (%v)", n.WorkflowRunID, err)
	}
	if _, err := tx.Exec("select workflow_node_run.* from workflow_node_run where id = $1 for update nowait", n.ID); err != nil {
		return fmt.Errorf("Unable to take lock on workflow_run ID=%d (%v)", n.WorkflowRunID, err)
	}

	if err := execute(db, n); err != nil {
		return err
	}

	//Commit all the things
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to commit tx")
	}

	return nil
}

//execute is called by the scheduler. You should not call this by yourself
func execute(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) (err error) {
	t0 := time.Now()
	log.Debug("workflow.execute> Begin [#%d.%d] runID=%d (%s)", n.Number, n.SubNumber, n.WorkflowRunID, n.Status)
	defer func() {
		log.Debug("workflow.execute> End [#%d.%d] runID=%d (%s) - %.3fs", n.Number, n.SubNumber, n.WorkflowRunID, n.Status, time.Since(t0).Seconds())
		if err != nil {
			log.Error("workflow.execute> Unable to execute run %d: %v", n.WorkflowRunID, err)
			run, errw := loadAndLockRunByID(db, n.WorkflowRunID)
			if errw != nil {
				log.Error("workflow.execute> Unable to add infos on run %d: %v", n.WorkflowRunID, errw)
				return
			}
			AddWorkflowRunInfo(run, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{err},
			})
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
			if err := addJobsToQueue(db, stage, n); err != nil {
				return err
			}
			if stage.Status == sdk.StatusSkipped || stage.Status == sdk.StatusDisabled {
				continue
			}
			break
		}

		//If stage is waiting, nothing to do
		if stage.Status == sdk.StatusWaiting {
			break
		}

		if stage.Status == sdk.StatusBuilding {
			newStatus = sdk.StatusBuilding.String()

			var end bool
			end, errSync := syncStage(db, stage)
			if errSync != nil {
				return errSync
			}
			if end {
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

	n.Status = newStatus
	// Save the node run in database
	if err := UpdateNodeRun(db, n); err != nil {
		return sdk.WrapError(fmt.Errorf("Unable to update node id=%d at status %s", n.ID, n.Status), "workflow.execute> Unable to execute node")
	}

	//Reload the workflow
	updatedWorkflowRun, err := LoadRunByID(db, n.WorkflowRunID)
	if err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to reload workflow run id=%d", n.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	if n.Status == sdk.StatusSuccess.String() {
		if err := processWorkflowRun(db, updatedWorkflowRun, nil, nil, nil); err != nil {
			return sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
		}
	}

	//Delete jobs only when node is over
	if n.Status == sdk.StatusSuccess.String() || n.Status == sdk.StatusFail.String() {
		//Delete the line in workflow_node_run_job
		if err := DeleteNodeJobRuns(db, n.ID); err != nil {
			return sdk.WrapError(err, "workflow.execute> Unable to delete node %d job runs ", n.ID)
		}
	}

	return nil
}

func addJobsToQueue(db gorp.SqlExecutor, stage *sdk.Stage, run *sdk.WorkflowNodeRun) error {
	log.Debug("addJobsToQueue> add %d in stage %s", run.ID, stage.Name)

	conditionsOK, err := sdk.WorkflowCheckConditions(stage.Conditions(), run.BuildParameters)
	if err != nil {
		log.Warning("addJobsToQueue> Cannot compute prerequisites on stage %s(%d): err", stage.Name, stage.ID, err)
		return err
	}

	if !conditionsOK {
		stage.Status = sdk.StatusSkipped
	}
	if !stage.Enabled {
		stage.Status = sdk.StatusDisabled
	}

	//Browse the jobs
	for _, job := range stage.Jobs {
		//Process variables for the jobs
		jobParams, errParam := getNodeJobRunParameters(db, job, run, stage)

		//Create the job run
		job := sdk.WorkflowNodeJobRun{
			WorkflowNodeRunID: run.ID,
			Start:             time.Time{},
			Queued:            time.Now(),
			Status:            sdk.StatusWaiting.String(),
			Parameters:        jobParams,
			Job: sdk.ExecutedJob{
				Job: job,
			},
		}

		if !stage.Enabled || !job.Job.Enabled {
			job.Status = sdk.StatusDisabled.String()
		} else if !conditionsOK {
			job.Status = sdk.StatusSkipped.String()
		}

		if errParam != nil {
			job.Status = sdk.StatusFail.String()

			errm, ok := errParam.(*sdk.MultiError)
			spawnInfos := sdk.SpawnMsg{
				ID: sdk.MsgSpawnInfoJobError.ID,
			}

			if ok {
				for _, e := range *errm {
					spawnInfos.Args = append(spawnInfos.Args, e.Error())
				}
			} else {
				spawnInfos.Args = []interface{}{errParam.Error()}
			}

			job.SpawnInfos = []sdk.SpawnInfo{sdk.SpawnInfo{
				APITime:    time.Now(),
				Message:    spawnInfos,
				RemoteTime: time.Now(),
			}}

		}

		//Insert in database
		if err := insertWorkflowNodeJobRun(db, &job); err != nil {
			return sdk.WrapError(err, "addJobsToQueue> Unable to insert in table workflow_node_run_job")
		}

		//Put the job run in database
		event.PublishJobRun(run, &job)
		stage.RunJobs = append(stage.RunJobs, job)
	}

	return nil
}

func syncStage(db gorp.SqlExecutor, stage *sdk.Stage) (bool, error) {
	stageEnd := true
	finalStatus := sdk.StatusBuilding

	// browse all running jobs
	for indexJob := range stage.RunJobs {
		pbJob := &stage.RunJobs[indexJob]
		// If job is runnning, sync it
		if pbJob.Status == sdk.StatusBuilding.String() || pbJob.Status == sdk.StatusWaiting.String() {
			pbJobDB, errJob := LoadNodeJobRun(db, pbJob.ID)
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
	if stageEnd || len(stage.RunJobs) == 0 {
		if len(stage.PipelineBuildJobs) == 0 {
			finalStatus = sdk.StatusSuccess
			stageEnd = true
		}
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
