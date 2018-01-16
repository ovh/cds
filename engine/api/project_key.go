package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getAllKeysProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		appName := r.FormValue("appName")

		allkeys := struct {
			ProjectKeys     []sdk.ProjectKey     `json:"project_key"`
			ApplicationKeys []sdk.ApplicationKey `json:"application_key"`
			EnvironmentKeys []sdk.EnvironmentKey `json:"environment_key"`
		}{}

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getAllKeysProjectHandler> Cannot load project")
		}
		projectKeys, errK := project.LoadAllKeysByID(api.mustDB(), p.ID)
		if errK != nil {
			return sdk.WrapError(errK, "getAllKeysProjectHandler> Cannot load project keys")
		}
		allkeys.ProjectKeys = projectKeys

		if appName == "" {
			appKeys, errA := application.LoadAllApplicationKeysByProject(api.mustDB(), p.ID)
			if errA != nil {
				return sdk.WrapError(errA, "getAllKeysProjectHandler> Cannot load application keys")
			}
			allkeys.ApplicationKeys = appKeys
		} else {
			app, errA := application.LoadByName(api.mustDB(), api.Cache, p.Key, appName, getUser(ctx))
			if errA != nil {
				return sdk.WrapError(errA, "getAllKeysProjectHandler> Cannot load application")
			}
			if errK := application.LoadAllKeys(api.mustDB(), app); errK != nil {
				return sdk.WrapError(errK, "getAllKeysProjectHandler> Cannot load application keys")
			}
			allkeys.ApplicationKeys = app.Keys
		}

		envKeys, errP := environment.LoadAllEnvironmentKeysByProject(api.mustDB(), p.ID)
		if errP != nil {
			return sdk.WrapError(errP, "getAllKeysProjectHandler> Cannot load environemnt keys")
		}
		allkeys.EnvironmentKeys = envKeys

		return WriteJSON(w, r, allkeys, http.StatusOK)
	}
}

func (api *API) getKeysInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getKeysInProjectHandler> Cannot load project")
		}

		if errK := project.LoadAllKeys(api.mustDB(), p); errK != nil {
			return sdk.WrapError(errK, "getKeysInProjectHandler> Cannot load project keys")
		}

		return WriteJSON(w, r, p.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		keyName := vars["name"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithKeys)
		if errP != nil {
			return sdk.WrapError(errP, "deleteKeyInProjectHandler> Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteKeyInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		for _, k := range p.Keys {
			if k.Name == keyName {
				if err := project.DeleteProjectKey(tx, p.ID, keyName); err != nil {
					return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot delete key %s", k.Name)
				}
				if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectKeysLastModificationType); err != nil {
					return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot update project last modified date")
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteKeyInProjectHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

func (api *API) addKeyInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		var newKey sdk.ProjectKey
		if err := UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInProjectHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "addKeyInProjectHandler> Cannot load project")
		}
		newKey.ProjectID = p.ID

		switch newKey.Type {
		case sdk.KeyTypeSSH:
			pubR, privR, errGenerate := keys.GenerateSSHKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInProjectHandler> Cannot generate sshKey")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return sdk.WrapError(errPub, "addKeyInProjectHandler> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return sdk.WrapError(errPriv, "addKeyInProjectHandler> Unable to read private key")
			}
			newKey.Public = string(pub)
			newKey.Private = string(priv)
		case sdk.KeyTypePGP:
			kid, pubR, privR, errGenerate := keys.GeneratePGPKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInProjectHandler> Cannot generate pgpKey")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return sdk.WrapError(errPub, "addKeyInProjectHandler> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return sdk.WrapError(errPriv, "addKeyInProjectHandler> Unable to read private key")
			}
			newKey.Public = string(pub)
			newKey.Private = string(priv)
			newKey.KeyID = kid
		default:
			return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInProjectHandler> unknown key of type: %s", newKey.Type)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := project.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "addKeyInProjectHandler> Cannot insert project key")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectKeysLastModificationType); err != nil {
			return sdk.WrapError(err, "addKeyInProjectHandler> Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addKeyInProjectHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, newKey, http.StatusOK)
	}
}
