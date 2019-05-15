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

		opts := []exportentities.WorkflowOptions{}
		if withPermissions {
			opts = append(opts, exportentities.WorkflowWithPermissions)
		}

		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "Format invalid")
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, getAuthentifiedUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}
		if _, err := workflow.Export(ctx, api.mustDB(), api.Cache, proj, name, f, getAuthentifiedUser(ctx), w, opts...); err != nil {
			return sdk.WithStack(err)
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

		opts := []exportentities.WorkflowOptions{}
		if withPermissions {
			opts = append(opts, exportentities.WorkflowWithPermissions)
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, getAuthentifiedUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		pull, err := workflow.Pull(ctx, api.mustDB(), api.Cache, proj, name, exportentities.FormatYAML, project.EncryptWithBuiltinKey, getAuthentifiedUser(ctx), opts...)
		if err != nil {
			return err
		}

		// early returns as json if param set
		if FormBool(r, "json") {
			return service.WriteJSON(w, pull, http.StatusOK)
		}

		buf := new(bytes.Buffer)
		if err := pull.Tar(buf); err != nil {
			return err
		}

		w.Header().Add("Content-Type", "application/tar")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, buf)
		return sdk.WrapError(err, "unable to copy content buffer in the response writer")
	}
}
