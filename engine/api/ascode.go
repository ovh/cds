package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const workflowAsCodePattern = ".cds/**/*.yml"

// postImportAsCodeHandler
// @title Import workflow as code
// @description This the entrypoint to perform workflow as code. The first step is to post an operation leading to checkout application and scrapping files
// @requestBody {"vcs_Server":"github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
// @responseBody {"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
func (api *API) postImportAsCodeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		var ope = new(sdk.Operation)
		if err := UnmarshalBody(r, ope); err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler")
		}

		if ope.URL == "" {
			return sdk.ErrWrongRequest
		}

		if ope.LoadFiles.Pattern == "" {
			ope.LoadFiles.Pattern = workflowAsCodePattern
		}

		if ope.LoadFiles.Pattern != workflowAsCodePattern {
			return sdk.ErrWrongRequest
		}

		querier := services.Querier(api.mustDB(), api.Cache)

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			sdk.WrapError(errP, "postImportAsCodeHandler> Cannot load project")
		}
		vcsServer := repositoriesmanager.GetProjectVCSServer(p, ope.VCSServer)
		client, erra := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, vcsServer)
		if erra != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "postImportAsCodeHandler> Cannot get client got %s %s : %s", key, ope.VCSServer, erra)
		}
		branches, errB := client.Branches(ope.RepoFullName)
		if errB != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "postImportAsCodeHandler> Cannot list branches for %s/%s", ope.VCSServer, ope.RepoFullName)
		}
		for _, b := range branches {
			if b.Default {
				ope.Setup.Checkout.Branch = b.DisplayID
				break
			}
		}

		srvs, err := querier.FindByType(services.TypeRepositories)
		if err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to found repositories service")
		}

		if _, err := services.DoJSONRequest(srvs, http.MethodPost, "/operations", ope, ope); err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to perform operation")
		}

		return WriteJSON(w, ope, http.StatusCreated)
	}
}

// getImportAsCodeHandler
// @title Get import workflow as code operation details
// @description This route helps you to know if a "import as code" is over, and the details of the performed operation
// @requestBody None
// @responseBody  {"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"","ssh_key":"","user":"","password":"","branch":"","default_branch":"","pgp_key":""},"setup":{"checkout":{}},"load_files":{"pattern":".cds/**/*.yml","results":{"w-go-repo.yml":"bmFtZTogdy1nby1yZXBvCgkJCQkJdmVyc2lvbjogdjEuMAoJCQkJCXBpcGVsaW5lOiBidWlsZAoJCQkJCWFwcGxpY2F0aW9uOiBnby1yZXBvCgkJCQkJcGlwZWxpbmVfaG9va3M6CgkJCQkJLSB0eXBlOiBSZXBvc2l0b3J5V2ViSG9vawoJCQkJCQ=="}},"status":2}
func (api *API) getImportAsCodeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		var ope = new(sdk.Operation)

		querier := services.Querier(api.mustDB(), api.Cache)
		srvs, err := querier.FindByType(services.TypeRepositories)
		if err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to found repositories service")
		}

		if _, err := services.DoJSONRequest(srvs, http.MethodGet, "/operations/"+uuid, nil, ope); err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to get operation")
		}
		return WriteJSON(w, ope, http.StatusOK)
	}
}

// postPerformImportAsCodeHandler
// @title Perform workflow as code import
// @description This operation push the workflow as code into the project
// @requestBody None
// @responseBody translated message list
func (api *API) postPerformImportAsCodeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		uuid := vars["uuid"]

		var ope = new(sdk.Operation)

		querier := services.Querier(api.mustDB(), api.Cache)
		srvs, err := querier.FindByType(services.TypeRepositories)
		if err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to found repositories service")
		}

		if _, err := services.DoJSONRequest(srvs, http.MethodGet, "/operations/"+uuid, nil, ope); err != nil {
			return sdk.WrapError(err, "postImportAsCodeHandler> Unable to get operation")
		}

		if ope.Status != sdk.OperationStatusDone {
			return sdk.ErrMethodNotAllowed
		}

		// Create a buffer to write our archive to.
		buf := new(bytes.Buffer)
		// Create a new tar archive.
		tw := tar.NewWriter(buf)
		// Add some files to the archive.
		for fname, fcontent := range ope.LoadFiles.Results {
			log.Debug("postImportAsCodeHandler> Reading %s", fname)
			hdr := &tar.Header{
				Name: filepath.Base(fname),
				Mode: 0600,
				Size: int64(len(fcontent)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if n, err := tw.Write(fcontent); err != nil {
				return err
			} else if n == 0 {
				return fmt.Errorf("nothing to write")
			}
		}
		// Make sure to check the error on Close.
		if err := tw.Close(); err != nil {
			return err
		}

		tr := tar.NewReader(buf)
		opt := &workflowPushOption{
			VCSServer:          ope.VCSServer,
			RepositoryName:     ope.RepositoryInfo.Name,
			RepositoryStrategy: ope.RepositoryStrategy,
			Branch:             ope.Setup.Checkout.Branch,
			FromRepository:     ope.RepositoryInfo.FetchURL,
			IsDefaultBranch:    ope.Setup.Checkout.Branch == ope.RepositoryInfo.DefaultBranch,
		}
		allMsg, wrkflw, err := api.workflowPush(ctx, key, tr, opt)
		if err != nil {
			return err
		}
		msgListString := translate(r, allMsg)

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}
