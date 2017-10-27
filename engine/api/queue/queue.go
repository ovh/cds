package queue

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Pipelines is a goroutine responsible for pushing actions of a building pipeline in queue, in the wanted order
func Pipelines(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(2 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting queue.Pipelines: %v", c.Err())
				return
			}
		case <-tick:
			db := DBFunc()
			if db != nil {
				ids, err := pipeline.LoadBuildingPipelinesIDs(db)
				if err != nil {
					log.Warning("queue.Pipelines> Cannot load building pipelines: %s", err)
					continue
				}

				for _, id := range ids {
					runPipeline(DBFunc, store, id)
				}
			}
		}
	}
}

func runPipeline(DBFunc func() *gorp.DbMap, store cache.Store, pbID int64) {
	//Check if CDS is in maintenance mode
	var m bool
	store.Get("maintenance", &m)
	if m {
		log.Info("⚠ CDS maintenance in ON")
	}

	db := DBFunc()
	tx, errb := db.Begin()
	if errb != nil {
		log.Warning("queue.RunActions> cannot start tx for pb %d: %s", pbID, errb)
		return
	}
	defer tx.Rollback()

	// Reload pipeline build with a FOR UPDATE NOT WAIT
	// So only one instance of the API can update it and/or end it
	if err := pipeline.SelectBuildForUpdate(tx, pbID); err != nil {
		// if ErrNoRows, pipelines is already done
		if err == sql.ErrNoRows {
			return
		}
		pqerr, ok := err.(*pq.Error)
		// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
		if ok && pqerr.Code == "55P03" {
			return
		}
		log.Warning("queue.RunActions> Cannot load pb: %s", err)
		return
	}

	pb, errPB := pipeline.LoadPipelineBuildByID(db, pbID)
	if errPB != nil {
		log.Warning("queue.RunActions> Cannot load pb [%d]: %s", pbID, errPB)
		return
	}

	if pb.Status != sdk.StatusBuilding {
		return
	}

	pbNewStatus := sdk.StatusBuilding

	if len(pb.Stages) == 0 {
		// Pipeline is done
		pbNewStatus = sdk.StatusSuccess
	}

	// Browse Stage
	for stageIndex := range pb.Stages {
		stage := &pb.Stages[stageIndex]

		//We only add jobs to queue if we are not in maintenance
		if stage.Status == sdk.StatusWaiting && !m {
			if err := addJobsToQueue(tx, stage, pb); err != nil {
				log.Warning("queue.RunActions> Cannot add job to queue: %s", err)
				return
			}
			break
		}

		if stage.Status == sdk.StatusBuilding {
			end, errSync := syncPipelineBuildJob(tx, stage)
			if errSync != nil {
				log.Warning("queue.RunActions> Cannot sync building jobs on stage %s(%d) of pipeline %s(%d): %s", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID, errSync)
				return
			}

			if end {
				// Remove pipeline build job
				if err := pipeline.DeletePipelineBuildJob(tx, pb.ID); err != nil {
					log.Warning("queue.RunActions> Cannot remove pipeline build jobs for pipeline build %d: %s", pb.ID, err)
					return
				}

				if stage.Status == sdk.StatusFail {
					pb.Done = time.Now()
					pbNewStatus = sdk.StatusFail
					break
				}
				if stageIndex == len(pb.Stages)-1 {
					pb.Done = time.Now()
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

	if err := pipeline.UpdatePipelineBuildStatusAndStage(tx, pb, pbNewStatus); err != nil {
		log.Warning("RunActions> Cannot update UpdatePipelineBuildStatusAndStage on pb %d: %s", pb.ID, err)
		return
	}

	// If pipeline build succeed, run trigger
	if pb.Status == sdk.StatusSuccess {
		if err := pipelineBuildEnd(DBFunc, store, tx, pb); err != nil {
			log.Warning("RunActions> Cannot execute pipelineBuildEnd: %s", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("RunActions> Cannot commit tx on pb %d: %s\n", pb.ID, err)
		return
	}
}

func addJobsToQueue(tx gorp.SqlExecutor, stage *sdk.Stage, pb *sdk.PipelineBuild) error {
	//Check stage prerequisites
	prerequisitesOK, err := pipeline.CheckPrerequisites(*stage, pb)
	if err != nil {
		return sdk.WrapError(err, "addJobsToQueue> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d)", stage.Name, stage.ID, pb.Pipeline.Name, pb.ID)
	}
	stage.Status = sdk.StatusBuilding

	for _, job := range stage.Jobs {
		pbJobParams, errParam := getPipelineBuildJobParameters(tx, job, pb, stage)
		if errParam != nil {
			return sdk.WrapError(errParam, "addJobsToQueue> error on getPipelineBuildJobParameters")
		}
		groups, errGroups := getPipelineBuildJobExecutablesGroups(tx, pb)
		if errGroups != nil {
			return sdk.WrapError(errGroups, "addJobsToQueue> error on getPipelineBuildJobExecutablesGroups")
		}
		pbJob := sdk.PipelineBuildJob{
			PipelineBuildID: pb.ID,
			Parameters:      pbJobParams,
			Job: sdk.ExecutedJob{
				Job: job,
			},
			ExecGroups: groups,
			Queued:     time.Now(),
			Status:     sdk.StatusWaiting.String(),
			Start:      time.Now(),
		}

		if !stage.Enabled || !pbJob.Job.Enabled {
			pbJob.Status = sdk.StatusDisabled.String()
		} else if !prerequisitesOK {
			pbJob.Status = sdk.StatusSkipped.String()
		}
		if err := pipeline.InsertPipelineBuildJob(tx, &pbJob); err != nil {
			return sdk.WrapError(err, "addJobToQueue> Cannot insert job in queue for pipeline build %d", pb.ID)
		}

		event.PublishActionBuild(pb, &pbJob)
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
			case sdk.StatusStopped.String():
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

func pipelineBuildEnd(DBFunc func() *gorp.DbMap, store cache.Store, tx gorp.SqlExecutor, pb *sdk.PipelineBuild) error {
	// run trigger
	triggers, err := trigger.LoadAutomaticTriggersAsSource(tx, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
		if ok && pqerr.Code == "55P03" {
			return pqerr
		}
		if ok {
			return sdk.WrapError(pqerr, "pipelineBuildEnd> Cannot load trigger: %s", pqerr.Code)
		}
		return sdk.WrapError(err, "pipelineBuildEnd> Cannot load trigger for %s-%s-%s[%s] (%d, %d, %d)", pb.Pipeline.ProjectKey, pb.Application.Name, pb.Pipeline.Name, pb.Environment.Name, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
	}

	if len(triggers) > 0 {
		log.Debug("(v%d) Loaded %d potential triggers for  %s[%s]", pb.Version, len(triggers), pb.Pipeline.Name, pb.Environment.Name)
	}

	for _, t := range triggers {
		// Check prerequisites
		log.Debug("Checking %d prerequisites for trigger %s/%s/%s -> %s/%s/%s", len(t.Prerequisites), t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name)
		prereqOK, err := trigger.CheckPrerequisites(t, pb)
		if err != nil {
			log.Warning("pipelineScheduler> Cannot check trigger prereq: %s", err)
			continue
		}
		if !prereqOK {
			log.Debug("Prerequisites not met for trigger %s/%s/%s[%s] -> %s/%s/%s[%s]", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)
			continue
		}

		parameters := t.Parameters
		// Add parent build info
		parentParams := ParentBuildInfos(pb)
		parameters = append(parameters, parentParams...)

		// Start build
		app, err := application.LoadByName(tx, store, t.DestProject.Key, t.DestApplication.Name, nil, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
		if err != nil {
			return sdk.WrapError(err, "pipelineBuildEnd> Cannot load destination application")
		}

		log.Debug("Prerequisites OK for trigger %s/%s/%s-%s -> %s/%s/%s-%s (version %d)", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name, t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name, pb.Version)

		trigger := sdk.PipelineBuildTrigger{
			ManualTrigger:       false,
			TriggeredBy:         pb.Trigger.TriggeredBy,
			ParentPipelineBuild: pb,
			VCSChangesAuthor:    pb.Trigger.VCSChangesAuthor,
			VCSChangesBranch:    pb.Trigger.VCSChangesBranch,
			VCSChangesHash:      pb.Trigger.VCSChangesHash,
			VCSRemote:           pb.Trigger.VCSRemote,
			VCSRemoteURL:        pb.Trigger.VCSRemoteURL,
			ScheduledTrigger:    pb.Trigger.ScheduledTrigger,
		}

		_, err = RunPipeline(DBFunc, store, tx, t.DestProject.Key, app, t.DestPipeline.Name, t.DestEnvironment.Name, parameters, pb.Version, trigger, &sdk.User{Admin: true})
		if err != nil {
			return sdk.WrapError(err, "pipelineScheduler> Cannot run pipeline on project %s, application %s, pipeline %s, env %s", t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)
		}
	}
	return nil
}

// ParentBuildInfos fetch parent build data and injects them as {{.cds.parent.*}} parameters
func ParentBuildInfos(pb *sdk.PipelineBuild) []sdk.Parameter {
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

	return params
}

func getPipelineBuildJobParameters(db gorp.SqlExecutor, j sdk.Job, pb *sdk.PipelineBuild, stage *sdk.Stage) ([]sdk.Parameter, error) {
	projectVariables, err := project.GetAllVariableInProject(db, pb.Pipeline.ProjectID)
	if err != nil {
		return nil, sdk.WrapError(err, "getPipelineBuildJobParameters> err GetAllVariableInProject on ID %d", pb.Pipeline.ProjectID)
	}
	// Load application Variables
	appVariables, err := application.GetAllVariableByID(db, pb.Application.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "getPipelineBuildJobParameters> err GetAllVariableByID for app ID %d", pb.Application.ID)
	}
	// Load environment Variables
	envVariables, err := environment.GetAllVariableByID(db, pb.Environment.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "getPipelineBuildJobParameters> err GetAllVariableByID for env ID %d", pb.Environment.ID)
	}

	pipelineParameters, err := pipeline.GetAllParametersInPipeline(db, pb.Pipeline.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "getPipelineBuildJobParameters> err GetAllParametersInPipeline for pip %d", pb.Pipeline.ID)
	}

	return action.ProcessActionBuildVariables(projectVariables, appVariables, envVariables, pipelineParameters, pb.Parameters, stage, j.Action), nil
}

func getPipelineBuildJobExecutablesGroups(db gorp.SqlExecutor, pb *sdk.PipelineBuild) ([]sdk.Group, error) {
	query := `
	SELECT distinct("group".id), "group".name FROM "group"
	LEFT JOIN application_group ON application_group.group_id = "group".id
	LEFT JOIN pipeline_group ON pipeline_group.group_id = "group".id
	LEFT JOIN project_group ON project_group.group_id = "group".id
	LEFT JOIN pipeline_build ON pipeline_build.pipeline_id = pipeline_group.pipeline_id
	LEFT OUTER JOIN environment_group ON environment_group.group_id = "group".id
	WHERE pipeline_build.id = $1
		AND pipeline_build.pipeline_id = pipeline_group.pipeline_id
		AND pipeline_group.group_id = "group".id
		AND application_group.role >= $2
		AND pipeline_group.role >= $2
		AND (environment_group.role >= $2 OR pipeline_build.environment_id = $3);
	`

	var groups []sdk.Group
	rows, err := db.Query(query, pb.ID, permission.PermissionReadExecute, sdk.DefaultEnv.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "getPipelineBuildJobExecutablesGroups> err query")
	}
	defer rows.Close()

	var sharedInfraIn bool
	for rows.Next() {
		var g sdk.Group
		var groupID sql.NullInt64
		var groupName sql.NullString

		if err := rows.Scan(&groupID, &groupName); err != nil {
			return nil, sdk.WrapError(err, "getPipelineBuildJobExecutablesGroups> err scan")
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
