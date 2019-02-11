package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getPipelineExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}

		// Export
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "Format invalid")
		}
		if _, err := pipeline.Export(api.mustDB(), api.Cache, key, name, f, w); err != nil {
			return sdk.WrapError(err, "getPipelineExportHandler")
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		w.WriteHeader(http.StatusOK)

		return nil
	}
}
