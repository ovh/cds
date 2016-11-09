package pipeline

import (
	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database"
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
	//Be confident: use the pipeline
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
		for _, job := range s.Actions {
			log.Debug("pipeline.importNew> Creating job %s on stage %s on pipeline %s", job.Name, s.Name, pip.Name)
			//Default is job enabled
			job.Enabled = true
			//No parameter
			job.Parameters = []sdk.Parameter{}
			//Set relation with stage
			job.PipelineStageID = s.ID
			//Insert Actions type Joined = Job
			job.Type = sdk.JoinedAction
			if err := action.InsertAction(db, &job, false); err != nil {
				return err
			}
			log.Debug("pipeline.importNew> Linking job %s to stage %s on pipeline %s", job.Name, s.Name, pip.Name)
			//Then insert Job
			if err := InsertPipelineJob(db, pip, &s, &job); err != nil {
				return err
			}
		}
	}

	return nil
}
