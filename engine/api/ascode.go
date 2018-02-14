package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const workflowAsCodePattern = ".cds/**/*.yml"

// postImportAsCodeHandler
// @title Import workflow as code
// @description This the entrypoint to perform workflow as code. The first step is to post an operation leading to checkout application and scrapping files
// @requestBody {"url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
// @responseBody {"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
func (api *API) postImportAsCodeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		allMsg, err := api.workflowPush(ctx, key, tr)
		if err != nil {
			return err
		}
		msgListString := translate(r, allMsg)

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}
