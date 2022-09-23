package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) updateAsCodePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]

		branch := FormString(r, "branch")
		message := FormString(r, "message")

		if branch == "" || message == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing branch or message data")
		}

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return err
		}
		if err := p.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys)
		if err != nil {
			return err
		}

		pipelineDB, err := pipeline.LoadPipeline(ctx, tx, key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		if pipelineDB.FromRepository == "" {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "current pipeline is not ascode")
		}

		wkHolder, err := workflow.LoadByRepo(ctx, tx, *proj, pipelineDB.FromRepository, workflow.LoadOptions{
			WithTemplate: true,
		})
		if err != nil {
			return err
		}
		if wkHolder.TemplateInstance != nil {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot edit a pipeline that was generated by a template")
		}

		var rootApp *sdk.Application
		if wkHolder.WorkflowData.Node.Context != nil && wkHolder.WorkflowData.Node.Context.ApplicationID != 0 {
			rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(ctx, tx, wkHolder.WorkflowData.Node.Context.ApplicationID)
			if err != nil {
				return err
			}
		}
		if rootApp == nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow %s that hold the pipeline", wkHolder.Name)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, key, rootApp.VCSServer)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "updateAsCodePipelineHandler> cannot get client got %s %s : %v", key, rootApp.VCSServer, err)
		}

		b, err := client.Branch(ctx, rootApp.RepositoryFullname, sdk.VCSBranchFilters{BranchName: branch})
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		if b != nil && b.Default {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot push the the default branch on your git repository")
		}

		u := getAPIConsumer(ctx)

		wpi := exportentities.NewPipelineV1(p)
		wp := exportentities.WorkflowComponents{
			Pipelines: []exportentities.PipelineV1{wpi},
		}

		ope, err := operation.PushOperationUpdate(ctx, tx, api.Cache, *proj, wp, rootApp.VCSServer, rootApp.RepositoryFullname, branch, message, rootApp.RepositoryStrategy, u)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("UpdateAsCodePipelineHandler-%s", ope.UUID), func(ctx context.Context) {
			ed := ascode.EntityData{
				FromRepo:      pipelineDB.FromRepository,
				Type:          ascode.PipelineEvent,
				ID:            pipelineDB.ID,
				Name:          pipelineDB.Name,
				OperationUUID: ope.UUID,
			}
			ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, api.GoRoutines, *proj, *wkHolder, *rootApp, ed, u)
		})

		return service.WriteJSON(w, sdk.Operation{
			UUID:   ope.UUID,
			Status: ope.Status,
		}, http.StatusOK)
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
			return err
		}
		if err := p.IsValid(); err != nil {
			return err
		}

		pipelineDB, err := pipeline.LoadPipeline(ctx, api.mustDB(), key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		if pipelineDB.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
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
			return sdk.WithStack(err)
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

		proj, errP := project.Load(ctx, db, key, project.LoadOptions.WithGroups)
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

		if err := pipeline.ImportUpdate(ctx, tx, *proj, audit.Pipeline, msgChan, pipeline.ImportOptions{}); err != nil {
			return sdk.WrapError(err, "cannot import pipeline")
		}

		close(msgChan)
		done.Wait()

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return err
		}
		if err := p.IsValid(); err != nil {
			return err
		}

		// Check that pipeline does not already exists
		exist, err := pipeline.ExistPipeline(api.mustDB(), proj.ID, p.Name)
		if err != nil {
			return sdk.WrapError(err, "cannot check if pipeline exist")
		}
		if exist {
			return sdk.NewErrorFrom(sdk.ErrPipelineAlreadyExists, "pipeline %s already exists", p.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		p.ProjectID = proj.ID
		if err := pipeline.InsertPipeline(tx, &p); err != nil {
			return sdk.WrapError(err, "cannot insert pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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
		withWorkflows := service.FormBool(r, "withWorkflows")
		withAsCodeEvent := service.FormBool(r, "withAsCodeEvents")

		p, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineName, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", pipelineName)
		}

		if p.FromRepository != "" {
			proj, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
			if err != nil {
				return err
			}

			wkAscodeHolder, err := workflow.LoadByRepo(ctx, api.mustDB(), *proj, p.FromRepository, workflow.LoadOptions{
				WithTemplate: true,
			})
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(err, "cannot found workflow holder of the pipeline")
			}
			p.WorkflowAscodeHolder = wkAscodeHolder

			// FIXME from_repository should never be set if the workflow holder was deleted
			if p.WorkflowAscodeHolder == nil {
				p.FromRepository = ""
			}
		}

		if withAsCodeEvent && p.WorkflowAscodeHolder != nil {
			events, err := ascode.LoadEventsByWorkflowID(ctx, api.mustDB(), p.WorkflowAscodeHolder.ID)
			if err != nil {
				return err
			}
			p.AsCodeEvents = events
		}

		if withWorkflows {
			wf, err := workflow.LoadByPipelineName(ctx, api.mustDB(), projectKey, pipelineName)
			if err != nil {
				return sdk.WrapError(err, "cannot load workflows using pipeline %s", p.Name)
			}
			p.Usage = &sdk.Usage{}
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
		withUsage := service.FormBool(r, "withUsage")
		withoutDetails := service.FormBool(r, "withoutDetails")

		project, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.Default)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNoProject) {
				log.Warn(ctx, "getPipelinesHandler: Cannot load %s: %s\n", key, err)
			}
			return err
		}

		pips, err := pipeline.LoadPipelines(api.mustDB(), project.ID, withoutDetails)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				log.Warn(ctx, "getPipelinesHandler>Cannot load pipelines: %s\n", err)
			}
			return err
		}

		if withUsage {
			for i := range pips {
				p := &pips[i]
				wf, err := workflow.LoadByPipelineName(ctx, api.mustDB(), key, p.Name)
				if err != nil {
					return sdk.WrapError(err, "cannot load workflows using pipeline %s", p.Name)
				}
				p.Usage = &sdk.Usage{}
				p.Usage.Workflows = wf
			}
		}

		return service.WriteJSON(w, pips, http.StatusOK)
	}
}

func (api *API) deletePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]

		proj, errP := project.Load(ctx, api.mustDB(), key)
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
			return sdk.WithStack(err)
		}

		event.PublishPipelineDelete(ctx, key, *p, getAPIConsumer(ctx))
		return nil
	}
}
