package queue

import (
	"database/sql"
	"fmt"
	"time"

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

		db := database.DB()
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
func RunActions(db *sql.DB, pb sdk.PipelineBuild) {
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

	// OH! AN EMPTY PIPELINE
	if len(pb.Pipeline.Stages) == 0 {
		// Pipeline is done
		pipelineBuildEnd(tx, pb)
		return
	}

	var runningStage = -1
	var doneStage = 0
	for stageIndex, s := range pb.Pipeline.Stages {
		// Need len(s.Actions) on Success to go to next stage, count them
		var numberOfActionSuccess int

		// browse all actions in the current stage
		for _, a := range s.Actions {
			// get action status
			status, errActionStatus := pipeline.LoadActionStatus(tx, a.PipelineActionID, pb.ID)
			if errActionStatus != nil && errActionStatus != sql.ErrNoRows {
				log.Warning("queue.RunActions> Cannot load action %s with pipelineBuildID %d: %s\n", a.Name, pb.ID, errActionStatus)
				return
			}

			//Check stage prerequisites
			prerequisitesOK, errc := pipeline.CheckPrerequisites(s, pb)
			if errc != nil {
				log.Warning("queue.RunActions> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d): %s\n", s.Name, s.ID, pb.Pipeline.Name, pb.ID, errc)
				return
			}
			//If stage is disabled, we have to disable all actions
			if !s.Enabled || !prerequisitesOK {
				//newActionBuild, and set it to disabled
				if errActionStatus != nil && errActionStatus == sql.ErrNoRows && (runningStage == -1 || stageIndex == runningStage) {
					var actionBuild *sdk.ActionBuild
					actionBuild, err = newActionBuild(tx, a, pb, s.ID)
					if err != nil {
						log.Warning("queue.RunActions> Cannot schedule action: %s\n", err)
						return
					}
					runningStage = stageIndex

					if !s.Enabled {
						status = sdk.StatusDisabled
					} else {
						status = sdk.StatusSkipped
					}

					log.Debug("queue.RunActions> Disable action %d %s (status=%s)", actionBuild.ID, actionBuild.ActionName, status)
					if err := pipeline.UpdateActionBuildStatus(tx, actionBuild, status); err != nil {
						log.Warning("queue.RunActions> Cannot disable action %s with pipelineBuildID %d: %s\n", a.Name, pb.ID, err)
					}

					continue
				}
				numberOfActionSuccess++
			} else {
				// If no row, action should be scheduled if current stage is running
				if errActionStatus != nil && errActionStatus == sql.ErrNoRows {
					if runningStage == -1 || stageIndex == runningStage {
						_, err = newActionBuild(tx, a, pb, s.ID)
						if err != nil {
							log.Warning("queue.RunActions> Cannot schedule action: %s\n", err)
							return
						}
						runningStage = stageIndex
						continue
					}
				}
				if status == sdk.StatusSuccess || status == sdk.StatusDisabled {
					numberOfActionSuccess++
				}

				//condition de sortie
				if status == sdk.StatusFail {
					if err := pipeline.UpdatePipelineBuildStatus(tx, pb, status); err != nil {
						log.Warning("queue.RunActions> Cannot update pipeline status: %s\n", err)
					} else {
						err = tx.Commit()
						if err != nil {
							log.Warning("queue.RunActions> Cannot commit tx on pb %d: %s\n", pb.ID, err)
						}
					}
					return
				}

			}
			if status == sdk.StatusBuilding || status == sdk.StatusWaiting {
				runningStage = stageIndex
			}
		}
		// If all action are done or skipped AND previous stage is done, then current stage is done
		if numberOfActionSuccess == len(s.Actions) && doneStage == stageIndex {
			doneStage = stageIndex + 1
			log.Debug("queue.RunActions> Stage #%d is DONE\n", doneStage)
		}

		// If all actions in stage are OK or skipped
		if numberOfActionSuccess == len(s.Actions) { // Then go to next stage !
			// But if current stage is the last one...
			if doneStage == len(pb.Pipeline.Stages) { // Oh wait there is no next stage
				pipelineBuildEnd(tx, pb)
				return
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("queue.RunActions>Cannot commit transaction: %s", err)
		return
	}
	return

}

func pipelineBuildEnd(tx *sql.Tx, pb sdk.PipelineBuild) {
	log.Debug("pipelineBuildEnd> Updating pipeline build %d status to Success", pb.ID)

	if err := pipeline.UpdatePipelineBuildStatus(tx, pb, sdk.StatusSuccess); err != nil {
		log.Warning("pipelineBuildEnd> Cannot update pipeline status: %s\n", err)
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			log.Warning("pipelineBuildEnd> Cannot commit tx on pb %d: %s\n", pb.ID, err)
		}
	}()

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
		parentParams, err := ParentBuildInfos(pb)
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
func ParentBuildInfos(pb sdk.PipelineBuild) ([]sdk.Parameter, error) {
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

func newActionBuild(db database.QueryExecuter, a sdk.Action, pb sdk.PipelineBuild, stageID int64) (*sdk.ActionBuild, error) {
	log.Info("newActionBuild> Starting action %s for pipeline %s #%d\n", a.Name,
		pb.Pipeline.Name, pb.BuildNumber)

	pipelineActionArgs, err := loadPipelineActionArguments(db, a.PipelineActionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Debug("newActionBuild> err loadPipelineActionArguments: %s", err)
		return nil, err
	}

	// Get project and pipeline Information
	projectData, pipelineData, err := project.LoadProjectAndPipelineByPipelineActionID(db, a.PipelineActionID)
	if err != nil {
		log.Debug("newActionBuild> err LoadProjectAndPipelineByPipelineActionID: %s", err)
		return nil, err
	}

	// Load project Variables
	projectVariables, err := project.GetAllVariableInProject(db, projectData.ID)
	if err != nil {
		log.Debug("newActionBuild> err GetAllVariableInProject: %s", err)
		return nil, err
	}
	// Load application Variables
	appVariables, err := application.GetAllVariableByID(db, pb.Application.ID)
	if err != nil {
		log.Debug("newActionBuild> err GetAllVariableByID for app ID: %s", err)
		return nil, err
	}
	// Load environment Variables
	envVariables, err := environment.GetAllVariableByID(db, pb.Environment.ID)
	if err != nil {
		log.Debug("newActionBuild> err GetAllVariableByID for env ID : %s", err)
		return nil, err
	}

	pipelineParameters, err := pipeline.GetAllParametersInPipeline(db, pipelineData.ID)
	if err != nil {
		log.Debug("newActionBuild> err GetAllParametersInPipeline: %s", err)
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
		pipelineActionArgs,
		pb.Parameters, a)

	if err != nil {
		log.Debug("newActionBuild> err ProcessActionBuildVariables: %s", err)
		return nil, err
	}

	ab := sdk.ActionBuild{
		PipelineBuildID:  pb.ID,
		PipelineID:       pb.Pipeline.ID,
		PipelineActionID: a.PipelineActionID,
		Args:             params,
		ActionName:       a.Name,
		Status:           sdk.StatusWaiting,
	}

	if !a.Enabled {
		ab.Status = sdk.StatusDisabled
		ab.Done = time.Now()
	}

	if err := InsertActionBuild(db, &ab); err != nil {
		log.Debug("newActionBuild> err InsertBuild: %s", err)
		return nil, fmt.Errorf("Cannot push action %s for pipeline %s #%d in build queue: %s\n",
			a.Name, pb.Pipeline.Name, ab.PipelineBuildID, err)
	}

	event.PublishActionBuild(&pb, &ab)

	return &ab, nil
}
