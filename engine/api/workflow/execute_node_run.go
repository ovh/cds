package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
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
func execute(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	t0 := time.Now()
	log.Debug("workflow.execute> Begin [#%d.%d] runID=%d (%s)", n.Number, n.SubNumber, n.WorkflowRunID, n.Status)
	defer func() {
		log.Debug("workflow.execute> End [#%d.%d] runID=%d (%s) - %.3fs", n.Number, n.SubNumber, n.WorkflowRunID, n.Status, time.Since(t0).Seconds())
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
		log.Debug("workflow.execute> stage %d is %s", stageIndex, stage.Status.String())

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
			break
		}

		//If stage is waiting, nothing to do
		if stage.Status == sdk.StatusWaiting {
			break
		}

		if stage.Status == sdk.StatusBuilding {
			newStatus := sdk.StatusBuilding.String()

			var end bool
			end, errSync := syncStage(db, stage)
			if errSync != nil {
				return errSync
			}
			if end {
				log.Debug("workflow.execute> Stage [#%d.%d] ends runID=%d. Node is %s", n.Number, n.SubNumber, n.WorkflowRunID, newStatus)
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
	updatedWorkflowRun, err := loadRunByID(db, n.WorkflowRunID)
	if err != nil {
		return sdk.WrapError(err, "workflow.execute> Unable to reload workflow run id=%d", n.WorkflowRunID)
	}

	// If pipeline build succeed, reprocess the workflow (in the same transaction)
	if n.Status == sdk.StatusSuccess.String() {
		if err := processWorkflowRun(db, updatedWorkflowRun, nil, nil, nil); err != nil {
			sdk.WrapError(err, "workflow.execute> Unable to reprocess workflow !")
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
	//Check stage prerequisites
	var prerequisitesOK = true
	/*
		prerequisitesOK, err := pipeline.CheckPrerequisites(*stage, pb)
		if err != nil {
			log.Warning("addJobsToQueue> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d): %s\n", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID, err)
			return err
		}
	*/

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
		} else if !prerequisitesOK {
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

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := loadRunByID(db, run.WorkflowRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeJobRunVariables> Unable to load workflow run")
	}

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeJobRunVariables>")
	}
	vars := map[string]string{}

	//Load project
	proj, err := project.Load(db, w.Workflow.ProjectKey, nil, project.LoadOptions.WithVariables)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeJobRunVariables> Unable to load project")
	}
	tmp := sdk.ParametersFromProjectVariables(proj)
	for k, v := range tmp {
		vars[k] = v
	}

	// compute application variables
	if n.Context != nil && n.Context.Application != nil {
		vars["cds.application"] = n.Context.Application.Name
		tmp := sdk.ParametersFromApplicationVariables(n.Context.Application)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute environment variables
	if n.Context != nil && n.Context.Environment != nil {
		vars["cds.environment"] = n.Context.Environment.Name
		tmp := sdk.ParametersFromEnvironmentVariables(n.Context.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute pipeline parameters
	tmp = sdk.ParametersFromPipelineParameters(run.PipelineParameters)
	for k, v := range tmp {
		vars[k] = v
	}

	// compute payload
	tmp = sdk.ParametersToMap(run.Payload)

	tmp["cds.project"] = w.Workflow.ProjectKey
	tmp["cds.workflow"] = w.Workflow.Name
	tmp["cds.pipeline"] = n.Pipeline.Name
	tmp["cds.stage"] = stage.Name
	tmp["cds.job"] = j.Action.Name
	tmp["cds.version"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	errm := &sdk.MultiError{}

	params := []sdk.Parameter{}
	for k, v := range tmp {
		s, err := sdk.Interpolate(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}
