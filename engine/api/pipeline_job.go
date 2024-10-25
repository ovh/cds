package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) addJobToStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]
		stageIDString := vars["stageID"]

		stageID, errp := strconv.ParseInt(stageIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "addJobToStageHandler> Stage ID must be an int: %s", errp)
		}

		var job sdk.Job
		if err := service.UnmarshalBody(r, &job); err != nil {
			return err
		}

		// check that actions used by job are valid
		if err := job.IsValid(); err != nil {
			return err
		}

		pip, errl := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineName, true)
		if errl != nil {
			return sdk.WrapError(sdk.ErrPipelineNotFound, "addJobToStageHandler> Cannot load pipeline %s for project %s: %s", pipelineName, projectKey, errl)
		}
		if pip.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pip); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
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
		defer tx.Rollback() // nolint

		// check that action used by job can be used by pipeline's project
		project, err := project.Load(ctx, tx, pip.ProjectKey, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WithStack(err)
		}
		groupIDs := make([]int64, 0, len(project.ProjectGroups)+1)
		groupIDs = append(groupIDs, group.SharedInfraGroup.ID)
		for i := range project.ProjectGroups {
			groupIDs = append(groupIDs, project.ProjectGroups[i].Group.ID)
		}
		if err := action.CheckChildrenForGroupIDs(ctx, tx, &job.Action, groupIDs); err != nil {
			return err
		}

		if err := pipeline.CreateAudit(tx, pip, pipeline.AuditAddJob, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "cannot create audit")
		}

		rs, errlb := action.GetRequirementsDistinctBinary(tx)
		if errlb != nil {
			return sdk.WrapError(errlb, "cannot load all binary requirements")
		}

		//Default value is job enabled
		job.Action.Enabled = true
		job.Enabled = true
		if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
			return sdk.WrapError(err, "cannot insert job in database")
		}

		if err := workermodel.ComputeRegistrationNeeds(tx, rs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "cannot compute registration needs")
		}

		if err := pipeline.UpdatePipeline(tx, pip); err != nil {
			return sdk.WrapError(err, "cannot update pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pip); err != nil {
			return sdk.WrapError(err, "cannot load stages")
		}

		event.PublishPipelineJobAdd(ctx, projectKey, pipelineName, stage, job, getUserConsumer(ctx))

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) updateJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		pipName := vars["pipelineKey"]

		jobID, err := requestVarInt(r, "jobID")
		if err != nil {
			return err
		}

		stageID, err := requestVarInt(r, "stageID")
		if err != nil {
			return err
		}

		var job sdk.Job
		if err := service.UnmarshalBody(r, &job); err != nil {
			return err
		}

		// Set ActionID (aka child_id in action_edge) for ascode action
		asCodeAction, err := action.LoadAllByTypes(ctx, api.mustDB(), []string{sdk.AsCodeAction})
		if err != nil {
			return err
		}
		if len(asCodeAction) != 1 {
			return sdk.WrapError(sdk.ErrUnknownError, "missing ascode action type")
		}

		for i := range job.Action.Actions {
			a := &job.Action.Actions[i]
			if a.Type == sdk.AsCodeAction {
				a.ID = asCodeAction[0].ID
			}
		}

		// check that actions used by job are valid
		if err := job.IsValid(); err != nil {
			return err
		}
		if jobID != job.PipelineActionID {
			return sdk.WrapError(sdk.ErrInvalidID, "pipeline action does not match")
		}

		// load old pipeline
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), key, pipName, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", pipName)
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "cannot load pipeline stages")
		}

		// check that given job/stage exists in pipeline.
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
			return sdk.WrapError(sdk.ErrNotFound, "job not found in pipeline")
		}

		rx := sdk.NamePatternSpaceRegex
		// stage name mandatory if there are many stages
		if len(pipelineData.Stages) > 1 && !rx.MatchString(stage.Name) {
			return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid stage name '%s'. It should match %s", stage.Name, sdk.NamePatternSpace))
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		// check that action used by job can be used by pipeline's project
		project, err := project.Load(ctx, tx, pipelineData.ProjectKey, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WithStack(err)
		}
		groupIDs := make([]int64, 0, len(project.ProjectGroups)+1)
		groupIDs = append(groupIDs, group.SharedInfraGroup.ID)
		for i := range project.ProjectGroups {
			groupIDs = append(groupIDs, project.ProjectGroups[i].Group.ID)
		}
		if err := action.CheckChildrenForGroupIDs(ctx, tx, &job.Action, groupIDs); err != nil {
			return err
		}

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditUpdateJob, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "cannot create audit")
		}

		rs, errlb := action.GetRequirementsDistinctBinary(tx)
		if errlb != nil {
			return sdk.WrapError(errlb, "cannot load all binary requirements")
		}

		if err := pipeline.UpdateJob(ctx, tx, &job); err != nil {
			return sdk.WrapError(err, "cannot update pipeline job in database")
		}

		if err := workermodel.ComputeRegistrationNeeds(tx, rs, job.Action.Requirements); err != nil {
			return sdk.WrapError(err, "cannot compute registration needs")
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "cannot update pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "cannot load stages")
		}

		event.PublishPipelineJobUpdate(ctx, key, pipName, stage, oldJob, job, getUserConsumer(ctx))

		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}

func (api *API) deleteJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		pipName := vars["pipelineKey"]
		jobIDString := vars["jobID"]

		jobID, errp := strconv.ParseInt(jobIDString, 10, 64)
		if errp != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteJobHandler>ID is not a int: %s", errp)
		}

		pipelineData, errl := pipeline.LoadPipeline(ctx, api.mustDB(), key, pipName, true)
		if errl != nil {
			return sdk.WrapError(errl, "deleteJobHandler>Cannot load pipeline %s", pipName)
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
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
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditDeleteJob, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		if err := pipeline.DeleteJob(tx, jobToDelete); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline action")
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot update pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
		}

		event.PublishPipelineJobDelete(ctx, key, pipName, stage, jobToDelete, getUserConsumer(ctx))

		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}
