package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
)

func getKeysInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "getKeysInApplicationHandler> Cannot load application")
	}

	if errK := application.LoadAllKeys(db, app); errK != nil {
		return sdk.WrapError(errK, "getKeysInApplicationHandler> Cannot load application keys")
	}

	return WriteJSON(w, r, app.Keys, http.StatusOK)
}

func deleteKeyInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	keyName := vars["name"]

	app, errA := application.LoadByName(db, key, appName, c.User, application.LoadOptions.WithKeys)
	if errA != nil {
		return sdk.WrapError(errA, "deleteKeyInApplicationHandler> Cannot load application")
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "v> Cannot start transaction")
	}
	defer tx.Rollback()
	for _, k := range app.Keys {
		if k.Name == keyName {
			if err := application.DeleteApplicationKey(tx, app.ID, keyName); err != nil {
				return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot delete key %s", k.Name)
			}
			if err := application.UpdateLastModified(tx, app, c.User); err != nil {
				return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot update application last modified date")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteKeyInApplicationHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func addKeyInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var newKey sdk.ApplicationKey
	if err := UnmarshalBody(r, &newKey); err != nil {
		return err
	}

	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "addKeyInApplicationHandler> Cannot load application")
	}
	newKey.ApplicationID = app.ID

	switch newKey.Type {
	case sdk.KeyTypeSsh:
		pub, priv, errGenerate := keys.Generatekeypair(newKey.Name)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInApplicationHandler> Cannot generate sshKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	case sdk.KeyTypePgp:
		pub, priv, errGenerate := keys.GeneratePGPKeyPair(newKey.Name, c.User)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInApplicationHandler> Cannot generate pgpKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	default:
		return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInApplicationHandler> unknown key of type: %s", newKey.Type)
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "addKeyInApplicationHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := application.InsertKey(tx, &newKey); err != nil {
		return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot insert application key")
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addKeyInApplicationHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, newKey, http.StatusOK)
}
