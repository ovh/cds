package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func getKeysInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "getKeysInProjectHandler> Cannot load project")
	}

	if errK := project.LoadAllKeys(db, p); errK != nil {
		return sdk.WrapError(errK, "getKeysInProjectHandler> Cannot load project keys")
	}

	return WriteJSON(w, r, p.Keys, http.StatusOK)
}

func deleteKeyInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	keyName := vars["name"]

	p, errP := project.Load(db, key, c.User, project.LoadOptions.WithKeys)
	if errP != nil {
		return sdk.WrapError(errP, "deleteKeyInProjectHandler> Cannot load project")
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "deleteKeyInProjectHandler> Cannot start transaction")
	}
	defer tx.Rollback()
	for _, k := range p.Keys {
		if k.Name == keyName {
			if err := project.DeleteProjectKey(tx, p.ID, keyName); err != nil {
				return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot delete key %s", k.Name)
			}
			if err := project.UpdateLastModified(db, c.User, p); err != nil {
				return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot update project last modified date")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func addKeyInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	var newKey sdk.ProjectKey
	if err := UnmarshalBody(r, &newKey); err != nil {
		return err
	}

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "addKeyInProjectHandler> Cannot load project")
	}
	newKey.ProjectID = p.ID

	switch newKey.Type {
	case sdk.KeyTypeSsh:
		pub, priv, errGenerate := keys.Generatekeypair(newKey.Name)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInProjectHandler> Cannot generate sshKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	case sdk.KeyTypePgp:
		pub, priv, errGenerate := keys.GeneratePGPKeyPair(newKey.Name, c.User)
		if errGenerate != nil {
			return sdk.WrapError(errGenerate, "addKeyInProjectHandler> Cannot generate pgpKey")
		}
		newKey.Public = pub
		newKey.Private = priv
	default:
		return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInProjectHandler> unknown key of type: %s", newKey.Type)
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "addKeyInProjectHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := project.InsertKey(tx, &newKey); err != nil {
		return sdk.WrapError(err, "addKeyInProjectHandler> Cannot insert project key")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "addKeyInProjectHandler> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addKeyInProjectHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, newKey, http.StatusOK)
}
