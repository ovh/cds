package queue

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Pipelines is a goroutine responsible for pushing actions of a building pipeline in queue, in the wanted order
func Pipelines() {
	// If this goroutine exits, then it's a crash
	defer log.Fatalf("Goroutine of scheduler.Schedule exited - Exit CDS Engine")

	for {
		time.Sleep(2 * time.Second)

		//Check if CDS is in maintenance mode
		var m bool
		cache.Get("maintenance", &m)
		if m {
			log.Warning("⚠ CDS maintenance in ON")
			time.Sleep(30 * time.Second)
		}

		db := database.DBMap(database.DB())
		if db != nil && !m {
			pipelines, err := pipeline.LoadBuildingPipelines(db)
			if err != nil {
				log.Warning("queue.Pipelines> Cannot load building pipelines: %s\n", err)
				// Add some extra sleep if db is down...
				time.Sleep(3 * time.Second)
				continue
			}

			for i := range pipelines {
				RunActions(db, pipelines[i])
			}
		}
	}
}

// RunActions Schedule action for the given Build
func RunActions(db *gorp.DbMap, pb sdk.PipelineBuild) {
	tx, err := db.Begin()
	if err != nil {
		log.Warning("queue.RunActions> cannot start tx for pb %d: %s\n", pb.ID, err)
		return
	}
	defer tx.Rollback()

	// Reload pipeline build with a FOR UPDATE NOT WAIT
	// So only one instance of the API can update it and/or end it
	err = pipeline.SelectBuildForUpdate(tx, pb.ID)
	if err != nil {
		// if ErrNoRows, pipelines is already done
		if err == sql.ErrNoRows {
			return
		}
		pqerr, ok := err.(*pq.Error)
		// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
		if ok && pqerr.Code == "55P03" {
			return
		}
		log.Warning("queue.RunActions> Cannot load pb: %s\n", err)
		return
	}

	pbNewStatus := sdk.StatusBuilding

	// OH! AN EMPTY PIPELINE
	if len(pb.Stages) == 0 {
		// Pipeline is done
		pbNewStatus = sdk.StatusSuccess
	}

	// Browse Stage
	for stageIndex := range pb.Stages {
		stage := &pb.Stages[stageIndex]

		if stage.Status == sdk.StatusWaiting {
			if err := addJobsToQueue(tx, stage, pb); err != nil {
				log.Warning("queue.RunActions> Cannot add job to queue: %s", err)
				return
			}
			break
		}

		if stage.Status == sdk.StatusBuilding {
			end, errSync := syncPipelineBuildJob(tx, stage)
			if errSync != nil {
				log.Warning("queue.RunActions> Cannot sync building jobs on stage %s(%d) of pipeline %s(%d): %s\n", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID, errSync)
				return
			}

			if end {
				// Remove pipeline build job
				if err := pipeline.DeletePipelineBuildJob(tx, pb.ID); err != nil {
					log.Warning("queue.RunActions> Cannot remove pipeline build jobs for pipeline build %d: %s\n", pb.ID, err)
					return
				}

				if stage.Status == sdk.StatusFail {
					pbNewStatus = sdk.StatusFail
					break
				}
				if stageIndex == len(pb.Stages)-1 {
					pbNewStatus = sdk.StatusSuccess
					break
				}
				if stageIndex != len(pb.Stages)-1 {
					// Prepare scheduling next stage
					pb.Stages[stageIndex+1].Status = sdk.StatusWaiting
					continue
				}
			}
		}
	}

	if err := pipeline.UpdatePipelineBuildStatusAndStage(tx, &pb, pbNewStatus); err != nil {
		log.Warning("RunActions> Cannot update UpdatePipelineBuildStatusAndStage on pb %d: %s\n", pb.ID, err)
	}

	// If pipeline build succeed, run trigger
	if pb.Status == sdk.StatusSuccess {
		pipelineBuildEnd(tx, pb)
	}

	if err := tx.Commit(); err != nil {
		log.Warning("RunActions> Cannot commit tx on pb %d: %s\n", pb.ID, err)
	}
}

func addJobsToQueue(tx gorp.SqlExecutor, stage *sdk.Stage, pb sdk.PipelineBuild) error {
	//Check stage prerequisites
	prerequisitesOK, err := pipeline.CheckPrerequisites(*stage, pb)
	if err != nil {
		log.Warning("addJobsToQueue> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d): %s\n", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID, err)
		return err
	}
	stage.Status = sdk.StatusBuilding

	for _, job := range stage.Jobs {
		pbJobParams, errParam := getPipelineBuildJobParameters(tx, job, pb)
		if errParam != nil {
			log.Warning("addJobsToQueue> Cannot get action build parameters for pipeline build %d: %s\n", pb.ID, err)
			return errParam
		}
		pbJob := sdk.PipelineBuildJob{
			PipelineBuildID: pb.ID,
			Parameters:      pbJobParams,
			Job:             job,
			Queued:          time.Now(),
			Status:          sdk.StatusWaiting.String(),
			Start:           time.Now(),
		}

		if !stage.Enabled {
			pbJob.Status = sdk.StatusDisabled.String()
		} else if !prerequisitesOK {
			pbJob.Status = sdk.StatusSkipped.String()
		}
		if err := pipeline.InsertPipelineBuildJob(tx, &pbJob); err != nil {
			log.Warning("addJobToQueue> Cannot insert job in queue for pipeline build %d: %s\n", pb.ID, err)
			return err
		}
		event.PublishActionBuild(&pb, &pbJob)
		stage.PipelineBuildJobs = append(stage.PipelineBuildJobs, pbJob)
	}

	return nil
}

func syncPipelineBuildJob(db gorp.SqlExecutor, stage *sdk.Stage) (bool, error) {
	stageEnd := true
	finalStatus := sdk.StatusBuilding

	// browse all running jobs
	for indexJob := range stage.PipelineBuildJobs {
		pbJob := &stage.PipelineBuildJobs[indexJob]
		// If job is runnning, sync it
		if pbJob.Status == sdk.StatusBuilding.String() || pbJob.Status == sdk.StatusWaiting.String() {
			pbJobDB, errJob := pipeline.GetPipelineBuildJob(db, pbJob.ID)
			if errJob != nil {
				return stageEnd, errJob
			}

			if pbJobDB.Status == sdk.StatusBuilding.String() || pbJobDB.Status == sdk.StatusWaiting.String() {
				stageEnd = false
			}

			// If already sync
			if pbJobDB.Status == pbJob.Status {
				continue
			}

			pbJob.Status = pbJobDB.Status
			pbJob.Start = pbJobDB.Start
			pbJob.Done = pbJobDB.Done
			pbJob.Model = pbJobDB.Model
		}
	}

	if stageEnd || len(stage.PipelineBuildJobs) == 0 {
		if len(stage.PipelineBuildJobs) == 0 {
			finalStatus = sdk.StatusSuccess
			stageEnd = true
		}
		// Determine final stage status
	finalStageLoop:

		for _, buildJob := range stage.PipelineBuildJobs {
			switch buildJob.Status {
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

func pipelineBuildEnd(tx gorp.SqlExecutor, pb sdk.PipelineBuild) {
	// run trigger
	triggers, err := trigger.LoadAutomaticTriggersAsSource(tx, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
		if ok && pqerr.Code == "55P03" {
			return
		}
		if ok {
			log.Warning("pipelineBuildEnd> Cannot load trigger: %s (%s)\n", pqerr, pqerr.Code)
			return
		}
		log.Warning("pipelineBuildEnd> Cannot load trigger for %s-%s-%s[%s] (%d, %d, %d): %s\n", pb.Pipeline.ProjectKey, pb.Application.Name, pb.Pipeline.Name, pb.Environment.Name, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, err)
		return
	}

	if len(triggers) > 0 {
		log.Debug("(v%d) Loaded %d potential triggers for  %s[%s]", pb.Version, len(triggers), pb.Pipeline.Name, pb.Environment.Name)
	}
	for _, t := range triggers {

		// Check prerequisites
		log.Debug("Checking %d prerequisites for trigger %s/%s/%s -> %s/%s/%s\n", len(t.Prerequisites), t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name)
		prereqOK, err := trigger.CheckPrerequisites(t, pb)
		if err != nil {
			log.Warning("pipelineScheduler> Cannot check trigger prereq: %s\n", err)
			continue
		}
		if !prereqOK {
			log.Debug("Prerequisites not met for trigger %s/%s/%s[%s] -> %s/%s/%s[%s]\n", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)
			continue
		}

		parameters := t.Parameters
		// Add parent build info
		parentParams, err := ParentBuildInfos(&pb)
		if err != nil {
			log.Warning("pipelineBuildEnd> Cannot create parent build infos: %s\n", err)
			continue
		}
		parameters = append(parameters, parentParams...)

		// Start build
		app, err := application.LoadApplicationByName(tx, t.DestProject.Key, t.DestApplication.Name, application.WithClearPassword())
		if err != nil {
			log.Warning("pipelineBuildEnd> Cannot load destination application: %s\n", err)
			continue
		}

		log.Info("Prerequisites OK for trigger %s/%s/%s-%s -> %s/%s/%s-%s (version %d)\n", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name, pb.Version)

		trigger := sdk.PipelineBuildTrigger{
			ManualTrigger:       false,
			TriggeredBy:         pb.Trigger.TriggeredBy,
			ParentPipelineBuild: &pb,
			VCSChangesAuthor:    pb.Trigger.VCSChangesAuthor,
			VCSChangesBranch:    pb.Trigger.VCSChangesBranch,
			VCSChangesHash:      pb.Trigger.VCSChangesHash,
		}

		_, err = RunPipeline(tx, t.DestProject.Key, app, t.DestPipeline.Name, t.DestEnvironment.Name, parameters, pb.Version, trigger, &sdk.User{Admin: true})
		if err != nil {
			log.Warning("pipelineScheduler> Cannot run pipeline on project %s, application %s, pipeline %s, env %s: %s\n", t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name, err)
			continue
		}
	}
}

// ParentBuildInfos fetch parent build data and injects them as {{.cds.parent.*}} parameters
func ParentBuildInfos(pb *sdk.PipelineBuild) ([]sdk.Parameter, error) {
	var params []sdk.Parameter

	p := sdk.Parameter{
		Name:  "cds.parent.buildNumber",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("%d", pb.BuildNumber),
	}
	params = append(params, p)

	p = sdk.Parameter{
		Name:  "cds.parent.pipeline",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("%s", pb.Pipeline.Name),
	}
	params = append(params, p)

	p = sdk.Parameter{
		Name:  "cds.parent.application",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("%s", pb.Application.Name),
	}
	params = append(params, p)

	p = sdk.Parameter{
		Name:  "cds.parent.environment",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("%s", pb.Environment.Name),
	}
	params = append(params, p)

	return params, nil
}

func getPipelineBuildJobParameters(db gorp.SqlExecutor, j sdk.Job, pb sdk.PipelineBuild) ([]sdk.Parameter, error) {

	// Get project and pipeline Information
	projectData, err := project.LoadProjectByPipelineActionID(db, j.PipelineActionID)
	if err != nil {
		log.Debug("getActionBuildParameters> err LoadProjectAndPipelineByPipelineActionID: %s", err)
		return nil, err
	}

	// Load project Variables
	projectVariables, err := project.GetAllVariableInProject(db, projectData.ID)
	if err != nil {
		log.Debug("getActionBuildParameters> err GetAllVariableInProject: %s", err)
		return nil, err
	}
	// Load application Variables
	appVariables, err := application.GetAllVariableByID(db, pb.Application.ID)
	if err != nil {
		log.Debug("getActionBuildParameters> err GetAllVariableByID for app ID: %s", err)
		return nil, err
	}
	// Load environment Variables
	envVariables, err := environment.GetAllVariableByID(db, pb.Environment.ID)
	if err != nil {
		log.Debug("getActionBuildParameters> err GetAllVariableByID for env ID : %s", err)
		return nil, err
	}

	pipelineParameters, err := pipeline.GetAllParametersInPipeline(db, pb.Pipeline.ID)
	if err != nil {
		log.Debug("getActionBuildParameters> err GetAllParametersInPipeline: %s", err)
		return nil, err
	}

	/* Create and process the full set of build variables from
	** - Project variables
	** - Pipeline variables
	** - Action definition in pipeline
	** - ActionBuild variables (global ones + trigger parameters)
	**
	** -> Replaces all placeholder but PasswordParameter
	 */
	params, err := action.ProcessActionBuildVariables(
		projectVariables,
		appVariables,
		envVariables,
		pipelineParameters,
		pb.Parameters, j.Action)
	return params, nil
}
