package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) updateAsCodePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]
		branch := FormString(r, "branch")
		message := FormString(r, "message")
		fromRepo := FormString(r, "repo")

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		// check pipeline name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "updateAsCodePipelineHandler: Pipeline name %s do not respect pattern", p.Name)
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithClearKeys)
		if err != nil {
			return err
		}

		pipelineDB, err := pipeline.LoadPipeline(ctx, api.mustDB(), key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		if pipelineDB.FromRepository == "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		apps, err := application.LoadAsCode(api.mustDB(), api.Cache, key, fromRepo)
		if err != nil {
			return err
		}

		u := getAPIConsumer(ctx)

		ope, err := pipeline.UpdatePipelineAsCode(ctx, api.Cache, api.mustDB(), proj, p, branch, message, &apps[0], u)
		if err != nil {
			return err
		}

		sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateAsCodePipelineHandler-%s", ope.UUID), func(ctx context.Context) {
			ed := ascode.EntityData{
				FromRepo:  pipelineDB.FromRepository,
				Type:      ascode.AsCodePipeline,
				ID:        pipelineDB.ID,
				Name:      pipelineDB.Name,
				Operation: ope,
			}
			asCodeEvent := ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, proj, &apps[0], ed, u)
			if asCodeEvent != nil {
				event.PublishAsCodeEvent(ctx, proj.Key, *asCodeEvent, u)
			}
		}, api.PanicDump())

		return service.WriteJSON(w, ope, http.StatusOK)
	}
}
func (api *API) updatePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		// check pipeline name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "updatePipelineHandler: Pipeline name %s do not respect pattern", p.Name)
		}

		pipelineDB, err := pipeline.LoadPipeline(ctx, api.mustDB(), key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		if pipelineDB.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			sdk.WrapError(errB, "updatePipelineHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineDB, pipeline.AuditUpdatePipeline, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		oldName := pipelineDB.Name
		pipelineDB.Name = p.Name
		pipelineDB.Description = p.Description

		if err := pipeline.UpdatePipeline(tx, pipelineDB); err != nil {
			return sdk.WrapError(err, "cannot update pipeline %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(ctx, key, p.Name, oldName, getAPIConsumer(ctx))

		return service.WriteJSON(w, pipelineDB, http.StatusOK)
	}
}

func (api *API) postPipelineRollbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]
		auditID, errConv := strconv.ParseInt(vars["auditID"], 10, 64)
		if errConv != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelineRollbackHandler> cannot convert auditID to int")
		}

		db := api.mustDB()
		u := getAPIConsumer(ctx)

		pipDB, err := pipeline.LoadPipeline(ctx, db, key, name, false)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline")
		}
		if pipDB.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		proj, errP := project.Load(db, api.Cache, key, project.LoadOptions.WithGroups)
		if errP != nil {
			return sdk.WrapError(errP, "postPipelineRollbackHandler> Cannot load project")
		}

		audit, errA := pipeline.LoadAuditByID(db, auditID)
		if errA != nil {
			return sdk.WrapError(errA, "postPipelineRollbackHandler> Cannot load audit %d", auditID)
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

		if err := pipeline.ImportUpdate(ctx, tx, proj, audit.Pipeline, msgChan, u); err != nil {
			return sdk.WrapError(err, "cannot import pipeline")
		}

		close(msgChan)
		done.Wait()

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(ctx, key, audit.Pipeline.Name, name, u)

		return service.WriteJSON(w, *audit.Pipeline, http.StatusOK)
	}
}

func (api *API) addPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, errl := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.Default)
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
		defer tx.Rollback() // nolint

		p.ProjectID = proj.ID
		if err := pipeline.InsertPipeline(tx, api.Cache, proj, &p); err != nil {
			return sdk.WrapError(err, "Cannot insert pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineAdd(ctx, key, p, getAPIConsumer(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelineAuditHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]

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
		projectKey := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]
		withApp := FormBool(r, "withApplications")
		withWorkflows := FormBool(r, "withWorkflows")
		withEnvironments := FormBool(r, "withEnvironments")
		withAsCodeEvent := FormBool(r, "withAsCodeEvents")

		p, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineName, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", pipelineName)
		}

		if withApp || withWorkflows || withEnvironments {
			p.Usage = &sdk.Usage{}
		}

		if withAsCodeEvent {
			events, errE := ascode.LoadAsCodeEventByRepo(ctx, api.mustDB(), p.FromRepository)
			if errE != nil {
				return errE
			}
			p.AsCodeEvents = events
		}

		if withWorkflows {
			wf, errW := workflow.LoadByPipelineName(ctx, api.mustDB(), projectKey, pipelineName)
			if errW != nil {
				return sdk.WrapError(errW, "getPipelineHandler> Cannot load workflows using pipeline %s", p.Name)
			}
			p.Usage.Workflows = wf
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		project, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.Default)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNoProject) {
				log.Warning(ctx, "getPipelinesHandler: Cannot load %s: %s\n", key, err)
			}
			return err
		}

		pip, err := pipeline.LoadPipelines(api.mustDB(), project.ID, true)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				log.Warning(ctx, "getPipelinesHandler>Cannot load pipelines: %s\n", err)
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
		key := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load project")
		}

		p, err := pipeline.LoadPipeline(ctx, api.mustDB(), proj.Key, pipelineName, false)
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
		defer tx.Rollback() // nolint

		if err := pipeline.DeleteAudit(tx, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline audit")
		}

		if err := pipeline.DeletePipeline(ctx, tx, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline %s", pipelineName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineDelete(ctx, key, *p, getAPIConsumer(ctx))
		return nil
	}
}
