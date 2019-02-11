package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postApplicationImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := FormBool(r, "force")

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
		if errp != nil {
			return sdk.WrapError(errp, "postApplicationImportHandler>> Unable load project")
		}

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
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
			errapp = json.Unmarshal(body, eapp)
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
		defer tx.Rollback()

		newApp, msgList, globalError := application.ParseAndImport(tx, api.Cache, proj, eapp, force, project.DecryptWithBuiltinKey, deprecatedGetUser(ctx))
		msgListString := translate(r, msgList)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "Unable to import application %s", eapp.Name)
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			log.Warning("%v", globalError)
			sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
			return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}
		event.PublishAddApplication(proj.Key, *newApp, deprecatedGetUser(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
