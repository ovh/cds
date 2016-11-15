package application

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Import is able to create a new application and all its components
func Import(db database.QueryExecuter, proj *sdk.Project, app *sdk.Application, repomanager *sdk.RepositoriesManager, msgChan chan<- msg.Message) error {
	//Save application in database
	if err := InsertApplication(db, proj, app); err != nil {
		return err
	}

	if msgChan != nil {
		msgChan <- msg.New(msg.AppCreated, app.Name)
	}

	//Inherit project groups if not provided
	if app.ApplicationGroups == nil {
		if msgChan != nil {
			msgChan <- msg.New(msg.AppGroupInheritPermission, app.Name)
		}
		app.ApplicationGroups = proj.ProjectGroups
	}

	//Insert group permission on application
	for i := range app.ApplicationGroups {
		//Load the group by name
		g, err := group.LoadGroup(db, app.ApplicationGroups[i].Group.Name)
		if err != nil {
			return err
		}
		log.Debug("application.Import> Insert group %d in application", g.ID)
		if err := group.InsertGroupInApplication(db, app.ID, g.ID, app.ApplicationGroups[i].Permission); err != nil {
			return err
		}
		if msgChan != nil {
			msgChan <- msg.New(msg.AppGroupSetPermission, g.Name, app.Name)
		}
	}

	if err := ImportPipelines(db, proj, app, msgChan); err != nil {
		return err
	}

	//Set repositories manager
	app.RepositoriesManager = repomanager
	if app.RepositoriesManager != nil && app.RepositoryFullname != "" && len(app.Pipelines) > 0 {
		if err := repositoriesmanager.InsertForApplication(db, app, proj.Key); err != nil {
			return err
		}
		//Manage hook
		if _, err := hook.CreateHook(db, proj.Key, repomanager, app.RepositoryFullname, app, &app.Pipelines[0].Pipeline); err != nil {
			return err
		}
		if msgChan != nil {
			msgChan <- msg.New(msg.HookCreated, app.RepositoryFullname, app.Pipelines[0].Pipeline.Name)
		}
	}

	return nil
}

//ImportPipelines is able to create pipelines on an existing application
func ImportPipelines(db database.QueryExecuter, proj *sdk.Project, app *sdk.Application, msgChan chan<- msg.Message) error {
	//Import pipelines
	for i := range app.Pipelines {
		//Import pipeline
		log.Debug("application.Import> Import pipeline %s", app.Pipelines[i].Pipeline.Name)
		if err := pipeline.Import(db, proj, &app.Pipelines[i].Pipeline, msgChan); err != nil {
			return err
		}

		//Check if application is attached to the pipeline
		attached, err := IsAttached(db, proj.ID, app.ID, app.Pipelines[i].Pipeline.Name)
		if err != nil {
			return err
		}

		//Attach pipeline
		if !attached {
			log.Debug("application.Import> Attach pipeline %s", app.Pipelines[i].Pipeline.Name)
			if err := AttachPipeline(db, app.ID, app.Pipelines[i].Pipeline.ID); err != nil {
				return err
			}
			if msgChan != nil {
				msgChan <- msg.New(msg.PipelineAttached, app.Pipelines[i].Pipeline.Name, app.Name)
			}
		}
	}

	//Insert triggers
	for i := range app.Pipelines {
		for j := range app.Pipelines[i].Triggers {
			t := &app.Pipelines[i].Triggers[j]
			//Source pipeline is always the current pipeline
			t.SrcPipeline = app.Pipelines[i].Pipeline

			//Source application is always the current application
			t.SrcApplication = *app

			//Load destination App
			if t.DestApplication.Name == "" {
				t.DestApplication = *app
			} else {
				dest, err := LoadApplicationByName(db, proj.Key, t.DestApplication.Name)
				if err != nil {
					return err
				}
				t.DestApplication = *dest
			}

			//Load or import source environmment
			if t.SrcEnvironment.Name == "" {
				t.SrcEnvironment = sdk.DefaultEnv
			} else {
				if err := environment.Import(db, proj, &t.SrcEnvironment, msgChan); err != nil {
					return err
				}
			}

			//Load or import destination environment
			if t.DestEnvironment.Name == "" {
				t.DestEnvironment = sdk.DefaultEnv
			} else {
				if err := environment.Import(db, proj, &t.DestEnvironment, msgChan); err != nil {
					return err
				}
			}

			//Load dest pipeline
			destPipeline, err := pipeline.LoadPipeline(db, proj.Key, t.DestPipeline.Name, false)
			if err != nil {
				return err
			}
			t.DestPipeline = *destPipeline

			//Check if environment and pipeline type are compatible
			if t.DestEnvironment.ID == sdk.DefaultEnv.ID && t.DestPipeline.Type == sdk.DeploymentPipeline {
				return sdk.ErrNoEnvironmentProvided
			}

			log.Debug("application.Import> creating trigger SrcApp=%d SrpPip=%d SrcEnv=%d DestApp=%d DestPip=%d DestEnv=%d", t.SrcApplication.ID, t.SrcPipeline.ID, t.SrcEnvironment.ID, t.DestApplication.ID, t.DestPipeline.ID, t.DestEnvironment.ID)

			//Check if trigger exists
			exists, err := trigger.Exists(db, t.SrcApplication.ID, t.SrcPipeline.ID, t.SrcEnvironment.ID, t.DestApplication.ID, t.DestPipeline.ID, t.DestEnvironment.ID)
			if err != nil {
				return err
			}
			if !exists {
				//Insert trigger
				if err := trigger.InsertTrigger(db, t); err != nil {
					return err
				}
				if msgChan != nil {
					msgChan <- msg.New(msg.PipelineTriggerCreated, t.SrcPipeline.Name, t.SrcApplication.Name, t.DestPipeline.Name, t.DestApplication.Name)
				}

			}
		}
	}
	return nil
}
