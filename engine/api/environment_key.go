package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func getKeysInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	env, errE := environment.LoadEnvironmentByName(db, key, envName)
	if errE != nil {
		return sdk.WrapError(errE, "getKeysInEnvironmentHandler> Cannot load environment")
	}

	if errK := environment.LoadAllKeys(db, env); errK != nil {
		return sdk.WrapError(errK, "getKeysInEnvironmentHandler> Cannot load environment keys")
	}

	return WriteJSON(w, r, env.Keys, http.StatusOK)
}

func deleteKeyInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	keyName := vars["name"]

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "deleteKeyInEnvironmentHandler> Cannot load project")
	}

	env, errE := environment.LoadEnvironmentByName(db, key, envName)
	if errE != nil {
		return sdk.WrapError(errE, "deleteKeyInEnvironmentHandler> Cannot load environment")
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "v> Cannot start transaction")
	}
	defer tx.Rollback()
	for _, k := range env.Keys {
		if k.Name == keyName {
			if err := environment.DeleteEnvironmentKey(tx, env.ID, keyName); err != nil {
				return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot delete key %s", k.Name)
			}
			if err := project.UpdateLastModified(tx, c.User, p); err != nil {
				return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot update application last modified date")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func addKeyInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	var newKey sdk.EnvironmentKey
	if err := UnmarshalBody(r, &newKey); err != nil {
		return err
	}

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "addKeyInEnvironmentHandler> Cannot load project")
	}

	env, errE := environment.LoadEnvironmentByName(db, key, envName)
	if errE != nil {
		return sdk.WrapError(errE, "addKeyInEnvironmentHandler> Cannot load environment")
	}

	newKey.EnvironmentID = env.ID

	switch newKey.Type {
	case sdk.KeyTypeSsh:
		pub, priv, errGenerate := keys.Generatekeypair(newKey.Name)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInEnvironmentHandler> Cannot generate sshKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	case sdk.KeyTypePgp:
		pub, priv, errGenerate := keys.GeneratePGPKeyPair(newKey.Name, c.User)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInEnvironmentHandler> Cannot generate pgpKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	default:
		return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInEnvironmentHandler> unknown key of type: %s", newKey.Type)
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "addKeyInEnvironmentHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := environment.InsertKey(tx, &newKey); err != nil {
		return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot insert application key")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, newKey, http.StatusOK)
}
