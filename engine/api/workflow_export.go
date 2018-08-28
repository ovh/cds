package api

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getWorkflowExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		withPermissions := FormBool(r, "withPermissions")

		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowExportHandler> Format invalid")
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowExportHandler> unable to load projet")
		}
		if _, err := workflow.Export(ctx, api.mustDB(), api.Cache, proj, name, f, withPermissions, getUser(ctx), w); err != nil {
			return sdk.WrapError(err, "getWorkflowExportHandler>")
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		return nil
	}
}

//Pull is only in yaml
func (api *API) getWorkflowPullHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		withPermissions := FormBool(r, "withPermissions")

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowPullHandler> unable to load projet")
		}

		buf := new(bytes.Buffer)
		if err := workflow.Pull(ctx, api.mustDB(), api.Cache, proj, name, exportentities.FormatYAML, withPermissions, project.EncryptWithBuiltinKey, getUser(ctx), buf); err != nil {
			return sdk.WrapError(err, "getWorkflowPullHandler")
		}

		w.Header().Add("Content-Type", "application/tar")
		w.WriteHeader(http.StatusOK)
		_, errC := io.Copy(w, buf)
		return sdk.WrapError(errC, "getWorkflowPullHandler> Unable to copy content buffer in the response writer")
	}
}
