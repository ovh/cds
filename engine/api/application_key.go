package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getKeysInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		log.Debug("%s %s", key, appName)

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "getKeysInApplicationHandler> Cannot load application")
		}

		if errK := application.LoadAllKeys(api.mustDB(), app); errK != nil {
			return sdk.WrapError(errK, "getKeysInApplicationHandler> Cannot load application keys")
		}

		return WriteJSON(w, app.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		keyName := vars["name"]

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithKeys)
		if errA != nil {
			return sdk.WrapError(errA, "deleteKeyInApplicationHandler> Cannot load application")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "v> Cannot start transaction")
		}
		defer tx.Rollback()

		var keyToDelete sdk.ApplicationKey
		for _, k := range app.Keys {
			if k.Name == keyName {
				keyToDelete = k
				if err := application.DeleteApplicationKey(tx, app.ID, keyName); err != nil {
					return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot delete key %s", k.Name)
				}
				if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
					return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot update application last modified date")
				}
			}
		}

		if keyToDelete.Name == "" {
			return sdk.WrapError(sdk.ErrKeyNotFound, "deleteKeyInApplicationHandler> key %s not found on application %s", keyName, app.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot commit transaction")
		}

		event.PublishApplicationKeyDelete(key, *app, keyToDelete, getUser(ctx))

		return WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) addKeyInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var newKey sdk.ApplicationKey
		if err := UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInApplicationHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "addKeyInApplicationHandler> Cannot load application")
		}
		newKey.ApplicationID = app.ID

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
		defer tx.Rollback()

		if err := application.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot insert application key")
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot commit transaction")
		}

		event.PublishApplicationKeyAdd(key, *app, newKey, getUser(ctx))

		return WriteJSON(w, newKey, http.StatusOK)
	}
}
