package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"encoding/json"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/engine/api/archivist"
)

func addJobToStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, err := strconv.ParseInt(stageIDString, 10, 64)
	if err != nil {
		log.Warning("addJobToStageHandler> Stage ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addJobToStageHandler> Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var job sdk.Job
	if err := json.Unmarshal(data, &job); err != nil {
		log.Warning("addJobToStageHandler> Cannot unmarshal body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	proj, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("addJobToStageHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("addJobToStageHandler> Cannot load pipeline %s for project %s: %s\n", pipelineName, projectKey, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	if err := pipeline.LoadPipelineStage(db, pip); err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load stages: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// check if stage is in the current pipeline
	found := false
	for _, s := range pip.Stages {
		if s.ID == stageID {
			found = true
		}
	}

	if !found {
		log.Warning("deletepipelineActionHandler>Stage not found\n")
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot insert job in database: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := pipeline.UpdatePipelineLastModified(tx, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot update pipeline last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//warnings, err := sanity.CheckActionRequirements(tx, proj.Key, pip.Name, a.ID)
	warnings, err := sanity.CheckAction(tx, proj, pip, job.Action.ID)
	if err != nil {
		log.Warning("addJobToStageHandler> Cannot check action %d requirements: %s\n", job.Action.ID, err)
		WriteError(w, r, err)
		return
	}

	if err := sanity.InsertActionWarnings(tx, proj.ID, pip.ID, job.Action.ID, warnings); err != nil {
		log.Warning("addJobToStageHandler> Cannot insert warning for action %d: %s\n", job.Action.ID, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	if err := pipeline.LoadPipelineStage(db, pip); err != nil {
		log.Warning("addJobToStageHandler> Cannot load stages: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pip, http.StatusOK)
}

func updateJobHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]
	jobIDString := vars["jobID"]

	jobID, err := strconv.ParseInt(jobIDString, 10, 64)
	if err != nil {
		log.Warning("updateJobHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 64)
	if err != nil {
		log.Warning("deletepipelineActionHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	var job sdk.Job

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateJobHandler>Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := json.Unmarshal(data, &job); err != nil {
		log.Warning("updateJobHandler>Cannot unmarshal request: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if jobID != job.PipelineActionID {
		log.Warning("updateJobHandler>Pipeline action does not match: %s\n", err)
		WriteError(w, r, err)
		return
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updateJobHandler>Cannot load pipeline %s: %s\n", pipName, err)
		WriteError(w, r, err)
		return
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load stages: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// check if job is in the current pipeline
	found := false
	for _, s := range pipelineData.Stages {
		if s.ID == stageID {
			for _, j := range s.Actions {
				if j.PipelineActionID == jobID {
					found = true
				}
			}
		}
	}

	if !found {
		log.Warning("deletepipelineActionHandler>Job not found\n")
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateJobHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := pipeline.UpdateJob(tx, &job, c.User.ID); err != nil {
		log.Warning("updateJobHandler> Cannot update in database: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData)
	if err != nil {
		log.Warning("updateJobHandler> Cannot update pipeline last_modified: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateJobHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("addJobToStageHandler> Cannot load stages: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pipelineData, http.StatusOK)
}

func deleteJobHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]
	jobIDString := vars["jobID"]

	jobID, err := strconv.ParseInt(jobIDString, 10, 64)
	if err != nil {
		log.Warning("deletepipelineActionHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 64)
	if err != nil {
		log.Warning("deletepipelineActionHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load pipeline %s: %s\n", pipName, err)
		WriteError(w, r, err)
		return
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load stages: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// check if job is in the current pipeline
	found := false
	for _, s := range pipelineData.Stages {
		if s.ID == stageID {
			for _, j := range s.Actions {
				if j.PipelineActionID == jobID {
					found = true
				}
			}
		}
	}

	if !found {
		log.Warning("deletepipelineActionHandler>Job not found\n")
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}



	// Select all pipeline build where given pipelineAction has been run
	query := `SELECT pipeline_build.id FROM pipeline_build
						JOIN action_build ON action_build.pipeline_build_id = pipeline_build.id
						WHERE action_build.pipeline_action_id = $1`
	var ids []int64
	rows, err := db.Query(query, jobID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> cannot retrieves pipeline build: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			rows.Close()
			log.Warning("deletePipelineActionHandler> cannot retrieves pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	log.Notice("deletePipelineActionHandler> Got %d PipelineBuild to archive\n", len(ids))

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot begin transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	// For each pipeline build, archive it to get out of relationnal
	for _, id := range ids {
		err = archivist.ArchiveBuild(tx, id)
		if err != nil {
			log.Warning("deletePipelineActionHandler> cannot archive pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	err = pipeline.DeleteJob(tx, jobID, c.User.ID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot delete pipeline action: %s", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot update pipeline last_modified: %s", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)

}
