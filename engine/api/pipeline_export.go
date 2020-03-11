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
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		f, err := exportentities.GetFormatFromPath(format)
		if err != nil {
			return err
		}

		pip, err := pipeline.Export(ctx, api.mustDB(), key, name)
		if err != nil {
			return err
		}
		buf, err := exportentities.Marshal(pip, f)
		if err != nil {
			return err
		}
		if _, err := w.Write(buf); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		w.WriteHeader(http.StatusOK)

		return nil
	}
}
