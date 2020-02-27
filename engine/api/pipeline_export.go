package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	yaml "gopkg.in/yaml.v2"
)

func (api *API) getPipelineExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["pipelineKey"]

		pip, err := pipeline.Export(ctx, api.mustDB(), key, name)
		if err != nil {
			return sdk.WithStack(err)
		}
		f, err := yaml.Marshal(pip)
		if err != nil {
			return sdk.WithStack(err)
		}
		if _, err := w.Write(f); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", string(exportentities.FormatYAML))
		w.WriteHeader(http.StatusOK)

		return nil
	}
}
