package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) updatePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permPipelineKey"]

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		// check pipeline name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "updatePipelineHandler: Pipeline name %s do not respect pattern", p.Name)
		}

		pipelineDB, err := pipeline.LoadPipeline(api.mustDB(), key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			sdk.WrapError(errB, "updatePipelineHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineDB, pipeline.AuditUpdatePipeline, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		oldName := pipelineDB.Name
		pipelineDB.Name = p.Name
		pipelineDB.Description = p.Description
		pipelineDB.Type = p.Type

		if err := pipeline.UpdatePipeline(tx, pipelineDB); err != nil {
			return sdk.WrapError(err, "cannot update pipeline %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(key, p.Name, oldName, deprecatedGetUser(ctx))

		return service.WriteJSON(w, pipelineDB, http.StatusOK)
	}
}

func (api *API) postPipelineRollbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permPipelineKey"]
		auditID, errConv := strconv.ParseInt(vars["auditID"], 10, 64)
		if errConv != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelineRollbackHandler> cannot convert auditID to int")
		}

		db := api.mustDB()
		u := deprecatedGetUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u)
		if errP != nil {
			return sdk.WrapError(errP, "postPipelineRollbackHandler> Cannot load project")
		}

		audit, errA := pipeline.LoadAuditByID(db, auditID)
		if errA != nil {
			return sdk.WrapError(errA, "postPipelineRollbackHandler> Cannot load audit %d", auditID)
		}

		if err := pipeline.LoadGroupByPipeline(ctx, db, audit.Pipeline); err != nil {
			return sdk.WrapError(err, "cannot load group by pipeline")
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "postPipelineRollbackHandler> cannot begin transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		done := new(sync.WaitGroup)
		done.Add(1)
		msgChan := make(chan sdk.Message)
		msgList := []sdk.Message{}
		go func(array *[]sdk.Message) {
			defer done.Done()
			for m := range msgChan {
				*array = append(*array, m)
			}
		}(&msgList)

		if err := pipeline.ImportUpdate(tx, proj, audit.Pipeline, msgChan, u); err != nil {
			return sdk.WrapError(err, "cannot import pipeline")
		}

		close(msgChan)
		done.Wait()

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(key, audit.Pipeline.Name, name, u)

		return service.WriteJSON(w, *audit.Pipeline, http.StatusOK)
	}
}

func (api *API) addPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errl := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.Default)
		if errl != nil {
			return sdk.WrapError(errl, "AddPipeline: Cannot load %s", key)
		}

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return err
		}

		// check pipeline name pattern
		if regexp := sdk.NamePatternRegex; !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "AddPipeline: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		}

		// Check that pipeline does not already exists
		exist, err := pipeline.ExistPipeline(api.mustDB(), proj.ID, p.Name)
		if err != nil {
			return sdk.WrapError(err, "cannot check if pipeline exist")
		}
		if exist {
			return sdk.WrapError(sdk.ErrConflict, "addPipeline> Pipeline %s already exists", p.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()

		p.ProjectID = proj.ID
		if err := pipeline.InsertPipeline(tx, api.Cache, proj, &p, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert pipeline")
		}

		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "Cannot load groupfrom project")
		}

		for _, g := range proj.ProjectGroups {
			p.GroupPermission = append(p.GroupPermission, g)
		}

		if err := group.InsertGroupsInPipeline(tx, proj.ProjectGroups, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot add groups on pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineAdd(key, p, deprecatedGetUser(ctx))

		p.Permission = permission.PermissionReadWriteExecute

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelineAuditHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]

		audits, err := pipeline.LoadAudit(api.mustDB(), projectKey, pipelineName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline audit")
		}
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		withApp := FormBool(r, "withApplications")
		withWorkflows := FormBool(r, "withWorkflows")
		withEnvironments := FormBool(r, "withEnvironments")

		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, true)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		p.Permission = permission.PipelinePermission(projectKey, p.Name, deprecatedGetUser(ctx))

		if withApp || withWorkflows || withEnvironments {
			p.Usage = &sdk.Usage{}
		}

		if withWorkflows {
			wf, errW := workflow.LoadByPipelineName(api.mustDB(), projectKey, pipelineName)
			if errW != nil {
				return sdk.WrapError(errW, "getPipelineHandler> Cannot load workflows using pipeline %s", p.Name)
			}
			p.Usage.Workflows = wf
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelineTypeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailablePipelineType, http.StatusOK)
	}
}

func (api *API) getPipelinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		project, err := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.Default)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNoProject) {
				log.Warning("getPipelinesHandler: Cannot load %s: %s\n", key, err)
			}
			return err
		}

		pip, err := pipeline.LoadPipelines(api.mustDB(), project.ID, true, deprecatedGetUser(ctx))
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				log.Warning("getPipelinesHandler>Cannot load pipelines: %s\n", err)
			}
			return err
		}

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) deletePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load project")
		}

		p, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		usedW, err := workflow.CountPipeline(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if pipeline is used by a workflow")
		}

		if usedW {
			return sdk.WrapError(sdk.ErrPipelineUsedByWorkflow, "Cannot delete a pipeline used by at least 1 workflow")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeleteAudit(tx, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline audit")
		}

		if err := pipeline.DeletePipeline(tx, p.ID, deprecatedGetUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline %s", pipelineName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineDelete(key, *p, deprecatedGetUser(ctx))
		return nil
	}
}
