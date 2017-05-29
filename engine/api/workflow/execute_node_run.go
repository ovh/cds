package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func execute(db *gorp.DbMap, n *sdk.WorkflowNodeRun) error {
	t0 := time.Now()
	log.Debug("workflow.execute> Begin [#%d.%d] runID=%d", n.Number, n.SubNumber, n.WorkflowRunID)
	defer func() {
		log.Debug("workflow.execute> End [#%d.%d] runID=%d - %.3fs", n.Number, n.SubNumber, n.WorkflowRunID, time.Since(t0).Seconds())
	}()

	//Start a transaction
	tx, errtx := db.Begin()
	if errtx != nil {
		return errtx
	}
	defer tx.Rollback()

	//Select for update on table workflow_run, workflow_node_run
	if _, err := tx.Exec("select workflow_run.* from workflow_run where id = $1 for update nowait", n.WorkflowRunID); err != nil {
		log.Debug("workflow.execute> Unable to take lock on workflow_run ID=%d (%v)", n.WorkflowRunID, err)
		return nil
	}
	if _, err := tx.Exec("select workflow_node_run.* from workflow_node_run where id = $1 for update nowait", n.ID); err != nil {
		log.Debug("workflow.execute> Unable to take lock on workflow_node_run ID=%d (%v)", n.ID, err)
		return nil
	}

	//Reload the workflow run ID to get the node, and the pipeline...
	workflowRun, errw := loadRunByID(tx, n.WorkflowRunID)
	if errw != nil {
		return sdk.WrapError(errw, "workflow.execute> Unable to load run id=%id", n.WorkflowRunID)
	}

	//If status is not waiting neither build: nothing to do
	if n.Status != sdk.StatusWaiting.String() && n.Status != sdk.StatusBuilding.String() {
		return nil
	}

	//New status is building
	n.Status = sdk.StatusBuilding.String()

	//If no stages ==> success
	if len(n.Stages) == 0 {
		n.Status = sdk.StatusSuccess.String()
	}

	//Browse stages
	for stageIndex := range n.Stages {
		stage := &n.Stages[stageIndex]
		log.Debug("workflow.execute> analyzing stage %s: %#v", stage.Name, stage)

		if stage.Status == sdk.StatusWaiting {
			//Add job to Queue
			//Insert data in workflow_node_run_job
			if err := addJobsToQueue(tx, stage, n); err != nil {
				return err
			}
		}

		if stage.Status == sdk.StatusBuilding {
			var end bool
			end, errSync := syncStage(tx, stage)
			if errSync != nil {
				return errSync
			}
			if end {
				//The job is over

				//Delete the line in workflow_node_run_job

				//If the stage is failed: Ends the node as failed

				//If the stage is stage is the last: Ends the node as success

				//Else (this is not the last stage): Set the status of the next stage at waiting
			}
		}
	}

	// Save the node run in database
	if err := updateWorkflowNodeRun(tx, n); err != nil {
		return sdk.WrapError(fmt.Errorf("Unable to update node id=%d", n.ID), "workflow.execute> Unable to execute node")
	}

	//Reload the workflow
	updatedWorkflowRun, err := loadRunByID(tx, workflowRun.ID)
	if err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to reload workflow run id=%d", workflowRun.ID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	if n.Status == sdk.StatusSuccess.String() {
		if err := processWorkflowRun(tx, updatedWorkflowRun, nil, nil, nil); err != nil {
			sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
		}
	}

	//Commit all the things
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to commit tx")
	}

	return nil
}

func addJobsToQueue(db gorp.SqlExecutor, stage *sdk.Stage, run *sdk.WorkflowNodeRun) error {
	log.Debug("addJobsToQueue> add %#v in stage %s", run, stage.Name)
	//Check stage prerequisites
	var prerequisitesOK = true
	/*
		prerequisitesOK, err := pipeline.CheckPrerequisites(*stage, pb)
		if err != nil {
			log.Warning("addJobsToQueue> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d): %s\n", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID, err)
			return err
		}
	*/

	//Update the stage status
	stage.Status = sdk.StatusBuilding

	//Browse the jobs
	for _, job := range stage.Jobs {
		//Process variables for the jobs
		jobParams, errParam := getNodeJobRunVariables(db, job, run, stage)
		if errParam != nil {
			return errParam
		}

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
		} else if !prerequisitesOK {
			job.Status = sdk.StatusSkipped.String()
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

func getNodeJobRunVariables(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := loadRunByID(db, run.WorkflowRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeJobRunVariables> Unable to load workflow run")
	}

	// Load project variables
	pv, err := project.GetAllVariableInProject(db, w.Workflow.ProjectID)
	if err != nil {
		return nil, err
	}

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeJobRunVariables>")
	}

	// compute application variables
	av := []sdk.Variable{}
	if n.Context != nil && n.Context.Application != nil {
		av = n.Context.Application.Variable
	}

	// compute environment variables
	ev := []sdk.Variable{}
	if n.Context != nil && n.Context.Environment != nil {
		ev = n.Context.Environment.Variable
	}

	return action.ProcessActionBuildVariables(pv, av, ev, run.PipelineParameter, run.Payload, stage, j.Action), nil
}
