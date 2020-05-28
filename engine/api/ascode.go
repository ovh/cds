package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

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
	"github.com/ovh/cds/sdk/log"
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

		p, err := project.Load(api.mustDB(), key, project.LoadOptions.WithClearKeys)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, api.mustDB(), p.Key, ope.VCSServer)
		if err != nil {
			return err
		}
		client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, p.Key, vcsServer)
		if err != nil {
			return sdk.NewErrorWithStack(err,
				sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth, "cannot get client for %s %s", key, ope.VCSServer))
		}

		branches, err := client.Branches(ctx, ope.RepoFullName)
		if err != nil {
			return sdk.WrapError(err, "cannot list branches for %s/%s", ope.VCSServer, ope.RepoFullName)
		}
		for _, b := range branches {
			if b.Default {
				ope.Setup.Checkout.Branch = b.DisplayID
				break
			}
		}

		if err := operation.PostRepositoryOperation(ctx, api.mustDB(), *p, ope, nil); err != nil {
			return sdk.WrapError(err, "cannot create repository operation")
		}
		ope.RepositoryStrategy.SSHKeyContent = sdk.PasswordPlaceholder
		ope.RepositoryStrategy.Password = sdk.PasswordPlaceholder

		return service.WriteJSON(w, ope, http.StatusCreated)
	}
}

// getImportAsCodeHandler
// @title Get import workflow as code operation details
// @description This route helps you to know if a "import as code" is over, and the details of the performed operation
// @requestBody None
// @responseBody  {"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"","ssh_key":"","user":"","password":"","branch":"","default_branch":"","pgp_key":""},"setup":{"checkout":{}},"load_files":{"pattern":".cds/**/*.yml","results":{"w-go-repo.yml":"bmFtZTogdy1nby1yZXBvCgkJCQkJdmVyc2lvbjogdjEuMAoJCQkJCXBpcGVsaW5lOiBidWlsZAoJCQkJCWFwcGxpY2F0aW9uOiBnby1yZXBvCgkJCQkJcGlwZWxpbmVfaG9va3M6CgkJCQkJLSB0eXBlOiBSZXBvc2l0b3J5V2ViSG9vawoJCQkJCQ=="}},"status":2}
func (api *API) getImportAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		ope, err := operation.GetRepositoryOperation(ctx, api.mustDB(), vars["uuid"])
		if err != nil {
			return sdk.WrapError(err, "cannot get repository operation status")
		}
		ope.RepositoryStrategy.SSHKeyContent = sdk.PasswordPlaceholder
		ope.RepositoryStrategy.Password = sdk.PasswordPlaceholder
		return service.WriteJSON(w, ope, http.StatusOK)
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

		//Load project
		proj, errp := project.Load(api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithFeatures(api.Cache),
			project.LoadOptions.WithClearIntegrations,
		)
		if errp != nil {
			return sdk.WrapError(errp, "postPerformImportAsCodeHandler> Cannot load project %s", key)
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

		consumer := getAPIConsumer(ctx)

		mods := []workflowtemplate.TemplateRequestModifierFunc{
			workflowtemplate.TemplateRequestModifiers.DefaultKeys(*proj),
		}
		if !opt.IsDefaultBranch {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
		}
		if opt.FromRepository != "" {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(ctx, api.mustDB(), api.Cache, *proj, opt.FromRepository))
		}
		var allMsg []sdk.Message
		msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), *consumer, *proj, &data, mods...)
		allMsg = append(allMsg, msgTemplate...)
		if err != nil {
			return err
		}
		msgPush, wrkflw, _, _, err := workflow.Push(ctx, api.mustDB(), api.Cache, proj, data, opt, getAPIConsumer(ctx), project.DecryptWithBuiltinKey)
		allMsg = append(allMsg, msgPush...)
		if err != nil {
			return sdk.WrapError(err, "unable to push workflow")
		}
		if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wrkflw, *consumer, wti); err != nil {
			return err
		}
		msgListString := translate(r, allMsg)

		// Grant CDS as a repository collaborator
		// TODO for this moment, this step is not mandatory. If it's failed, continue the ascode process
		vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, api.mustDB(), proj.Key, ope.VCSServer)
		if err != nil {
			return err
		}
		client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, proj.Key, vcsServer)
		if erra != nil {
			log.Error(ctx, "postPerformImportAsCodeHandler> Cannot get client for %s %s : %s", proj.Key, ope.VCSServer, erra)
		} else {
			if err := client.GrantWritePermission(ctx, ope.RepoFullName); err != nil {
				log.Error(ctx, "postPerformImportAsCodeHandler> Unable to grant CDS a repository %s/%s collaborator : %v", ope.VCSServer, ope.RepoFullName, err)
			}
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		event.PublishWorkflowAdd(ctx, proj.Key, *wrkflw, getAPIConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) postWorkflowAsCodeEventsResyncHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		proj, err := project.Load(api.mustDB(), projectKey,
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

		res, err := ascode.SyncEvents(ctx, api.mustDB(), api.Cache, *proj, *wf, getAPIConsumer(ctx).AuthentifiedUser)
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
