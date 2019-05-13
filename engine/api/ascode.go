package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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
			return sdk.WrapError(err, "postImportAsCodeHandler")
		}

		if ope.URL == "" {
			return sdk.ErrWrongRequest
		}

		if ope.LoadFiles.Pattern == "" {
			ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
		}

		if ope.LoadFiles.Pattern != workflow.WorkflowAsCodePattern {
			return sdk.ErrWrongRequest
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithFeatures, project.LoadOptions.WithClearKeys)
		if errP != nil {
			sdk.WrapError(errP, "postImportAsCodeHandler> Cannot load project")
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(p, ope.VCSServer)
		client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
		if erra != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "postImportAsCodeHandler> Cannot get client for %s %s : %s", key, ope.VCSServer, erra)
		}

		branches, errB := client.Branches(ctx, ope.RepoFullName)
		if errB != nil {
			return sdk.WrapError(errB, "postImportAsCodeHandler> Cannot list branches for %s/%s", ope.VCSServer, ope.RepoFullName)
		}
		for _, b := range branches {
			if b.Default {
				ope.Setup.Checkout.Branch = b.DisplayID
				break
			}
		}

		if err := workflow.PostRepositoryOperation(ctx, api.mustDB(), *p, ope, nil); err != nil {
			return sdk.WrapError(err, "Cannot create repository operation")
		}
		ope.RepositoryStrategy.SSHKeyContent = ""

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
		uuid := vars["uuid"]

		var ope = new(sdk.Operation)
		ope.UUID = uuid
		if err := workflow.GetRepositoryOperation(ctx, api.mustDB(), ope); err != nil {
			return sdk.WrapError(err, "Cannot get repository operation status")
		}
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
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithFeatures,
			project.LoadOptions.WithClearIntegrations,
		)
		if errp != nil {
			return sdk.WrapError(errp, "postPerformImportAsCodeHandler> Cannot load project %s", key)
		}

		var ope = new(sdk.Operation)
		ope.UUID = uuid

		if err := workflow.GetRepositoryOperation(ctx, api.mustDB(), ope); err != nil {
			return sdk.WrapError(err, "Unable to get repository operation")
		}

		if ope.Status != sdk.OperationStatusDone {
			return sdk.ErrMethodNotAllowed
		}

		tr, err := workflow.ReadCDSFiles(ope.LoadFiles.Results)
		if err != nil {
			return sdk.WrapError(err, "Unable to read cds files")
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

		allMsg, wrkflw, err := workflow.Push(ctx, api.mustDB(), api.Cache, proj, tr, opt, deprecatedGetUser(ctx), project.DecryptWithBuiltinKey)
		if err != nil {
			return sdk.WrapError(err, "Unable to push workflow")
		}
		msgListString := translate(r, allMsg)

		// Grant CDS as a repository collaborator
		// TODO for this moment, this step is not mandatory. If it's failed, continue the ascode process
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, ope.VCSServer)
		client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
		if erra != nil {
			log.Error("postPerformImportAsCodeHandler> Cannot get client for %s %s : %s", proj.Key, ope.VCSServer, erra)
		} else {
			if err := client.GrantWritePermission(ctx, ope.RepoFullName); err != nil {
				log.Error("postPerformImportAsCodeHandler> Unable to grant CDS a repository %s/%s collaborator : %v", ope.VCSServer, ope.RepoFullName, err)
			}
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
