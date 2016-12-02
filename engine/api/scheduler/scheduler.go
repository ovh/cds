package scheduler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Schedule is a goroutine responsible for pushing actions of a building pipeline in queue, in the wanted order
func Schedule() {

	// If this goroutine exits, then it's a crash
	defer log.Fatalf("Goroutine of scheduler.Schedule exited - Exit CDS Engine")

	for {
		time.Sleep(2 * time.Second)

		db := database.DB()
		if db != nil {
			pipelines, err := pipeline.LoadBuildingPipelines(db)
			if err != nil {
				log.Warning("Schedule> Cannot load building pipelines: %s\n", err)
				// Add some extra sleep if db is down...
				time.Sleep(3 * time.Second)
				continue
			}

			for i := range pipelines {
				PipelineScheduler(db, pipelines[i])
			}
		}
	}
}

// PipelineScheduler Schedule action for the given Build
func PipelineScheduler(db *sql.DB, pb sdk.PipelineBuild) {
	tx, err := db.Begin()
	if err != nil {
		log.Warning("PipelineScheduler> cannot start tx for pb %d: %s\n", pb.ID, err)
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
		log.Warning("PipelineScheduler> Cannot load pb: %s\n", err)
		return
	}

	// OH! AN EMPTY PIPELINE
	if len(pb.Pipeline.Stages) == 0 {
		// Pipeline is done
		scheduleEnd(tx, pb)
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
				log.Warning("PipelineScheduler> Cannot load action %s with pipelineBuildID %d: %s\n", a.Name, pb.ID, errActionStatus)
				return
			}

			//Check stage prerequisites
			prerequisitesOK, err := pipeline.CheckPrerequisites(s, pb)
			if err != nil {
				log.Warning("PipelineScheduler> Cannot compute prerequisites on stage %s(%d) of pipeline %s(%d): %s\n", s.Name, s.ID, pb.Pipeline.Name, pb.ID, err)
				return
			}
			//If stage is disabled, we have to disable all actions
			if !s.Enabled || !prerequisitesOK {
				//scheduleAction, and set it to disabled
				if errActionStatus != nil && errActionStatus == sql.ErrNoRows && (runningStage == -1 || stageIndex == runningStage) {
					var actionBuild *sdk.ActionBuild
					actionBuild, err = scheduleAction(tx, a, pb, s.ID)
					if err != nil {
						log.Warning("PipelineScheduler> Cannot schedule action: %s\n", err)
						return
					}
					runningStage = stageIndex

					if !s.Enabled {
						status = sdk.StatusDisabled
					} else {
						status = sdk.StatusSkipped
					}

					log.Debug("PipelineScheduler> Disable action %d %s (status=%s)", actionBuild.ID, actionBuild.ActionName, status)
					if err := build.UpdateActionBuildStatus(tx, actionBuild, status); err != nil {
						log.Warning("PipelineScheduler> Cannot disable action %s with pipelineBuildID %d: %s\n", a.Name, pb.ID, err)
					}

					continue
				}
				numberOfActionSuccess++
			} else {
				// If no row, action should be scheduled if current stage is running
				if errActionStatus != nil && errActionStatus == sql.ErrNoRows {
					if runningStage == -1 || stageIndex == runningStage {
						_, err = scheduleAction(tx, a, pb, s.ID)
						if err != nil {
							log.Warning("PipelineScheduler> Cannot schedule action: %s\n", err)
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
					//log.Info("PipelineScheduler> %s #%d: Action %s failed, stoping\n", pb.Pipeline.Name, pb.BuildNumber, a.Name)
					if err := pipeline.UpdatePipelineBuildStatus(tx, pb, status); err != nil {
						log.Warning("PipelineScheduler> Cannot update pipeline status: %s\n", err)
					} else {
						err = tx.Commit()
						if err != nil {
							log.Warning("PipelineScheduler> Cannot commit tx on pb %d: %s\n", pb.ID, err)
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
			log.Debug("PipelineScheduler> Stage #%d is DONE\n", doneStage)
		}

		// If all actions in stage are OK or skipped
		if numberOfActionSuccess == len(s.Actions) { // Then go to next stage !
			// But if current stage is the last one...
			if doneStage == len(pb.Pipeline.Stages) { // Oh wait there is no next stage
				scheduleEnd(tx, pb)
				return
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("PipelineScheduler>Cannot commit transaction: %s", err)
		return
	}
	return

}

func scheduleEnd(tx *sql.Tx, pb sdk.PipelineBuild) {
	log.Debug("buildScheduler> Updating pipeline build %d status to Success", pb.ID)

	if err := pipeline.UpdatePipelineBuildStatus(tx, pb, sdk.StatusSuccess); err != nil {
		log.Warning("pipelineScheduler> Cannot update pipeline status: %s\n", err)
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			log.Warning("scheduleEnd> Cannot commit tx on pb %d: %s\n", pb.ID, err)
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
			log.Warning("scheduleEnd> Cannot load trigger: %s (%s)\n", pqerr, pqerr.Code)
			return
		}
		log.Warning("scheduleEnd> Cannot load trigger for %s-%s-%s[%s] (%d, %d, %d): %s\n", pb.Pipeline.ProjectKey, pb.Application.Name, pb.Pipeline.Name, pb.Environment.Name, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, err)
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
			log.Warning("scheduleEnd> Cannot create parent build infos: %s\n", err)
			continue
		}
		parameters = append(parameters, parentParams...)

		// Start build
		app, err := application.LoadApplicationByName(tx, t.DestProject.Key, t.DestApplication.Name, application.WithClearPassword())
		if err != nil {
			log.Warning("scheduleEnd> Cannot load destination application: %s\n", err)
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

		_, err = Run(tx, t.DestProject.Key, app, t.DestPipeline.Name, t.DestEnvironment.Name, parameters, pb.Version, trigger, &sdk.User{Admin: true})
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

func scheduleAction(db database.QueryExecuter, a sdk.Action, pb sdk.PipelineBuild, stageID int64) (*sdk.ActionBuild, error) {
	log.Info("scheduleAction> Starting action %s for pipeline %s #%d\n", a.Name,
		pb.Pipeline.Name, pb.BuildNumber)

	pipelineActionArgs, err := loadPipelineActionArguments(db, a.PipelineActionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Debug("scheduleAction> err loadPipelineActionArguments: %s", err)
		return nil, err
	}

	// Get project and pipeline Information
	projectData, pipelineData, err := project.LoadProjectAndPipelineByPipelineActionID(db, a.PipelineActionID)
	if err != nil {
		log.Debug("scheduleAction> err LoadProjectAndPipelineByPipelineActionID: %s", err)
		return nil, err
	}

	// Load project Variables
	projectVariables, err := project.GetAllVariableInProject(db, projectData.ID)
	if err != nil {
		log.Debug("scheduleAction> err GetAllVariableInProject: %s", err)
		return nil, err
	}
	// Load application Variables
	appVariables, err := application.GetAllVariableByID(db, pb.Application.ID)
	if err != nil {
		log.Debug("scheduleAction> err GetAllVariableByID for app ID: %s", err)
		return nil, err
	}
	// Load environment Variables
	envVariables, err := environment.GetAllVariableByID(db, pb.Environment.ID)
	if err != nil {
		log.Debug("scheduleAction> err GetAllVariableByID for env ID : %s", err)
		return nil, err
	}

	pipelineParameters, err := pipeline.GetAllParametersInPipeline(db, pipelineData.ID)
	if err != nil {
		log.Debug("scheduleAction> err GetAllParametersInPipeline: %s", err)
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
		log.Debug("scheduleAction> err ProcessActionBuildVariables: %s", err)
		return nil, err
	}

	b := sdk.ActionBuild{
		PipelineBuildID:  pb.ID,
		PipelineID:       pb.Pipeline.ID,
		PipelineActionID: a.PipelineActionID,
		Args:             params,
		ActionName:       a.Name,
		Status:           sdk.StatusWaiting,
	}

	if !a.Enabled {
		b.Status = sdk.StatusDisabled
		b.Done = time.Now()
	}

	if err := InsertBuild(db, &b); err != nil {
		log.Debug("scheduleAction> err InsertBuild: %s", err)
		return nil, fmt.Errorf("Cannot push action %s for pipeline %s #%d in build queue: %s\n",
			a.Name, pb.Pipeline.Name, b.PipelineBuildID, err)
	}

	return &b, nil
}

func loadPipelineActionArguments(db database.Querier, pipelineActionID int64) ([]sdk.Parameter, error) {
	query := `SELECT args FROM pipeline_action WHERE id = $1`

	var argsJSON sql.NullString
	if err := db.QueryRow(query, pipelineActionID).Scan(&argsJSON); err != nil {
		return nil, err
	}

	var parameters []sdk.Parameter
	if argsJSON.Valid {
		if err := json.Unmarshal([]byte(argsJSON.String), &parameters); err != nil {
			return nil, err
		}
	}

	return parameters, nil
}

// InsertBuild Insert new action build
func InsertBuild(db database.QueryExecuter, b *sdk.ActionBuild) error {
	query := `INSERT INTO action_build (pipeline_action_id, args, status, pipeline_build_id, queued, start, done) VALUES($1, $2, $3, $4, $5, $5, $6) RETURNING id`

	if b.PipelineActionID == 0 {
		return fmt.Errorf("invalid pipeline action ID (0)")
	}

	if b.PipelineBuildID == 0 {
		return fmt.Errorf("invalid pipeline build ID (0)")
	}

	argsJSON, err := json.Marshal(b.Args)
	if err != nil {
		return err
	}

	if b.Status == "" {
		b.Status = sdk.StatusWaiting
	}

	//Set action_build.done to null is not set
	var done interface{}
	if b.Done.IsZero() {
		done = sql.NullString{
			String: "",
			Valid:  false,
		}
	} else {
		done = b.Done
	}

	err = db.QueryRow(query, b.PipelineActionID, string(argsJSON), b.Status.String(), b.PipelineBuildID, time.Now(), done).Scan(&b.ID)
	if err != nil {
		return err
	}

	notification.SendActionBuild(db, b, sdk.CreateNotifEvent, sdk.StatusWaiting)
	return nil
}

// Run  the given pipeline with the given parameters
func Run(db *sql.Tx, projectKey string, app *sdk.Application, pipelineName string, environmentName string, params []sdk.Parameter, version int64, trigger sdk.PipelineBuildTrigger, user *sdk.User) (*sdk.PipelineBuild, error) {

	// Load pipeline + Args
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load pipeline %s: %s\n", pipelineName, err)
		return nil, err
	}
	parameters, err := pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load pipeline %s parameters: %s\n", pipelineName, err)
		return nil, err
	}
	p.Parameter = parameters

	// Pipeline type check
	if p.Type == sdk.BuildPipeline && environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		log.Warning("scheduler.Run> Pipeline %s/%s/%s is a %s pipeline, but environment '%s' was provided\n", projectKey, app.Name, pipelineName, p.Type, environmentName)
		return nil, sdk.ErrEnvironmentProvided
	}
	if p.Type != sdk.BuildPipeline && (environmentName == "" || environmentName == sdk.DefaultEnv.Name) {
		log.Warning("scheduler.Run> Pipeline %s/%s/%s is a %s pipeline, but no environment was provided\n", projectKey, app.Name, pipelineName, p.Type)
		return nil, sdk.ErrNoEnvironmentProvided
	}

	applicationPipelineParams, err := application.GetAllPipelineParam(db, app.ID, p.ID)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load application pipeline args: %s\n", err)
		return nil, err
	}

	// Load project + var
	projectData, err := project.LoadProject(db, projectKey, user)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load project %s: %s\n", projectKey, err)
		return nil, err
	}
	projectsVar, err := project.GetAllVariableInProject(db, projectData.ID, project.WithClearPassword())
	if err != nil {
		log.Warning("scheduler.Run> Cannot load project variable: %s\n", err)
		return nil, err
	}
	projectData.Variable = projectsVar
	var env *sdk.Environment
	if environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, projectKey, environmentName)
		if err != nil {
			log.Warning("scheduler.Run> Cannot load environment %s for project %s: %s\n", environmentName, projectKey, err)
			return nil, err
		}
	} else {
		env = &sdk.DefaultEnv
	}

	pb, err := pipeline.InsertPipelineBuild(db, projectData, p, app, applicationPipelineParams, params, env, version, trigger)
	if err != nil {
		log.Warning("scheduler.Run> Cannot start pipeline %s: %s\n", pipelineName, err)
		return nil, err
	}

	return &pb, nil
}
