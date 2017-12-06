package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getEnvironmentExportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		withPermissions := FormBool(r, "withPermissions")

		// Export
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "getEnvironmentExportHandler> Format invalid")
		}
		if err := environment.Export(api.mustDB(), api.Cache, key, envName, f, withPermissions, getUser(ctx), project.EncryptWithBuiltinKey, w); err != nil {
			return sdk.WrapError(err, "getEnvironmentExportHandler")
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		w.WriteHeader(http.StatusOK)
		return nil
	}
}
