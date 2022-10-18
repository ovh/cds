package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	cdslog "github.com/ovh/cds/sdk/log"
)

// postImportAsCodeHandler
// @title Import workflow as code
// @description This the entrypoint to perform workflow as code. The first step is to post an operation leading to checkout application and scrapping files
// @requestBody {"vcs_Server":"github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
// @responseBody {"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
func (api *API) postImportAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		var ope = new(sdk.Operation)
		if err := service.UnmarshalBody(r, ope); err != nil {
			return err
		}

		if ope.URL == "" {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		if ope.LoadFiles.Pattern == "" {
			ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
		}

		if ope.LoadFiles.Pattern != workflow.WorkflowAsCodePattern {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		p, err := project.Load(ctx, tx, key, project.LoadOptions.WithClearKeys)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, p.Key, ope.VCSServer)
		if err != nil {
			return sdk.NewErrorWithStack(err,
				sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth, "cannot get client for %s %s", key, ope.VCSServer))
		}

		branch, err := repositoriesmanager.DefaultBranch(ctx, client, ope.RepoFullName)
		if err != nil {
			return err
		}
		ope.Setup.Checkout.Branch = branch.DisplayID

		if err := operation.PostRepositoryOperation(ctx, tx, *p, ope, nil); err != nil {
			return sdk.WrapError(err, "cannot create repository operation")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		u := getUserConsumer(ctx)

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("postImportAsCodeHandler-%s", ope.UUID), func(ctx context.Context) {
			ctx = context.WithValue(ctx, cdslog.Operation, ope.UUID)
			ctx = context.WithValue(ctx, cdslog.Repository, ope.RepoFullName)

			ope, err := operation.Poll(ctx, api.mustDB(), ope.UUID)
			if err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "%v", err)
				return
			}

			if ope.Status == sdk.OperationStatusError {
				log.Error(ctx, "operation %s error %+v", ope.UUID, ope.Error)
			}

			ope = &sdk.Operation{
				UUID:         ope.UUID,
				RepoFullName: ope.RepoFullName,
				Date:         ope.Date,
				LoadFiles:    ope.LoadFiles,
				Status:       ope.Status,
				Error:        ope.Error,
			}

			event.PublishOperation(ctx, p.Key, *ope, u)
		})

		return service.WriteJSON(w, sdk.Operation{
			UUID:   ope.UUID,
			Status: ope.Status,
		}, http.StatusCreated)
	}
}

// postPerformImportAsCodeHandler
// @title Perform workflow as code import
// @description This operation push the workflow as code into the project
// @requestBody None
// @responseBody translated message list
func (api *API) postPerformImportAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		uuid := vars["uuid"]

		if isService(ctx) {
			return sdk.ErrForbidden
		}

		if uuid == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given operation uuid")
		}

		proj, err := project.Load(ctx, api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithClearIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		ope, err := operation.GetRepositoryOperation(ctx, api.mustDB(), uuid)
		if err != nil {
			return sdk.WrapError(err, "unable to get repository operation")
		}

		if ope.Status != sdk.OperationStatusDone {
			return sdk.WithStack(sdk.ErrMethodNotAllowed)
		}

		tr, err := workflow.ReadCDSFiles(ope.LoadFiles.Results)
		if err != nil {
			return sdk.WrapError(err, "unable to read cds files")
		}

		//TODO: Delete branch and default branch
		ope.RepositoryStrategy.Branch = "{{.git.branch}}"
		ope.RepositoryStrategy.DefaultBranch = ope.RepositoryInfo.DefaultBranch
		opt := &workflow.PushOption{
			VCSServer:          ope.VCSServer,
			RepositoryName:     ope.RepositoryInfo.Name,
			RepositoryStrategy: ope.RepositoryStrategy,
			Branch:             ope.Setup.Checkout.Branch,
			FromRepository:     ope.RepositoryInfo.FetchURL,
			IsDefaultBranch:    ope.Setup.Checkout.Branch == ope.RepositoryInfo.DefaultBranch,
		}

		data, err := exportentities.UntarWorkflowComponents(ctx, tr)
		if err != nil {
			return err
		}

		consumer := getUserConsumer(ctx)

		mods := []workflowtemplate.TemplateRequestModifierFunc{
			workflowtemplate.TemplateRequestModifiers.DefaultKeys(*proj),
		}
		if !opt.IsDefaultBranch {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
		}
		if opt.FromRepository != "" {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(*proj, opt.FromRepository))
		}
		var allMsg []sdk.Message
		msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), api.Cache, *consumer, *proj, &data, mods...)
		allMsg = append(allMsg, msgTemplate...)
		if err != nil {
			return err
		}
		msgPush, wrkflw, _, _, err := workflow.Push(ctx, api.mustDB(), api.Cache, proj, data, opt, getUserConsumer(ctx), project.DecryptWithBuiltinKey)
		allMsg = append(allMsg, msgPush...)
		if err != nil {
			return sdk.WrapError(err, "unable to push workflow")
		}
		if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wrkflw, *consumer, wti); err != nil {
			return err
		}
		msgListString := translate(allMsg)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Grant CDS as a repository collaborator
		// TODO for this moment, this step is not mandatory. If it's failed, continue the ascode process
		client, erra := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, ope.VCSServer)
		if erra != nil {
			log.Error(ctx, "postPerformImportAsCodeHandler> Cannot get client for %s %s : %s", proj.Key, ope.VCSServer, erra)
		} else {
			if err := client.GrantWritePermission(ctx, ope.RepoFullName); err != nil {
				log.Error(ctx, "postPerformImportAsCodeHandler> Unable to grant CDS a repository %s/%s collaborator : %v", ope.VCSServer, ope.RepoFullName, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		event.PublishWorkflowAdd(ctx, proj.Key, *wrkflw, getUserConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) postWorkflowAsCodeEventsResyncHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		proj, err := project.Load(ctx, api.mustDB(), projectKey,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys,
		)
		if err != nil {
			return err
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, workflowName, workflow.LoadOptions{})
		if err != nil {
			return err
		}

		res, err := ascode.SyncEvents(ctx, api.mustDB(), api.Cache, *proj, *wf, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUser)
		if err != nil {
			return err
		}
		if res.Merged {
			if err := workflow.UpdateFromRepository(api.mustDB(), wf.ID, res.FromRepository); err != nil {
				return err
			}
		}

		return nil
	}
}
