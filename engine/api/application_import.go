package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) postApplicationImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := service.FormBool(r, "force")

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var eapp = new(exportentities.Application)
		var errapp error
		switch contentType {
		case "application/json":
			errapp = sdk.JSONUnmarshal(body, eapp)
		case "application/x-yaml", "text/x-yam":
			errapp = yaml.Unmarshal(body, eapp)
		default:
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unsupported content-type: %s", contentType))
		}

		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback() // nolint

		newApp, _, msgList, globalError := application.ParseAndImport(ctx, tx, api.Cache, *proj, eapp, application.ImportOptions{Force: force}, project.DecryptWithBuiltinKey, getUserConsumer(ctx))
		msgListString := translate(msgList)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "Unable to import application %s", eapp.Name)
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			sdkErr := sdk.ExtractHTTPError(globalError)
			return service.WriteJSON(w, append(msgListString, sdkErr.Error()), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
		event.PublishAddApplication(ctx, proj.Key, *newApp, getUserConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
