package pipeline

import (
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Import insert the pipeline in the project of check if the template is the same as existing
func Import(db database.QueryExecuter, proj *sdk.Project, pip *sdk.Pipeline, msgChan chan<- msg.Message) error {
	//Set projectID and Key in pipeline
	pip.ProjectID = proj.ID
	pip.ProjectKey = proj.Key

	//Check if pipeline exists
	ok, err := ExistPipeline(db, proj.ID, pip.Name)
	if err != nil {
		return err
	}
	if !ok {
		if msgChan != nil {
			msgChan <- msg.New(msg.PipelineCreated, pip.Name)
		}
		return importNew(db, proj, pip)
	}
	//Reload the pipeline
	pip2, err := LoadPipeline(db, proj.Key, pip.Name, false)
	if err != nil {
		return err
	}
	//Be confident: use the pipeline
	*pip = *pip2
	if msgChan != nil {
		msgChan <- msg.New(msg.PipelineExists, pip.Name)
	}
	return nil
}

func importNew(db database.QueryExecuter, proj *sdk.Project, pip *sdk.Pipeline) error {
	log.Debug("pipeline.importNew> Creating pipeline %s", pip.Name)
	//Insert pipeline
	if err := InsertPipeline(db, pip); err != nil {
		return err
	}

	//If no GroupPermission provided, inherit from project
	if pip.GroupPermission == nil {
		pip.GroupPermission = proj.ProjectGroups
	}

	//Insert group permission
	if err := group.InsertGroupsInPipeline(db, pip.GroupPermission, pip.ID); err != nil {
		return err
	}

	//Insert stages
	for i, s := range pip.Stages {
		log.Debug("pipeline.importNew> Creating stage %s on pipeline %s", s.Name, pip.Name)
		//Set default build order
		s.BuildOrder = i
		//Default is enabled
		s.Enabled = true
		//Set relation with pipeline
		s.PipelineID = pip.ID
		//Insert stage
		if err := InsertStage(db, &s); err != nil {
			return err
		}
		//Insert stage's Jobs
		for _, jobAction := range s.Actions {
			job := &sdk.Job{
				PipelineStageID: s.ID,
				Action:          jobAction,
				Enabled:         true,
				LastModified:    time.Now().Unix(),
			}

			log.Debug("pipeline.importNew> Creating job %s on stage %s on pipeline %s", job.Action.Name, s.Name, pip.Name)
			if err := InsertJob(db, job, s.ID, pip); err != nil {
				return err
			}
		}
	}

	return nil
}
