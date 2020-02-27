package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getApplicationExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		app, err := application.Export(api.mustDB(), api.Cache, key, appName, project.EncryptWithBuiltinKey)
		if err != nil {
			return sdk.WithStack(err)
		}
		f, err := yaml.Marshal(app)
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
