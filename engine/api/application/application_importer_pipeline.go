package application

import (
	"github.com/go-gorp/gorp"
	
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//All of this will be deprecated

//ImportPipelines is able to create pipelines on an existing application
func ImportPipelines(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, u *sdk.User, msgChan chan<- sdk.Message) error {
	//Import pipelines
	for i := range app.Pipelines {
		//Import pipeline
		log.Debug("application.Import> Import pipeline %s", app.Pipelines[i].Pipeline.Name)
		if err := pipeline.Import(db, proj, &app.Pipelines[i].Pipeline, msgChan, u); err != nil {
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
			if _, err := AttachPipeline(db, app.ID, app.Pipelines[i].Pipeline.ID); err != nil {
				return err
			}
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineAttached, app.Pipelines[i].Pipeline.Name, app.Name)
			}
		}
	}

	//Insert triggers
	for i := range app.Pipelines {
		for j := range app.Pipelines[i].Triggers {
			t := &app.Pipelines[i].Triggers[j]

			// You have an existing build pipeline. You want to create a template
			// for create a deploy package, and this template add trigger with only srcApp.
			// so, if SrcApplication.Name != "" -> load existing application.
			if t.SrcApplication.Name == "" {
				//Source application is the current application
				t.SrcApplication = *app
				log.Debug("ImportPipelines> current app")
			} else {
				log.Debug("Load t.SrcApplication.Name:%s", t.SrcApplication.Name)
				srcApp, err := LoadByName(db, store, proj.Key, t.SrcApplication.Name, u, LoadOptions.Default)
				if err != nil {
					return err
				}
				t.SrcApplication = *srcApp
			}

			// Same explanation for pipeline
			if t.SrcPipeline.Name == "" {
				//Source pipeline is the current pipeline
				t.SrcPipeline = app.Pipelines[i].Pipeline
				log.Debug("ImportPipelines> current pipeline")
			} else {
				log.Debug("ImportPipelines> Load t.SrcPipeline.Name:%s", t.SrcApplication.Name)
				srcPipeline, err := pipeline.LoadPipeline(db, proj.Key, t.SrcPipeline.Name, false)
				if err != nil {
					return err
				}
				t.SrcPipeline = *srcPipeline
			}

			//Load destination App
			if t.DestApplication.Name == "" {
				t.DestApplication = *app
			} else {
				dest, err := LoadByName(db, store, proj.Key, t.DestApplication.Name, u, LoadOptions.Default)
				if err != nil {
					return err
				}
				t.DestApplication = *dest
			}

			//Load dest pipeline
			if t.DestPipeline.Name == "" {
				t.DestPipeline = app.Pipelines[i].Pipeline
			} else {
				destPipeline, err := pipeline.LoadPipeline(db, proj.Key, t.DestPipeline.Name, false)
				if err != nil {
					return err
				}
				t.DestPipeline = *destPipeline
			}

			//Load or import source environmment
			if t.SrcEnvironment.Name == "" {
				t.SrcEnvironment = sdk.DefaultEnv
			} else {
				if err := environment.Import(db, proj, &t.SrcEnvironment, msgChan, u); err != nil {
					return sdk.WrapError(err, "ImportPipelines> Cannot import environment %s", t.SrcEnvironment.Name)
				}
			}

			//Load or import destination environment
			if t.DestEnvironment.Name == "" {
				t.DestEnvironment = sdk.DefaultEnv
			} else {
				if err := environment.Import(db, proj, &t.DestEnvironment, msgChan, u); err != nil {
					return sdk.WrapError(err, "ImportPipelines> Cannot import environment %s", t.DestEnvironment.Name)
				}
			}

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
					msgChan <- sdk.NewMessage(sdk.MsgPipelineTriggerCreated, t.SrcPipeline.Name, t.SrcApplication.Name, t.DestPipeline.Name, t.DestApplication.Name)
				}

			}
		}
	}
	return nil
}
