package application

import (
	"fmt"

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

//Import is able to create a new application and all its component
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
	for _, perm := range app.ApplicationGroups {
		//FIX ME:Reload group
		log.Debug("application.Import> Insert group %d in application", perm.Group.ID)
		if err := group.InsertGroupInApplication(db, app.ID, perm.Group.ID, perm.Permission); err != nil {
			return err
		}
		if msgChan != nil {
			msgChan <- msg.New(msg.AppGroupSetPermission, perm.Group.Name, app.Name)
		}
	}

	//Import pipelines
	for _, apip := range app.Pipelines {
		//Import pipeline
		log.Debug("application.Import> Insert pipeline %s", apip.Pipeline.Name)
		if err := pipeline.Import(db, proj, &apip.Pipeline, msgChan); err != nil {
			return err
		}
		//Attach pipeline
		log.Debug("application.Import> Attach pipeline %s", apip.Pipeline.Name)
		if err := AttachPipeline(db, app.ID, apip.Pipeline.ID); err != nil {
			return err
		}
		if msgChan != nil {
			msgChan <- msg.New(msg.PipelineAttached, apip.Pipeline.Name, app.Name)
		}
	}

	//Insert triggers
	for _, apip := range app.Pipelines {
		for _, t := range apip.Triggers {

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

			//Load source environmment
			if t.SrcEnvironment.Name == "" {
				t.SrcEnvironment = sdk.DefaultEnv
			} else {
				env, err := environment.LoadEnvironmentByName(db, proj.Key, t.SrcEnvironment.Name)
				if err != nil {
					return err
				}
				t.SrcEnvironment = *env
			}

			//Load destination environment
			if t.DestEnvironment.Name == "" {
				t.DestEnvironment = sdk.DefaultEnv
			} else {
				env, err := environment.LoadEnvironmentByName(db, proj.Key, t.DestEnvironment.Name)
				if err != nil {
					return err
				}
				t.DestEnvironment = *env
			}

			//Set source pipeline
			t.SrcPipeline = apip.Pipeline

			//Load dest pipeline
			destPipeline, err := pipeline.LoadPipeline(db, proj.Key, t.DestPipeline.Name, false)
			if err != nil {
				return err
			}
			t.DestPipeline = *destPipeline

			//Insert trigger
			if err := trigger.InsertTrigger(db, &t); err != nil {
				return err
			}
			if msgChan != nil {
				msgChan <- msg.New(msg.PipelineTriggerCreated, t.SrcPipeline.Name, t.SrcApplication.Name, t.DestPipeline.Name, t.DestApplication.Name)
			}
		}
	}
	fmt.Println("ici")
	//Set repositories manager
	app.RepositoriesManager = repomanager
	if app.RepositoriesManager != nil && app.RepositoryFullname != "" {
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
