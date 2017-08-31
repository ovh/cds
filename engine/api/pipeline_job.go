package api

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func addJobToStageHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		stageIDString := vars["stageID"]

		stageID, errp := strconv.ParseInt(stageIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "addJobToStageHandler> Stage ID must be an int: %s", errp)
		}

		var job sdk.Job
		if err := UnmarshalBody(r, &job); err != nil {
			return err
		}

		pip, errl := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
		if errl != nil {
			return sdk.WrapError(sdk.ErrPipelineNotFound, "addJobToStageHandler> Cannot load pipeline %s for project %s: %s", pipelineName, projectKey, errl)
		}

		if err := pipeline.LoadPipelineStage(db, pip); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler>Cannot load stages")
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
			return sdk.WrapError(sdk.ErrNotFound, "addJobToStageHandler>Stage not found")
		}

		tx, errb := db.Begin()
		if errb != nil {
			return errb
		}
		defer tx.Rollback()

		reqs, errlb := action.LoadAllBinaryRequirements(tx)
		if errlb != nil {
			return sdk.WrapError(errlb, "addJobToStageHandler> cannot load all binary requirements")
		}

		//Default value is job enabled
		job.Action.Enabled = true
		job.Enabled = true
		if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot insert job in database")
		}

		proj, errproj := project.Load(db, projectKey, c.User)
		if errproj != nil {
			return sdk.WrapError(errproj, "addJobToStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pip, c.User); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot update pipeline last modified date")
		}

		if err := worker.ComputeRegistrationNeeds(tx, reqs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot compute registration needs")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		cache.DeleteAll(cache.Key("application", projectKey, "*"))
		cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

		if err := pipeline.LoadPipelineStage(db, pip); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot load stages")
		}

		return WriteJSON(w, r, pip, http.StatusOK)
	}
}

func updateJobHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipName := vars["permPipelineKey"]
		stageIDString := vars["stageID"]
		jobIDString := vars["jobID"]

		jobID, errp := strconv.ParseInt(jobIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "updateJobHandler>ID is not a int: %s", errp)
		}

		stageID, errps := strconv.ParseInt(stageIDString, 10, 64)
		if errps != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "updateJobHandler>ID is not a int: %s", errps)
		}

		var job sdk.Job
		if err := UnmarshalBody(r, &job); err != nil {
			return err
		}

		if jobID != job.PipelineActionID {
			return sdk.WrapError(sdk.ErrInvalidID, "updateJobHandler>Pipeline action does not match")
		}

		pipelineData, errl := pipeline.LoadPipeline(db, key, pipName, false)
		if errl != nil {
			return sdk.WrapError(errl, "updateJobHandler>Cannot load pipeline %s", pipName)
		}

		if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
			return sdk.WrapError(err, "updateJobHandler>Cannot load stages")
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
			return sdk.WrapError(sdk.ErrNotFound, "updateJobHandler>Job not found")
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		reqs, errlb := action.LoadAllBinaryRequirements(tx)
		if errlb != nil {
			return sdk.WrapError(errlb, "updateJobHandler> cannot load all binary requirements")
		}

		if err := pipeline.UpdateJob(tx, &job, c.User.ID); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot update in database")
		}

		if err := worker.ComputeRegistrationNeeds(tx, reqs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot compute registration needs")
		}

		proj, errproj := project.Load(db, key, c.User)
		if errproj != nil {
			return sdk.WrapError(errproj, "addJobToStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, c.User); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot update pipeline last_modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot load stages")
		}

		return WriteJSON(w, r, pipelineData, http.StatusOK)
	}
}

func deleteJobHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipName := vars["permPipelineKey"]
		jobIDString := vars["jobID"]

		jobID, errp := strconv.ParseInt(jobIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteJobHandler>ID is not a int: %s", errp)
		}

		pipelineData, errl := pipeline.LoadPipeline(db, key, pipName, false)
		if errl != nil {
			return sdk.WrapError(errl, "deleteJobHandler>Cannot load pipeline %s", pipName)
		}

		if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
			return sdk.WrapError(err, "deleteJobHandler>Cannot load stages")
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
			return sdk.WrapError(sdk.ErrNotFound, "deleteJobHandler>Job not found")
		}

		tx, errb := db.Begin()
		if errb != nil {
			return sdk.WrapError(errb, "deleteJobHandler> Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeleteJob(tx, jobToDelete, c.User.ID); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot delete pipeline action")
		}

		proj, errproj := project.Load(db, key, c.User)
		if errproj != nil {
			return sdk.WrapError(errproj, "deleteJobHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, c.User); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot update pipeline last_modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot commit transaction")
		}

		k := cache.Key("application", key, "*")
		cache.DeleteAll(k)

		if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot load stages")
		}

		return WriteJSON(w, r, pipelineData, http.StatusOK)
	}
}
