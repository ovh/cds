package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getKeysInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		log.Debug("%s %s", key, appName)

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if errA != nil {
			return sdk.WrapError(errA, "getKeysInApplicationHandler> Cannot load application")
		}

		if errK := application.LoadAllKeys(api.mustDB(), app); errK != nil {
			return sdk.WrapError(errK, "getKeysInApplicationHandler> Cannot load application keys")
		}

		return service.WriteJSON(w, app.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		keyName := vars["name"]

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, application.LoadOptions.WithKeys)
		if errA != nil {
			return sdk.WrapError(errA, "deleteKeyInApplicationHandler> Cannot load application")
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "v> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		var keyToDelete sdk.ApplicationKey
		for _, k := range app.Keys {
			if k.Name == keyName {
				keyToDelete = k
				if err := application.DeleteApplicationKey(tx, app.ID, keyName); err != nil {
					return sdk.WrapError(err, "Cannot delete key %s", k.Name)
				}
			}
		}

		if keyToDelete.Name == "" {
			return sdk.WrapError(sdk.ErrKeyNotFound, "deleteKeyInApplicationHandler> key %s not found on application %s", keyName, app.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}
		event.PublishApplicationKeyDelete(ctx, key, *app, keyToDelete, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) addKeyInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		var newKey sdk.ApplicationKey
		if err := service.UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInApplicationHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if errA != nil {
			return sdk.WrapError(errA, "addKeyInApplicationHandler> Cannot load application")
		}
		newKey.ApplicationID = app.ID

		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if !strings.HasPrefix(newKey.Name, "app-") {
			newKey.Name = "app-" + newKey.Name
		}

		switch newKey.Type {
		case sdk.KeyTypeSSH:
			k, errK := keys.GenerateSSHKey(newKey.Name)
			if errK != nil {
				return sdk.WrapError(errK, "addKeyInApplicationHandler> Cannot generate ssh key")
			}
			newKey.Key = k
		case sdk.KeyTypePGP:
			k, errGenerate := keys.GeneratePGPKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInApplicationHandler> Cannot generate pgpKey")
			}
			newKey.Key = k
		default:
			return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInApplicationHandler> unknown key of type: %s", newKey.Type)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := application.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "Cannot insert application key")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishApplicationKeyAdd(ctx, key, *app, newKey, getAPIConsumer(ctx))

		return service.WriteJSON(w, newKey, http.StatusOK)
	}
}
