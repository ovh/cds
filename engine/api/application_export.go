package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getApplicationExportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		withPermissions := FormBool(r, "withPermissions")

		// Load app
		app, errload := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx),
			application.LoadOptions.WithVariablesWithClearPassword,
			application.LoadOptions.WithKeys,
		)
		if errload != nil {
			return sdk.WrapError(errload, "getApplicationExportHandler> Cannot load application %s", appName)
		}

		// Load permissions
		if withPermissions {
			perms, err := group.LoadGroupsByApplication(api.mustDB(), app.ID)
			if err != nil {
				return sdk.WrapError(err, "getApplicationExportHandler> Cannot load application %s permissions", appName)
			}
			app.ApplicationGroups = perms
		}

		// Parse variables
		appvars := []sdk.Variable{}
		for _, v := range app.Variable {
			switch v.Type {
			case sdk.KeyVariable:
				return sdk.WrapError(errload, "getApplicationExportHandler> Unable to eport application %s because of variable %s", appName, v.Name)
			case sdk.SecretVariable:
				content, err := project.EncryptWithBuiltinKey(api.mustDB(), app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, v.Name), v.Value)
				if err != nil {
					return sdk.WrapError(err, "getApplicationExportHandler> Unknown key type")
				}
				v.Value = content
				appvars = append(appvars, v)
			default:
				appvars = append(appvars, v)
			}
		}
		app.Variable = appvars

		// Prepare keys
		keys := []exportentities.EncryptedKey{}
		// Parse keys
		for _, k := range app.Keys {
			content, err := project.EncryptWithBuiltinKey(api.mustDB(), app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, k.Name), k.Private)
			if err != nil {
				return sdk.WrapError(err, "getApplicationExportHandler> Unable to encrypt key")
			}
			ek := exportentities.EncryptedKey{
				Type:    k.Type,
				Name:    k.Name,
				Content: content,
			}
			keys = append(keys, ek)
		}

		eapp, err := exportentities.NewApplication(app, withPermissions, keys)
		if err != nil {
			return sdk.WrapError(err, "getApplicationExportHandler> Unable to export application")
		}

		// Export
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "getApplicationExportHandler> Format invalid")
		}

		// Marshal to the desired format
		b, err := exportentities.Marshal(eapp, f)
		if err != nil {
			return sdk.WrapError(err, "getApplicationExportHandler>")
		}

		w.Header().Add("Content-Type", "application/"+format)
		w.WriteHeader(http.StatusOK)
		w.Write(b)

		return nil
	}
}
