package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getWorkflowExportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		withPermissions := FormBool(r, "withPermissions")

		wf, errload := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errload != nil {
			return sdk.WrapError(errload, "permWogetWorkflowExportHandlerrkflowName> Cannot load workflow %s", name)
		}

		e, err := exportentities.NewWorkflow(*wf, withPermissions)
		if err != nil {
			return err
		}

		// Export
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowExportHandler> Format invalid")
		}

		// Marshal to the desired format
		b, err := exportentities.Marshal(e, f)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowExportHandler>")
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		w.WriteHeader(http.StatusOK)
		w.Write(b)

		return nil
	}
}
