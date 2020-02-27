package api

import (
	"bytes"
	"context"
	"io"
	"net/http"

	v2 "github.com/ovh/cds/sdk/exportentities/v2"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

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
		withPermissions := FormBool(r, "withPermissions")

		opts := make([]v2.ExportOptions, 0)
		if withPermissions {
			opts = append(opts, v2.WorkflowWithPermissions)
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}
		wk, err := workflow.Export(ctx, api.mustDB(), api.Cache, *proj, name, opts...)
		if err != nil {
			return sdk.WithStack(err)
		}
		f, err := yaml.Marshal(wk)
		if err != nil {
			return sdk.WithStack(err)
		}
		if _, err := w.Write(f); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", string(exportentities.FormatYAML))
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

		opts := make([]v2.ExportOptions, 0)
		if withPermissions {
			opts = append(opts, v2.WorkflowWithPermissions)
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		pull, err := workflow.Pull(ctx, api.mustDB(), api.Cache, *proj, name, project.EncryptWithBuiltinKey, opts...)
		if err != nil {
			return err
		}

		// early returns as json if param set
		if FormBool(r, "json") {
			raw, err := pull.ToRaw()
			if err != nil {
				return err
			}
			return service.WriteJSON(w, raw, http.StatusOK)
		}

		buf := new(bytes.Buffer)
		if err := exportentities.TarWorkflowComponents(ctx, pull, buf); err != nil {
			return err
		}

		w.Header().Add("Content-Type", "application/tar")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, buf)
		return sdk.WrapError(err, "unable to copy content buffer in the response writer")
	}
}
