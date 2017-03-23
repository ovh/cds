package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addJobToStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, err := strconv.ParseInt(stageIDString, 10, 64)
	if err != nil {
		log.Warning("addJobToStageHandler> Stage ID must be an int: %s\n", err)
		return sdk.ErrInvalidID

	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("addJobToStageHandler> Cannot load pipeline %s for project %s: %s\n", pipelineName, projectKey, err)
		return sdk.ErrPipelineNotFound

	}

	if err := pipeline.LoadPipelineStage(db, pip); err != nil {
		log.Warning("addJobToStageHandler>Cannot load stages: %s\n", err)
		return err

	}

	// check if stage is in the current pipeline
	found := false
	for _, s := range pip.Stages {
		if s.ID == stageID {
			found = true
			break
		}
	}

	if !found {
		log.Warning("addJobToStageHandler>Stage not found\n")
		return sdk.ErrNotFound

	}

	tx, err := db.Begin()
	if err != nil {
		return err

	}
	defer tx.Rollback()

	//Default value is job enabled
	job.Action.Enabled = true
	job.Enabled = true
	if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot insert job in database: %s\n", err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot update pipeline last modified date: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		return err

	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	if err := pipeline.LoadPipelineStage(db, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot load stages: %s\n", err)
		return err

	}

	return WriteJSON(w, r, pip, http.StatusOK)
}

func updateJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]
	jobIDString := vars["jobID"]

	jobID, err := strconv.ParseInt(jobIDString, 10, 64)
	if err != nil {
		log.Warning("updateJobHandler>ID is not a int: %s\n", err)
		return sdk.ErrInvalidID

	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 64)
	if err != nil {
		log.Warning("updateJobHandler>ID is not a int: %s\n", err)
		return sdk.ErrInvalidID

	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	if jobID != job.PipelineActionID {
		log.Warning("updateJobHandler>Pipeline action does not match: %s\n", err)
		return err

	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updateJobHandler>Cannot load pipeline %s: %s\n", pipName, err)
		return err

	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("updateJobHandler>Cannot load stages: %s\n", err)
		return err

	}

	// check if job is in the current pipeline
	found := false
	for _, s := range pipelineData.Stages {
		if s.ID == stageID {
			for _, j := range s.Jobs {
				if j.PipelineActionID == jobID {
					found = true
					break
				}
			}
		}
	}

	if !found {
		log.Warning("updateJobHandler>Job not found\n")
		return sdk.ErrNotFound

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateJobHandler> Cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := pipeline.UpdateJob(tx, &job, c.User.ID); err != nil {
		log.Warning("updateJobHandler> Cannot update in database: %s\n", err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, pipelineData); err != nil {
		log.Warning("updateJobHandler> Cannot update pipeline last_modified: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateJobHandler> Cannot commit transaction: %s\n", err)
		return err

	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("updateJobHandler> Cannot load stages: %s\n", err)
		return err

	}

	return WriteJSON(w, r, pipelineData, http.StatusOK)
}

func deleteJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	jobIDString := vars["jobID"]

	jobID, err := strconv.ParseInt(jobIDString, 10, 64)
	if err != nil {
		log.Warning("deleteJobHandler>ID is not a int: %s\n", err)
		return sdk.ErrInvalidID

	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("deleteJobHandler>Cannot load pipeline %s: %s\n", pipName, err)
		return err

	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("deleteJobHandler>Cannot load stages: %s\n", err)
		return err

	}

	// check if job is in the current pipeline
	found := false
	var jobToDelete sdk.Job
stageLoop:
	for _, s := range pipelineData.Stages {
		for _, j := range s.Jobs {
			if j.PipelineActionID == jobID {
				jobToDelete = j
				found = true
				break stageLoop
			}
		}
	}

	if !found {
		log.Warning("deleteJobHandler>Job not found\n")
		return sdk.ErrNotFound

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteJobHandler> Cannot begin transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := pipeline.DeleteJob(tx, jobToDelete, c.User.ID); err != nil {
		log.Warning("deleteJobHandler> Cannot delete pipeline action: %s", err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, pipelineData); err != nil {
		log.Warning("deleteJobHandler> Cannot update pipeline last_modified: %s", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteJobHandler> Cannot commit transaction: %s", err)
		return err

	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("deleteJobHandler> Cannot load stages: %s", err)
		return err

	}

	return WriteJSON(w, r, pipelineData, http.StatusOK)

}
