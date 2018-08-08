package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) addJobToStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		pip, errl := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, true)
		if errl != nil {
			return sdk.WrapError(sdk.ErrPipelineNotFound, "addJobToStageHandler> Cannot load pipeline %s for project %s: %s", pipelineName, projectKey, errl)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pip); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler>Cannot load stages")
		}

		// check if stage is in the current pipeline
		var stage sdk.Stage
		for _, s := range pip.Stages {
			if s.ID == stageID {
				stage = s
				break
			}
		}

		if stage.ID == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "addJobToStageHandler>Stage not found")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return errb
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pip, pipeline.AuditAddJob, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot create audit")
		}

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

		if err := worker.ComputeRegistrationNeeds(tx, reqs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot compute registration needs")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pip); err != nil {
			return sdk.WrapError(err, "addJobToStageHandler> Cannot load stages")
		}

		event.PublishPipelineJobAdd(projectKey, pipelineName, stage, job, getUser(ctx))

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) updateJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		pipelineData, errl := pipeline.LoadPipeline(api.mustDB(), key, pipName, true)
		if errl != nil {
			return sdk.WrapError(errl, "updateJobHandler>Cannot load pipeline %s", pipName)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "updateJobHandler>Cannot load stages")
		}

		// check if job is in the current pipeline
		var stage sdk.Stage
		var oldJob sdk.Job
		for _, s := range pipelineData.Stages {
			if s.ID == stageID {
				for _, j := range s.Jobs {
					if j.PipelineActionID == jobID {
						stage = s
						oldJob = j
						break
					}
				}
			}
		}

		if oldJob.PipelineActionID == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "updateJobHandler>Job not found")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditUpdateJob, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot create audit")
		}

		reqs, errlb := action.LoadAllBinaryRequirements(tx)
		if errlb != nil {
			return sdk.WrapError(errlb, "updateJobHandler> cannot load all binary requirements")
		}

		if err := pipeline.UpdateJob(tx, &job, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot update in database")
		}

		if err := worker.ComputeRegistrationNeeds(tx, reqs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot compute registration needs")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "updateJobHandler> Cannot load stages")
		}

		event.PublishPipelineJobUpdate(key, pipName, stage, oldJob, job, getUser(ctx))

		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}

func (api *API) deleteJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipName := vars["permPipelineKey"]
		jobIDString := vars["jobID"]

		jobID, errp := strconv.ParseInt(jobIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteJobHandler>ID is not a int: %s", errp)
		}

		pipelineData, errl := pipeline.LoadPipeline(api.mustDB(), key, pipName, true)
		if errl != nil {
			return sdk.WrapError(errl, "deleteJobHandler>Cannot load pipeline %s", pipName)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "deleteJobHandler>Cannot load stages")
		}

		// check if job is in the current pipeline
		found := false
		var stage sdk.Stage
		var jobToDelete sdk.Job
	stageLoop:
		for _, s := range pipelineData.Stages {
			for _, j := range s.Jobs {
				if j.PipelineActionID == jobID {
					jobToDelete = j
					stage = s
					found = true
					break stageLoop
				}
			}
		}

		if !found {
			return sdk.WrapError(sdk.ErrNotFound, "deleteJobHandler>Job not found")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "deleteJobHandler> Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditDeleteJob, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot create audit")
		}

		if err := pipeline.DeleteJob(tx, jobToDelete, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot delete pipeline action")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "deleteJobHandler> Cannot load stages")
		}

		event.PublishPipelineJobDelete(key, pipName, stage, jobToDelete, getUser(ctx))

		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}
