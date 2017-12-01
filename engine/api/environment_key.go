package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getKeysInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getKeysInEnvironmentHandler> Cannot load environment")
		}

		if errK := environment.LoadAllKeys(api.mustDB(), env); errK != nil {
			return sdk.WrapError(errK, "getKeysInEnvironmentHandler> Cannot load environment keys")
		}

		return WriteJSON(w, r, env.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		keyName := vars["name"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "deleteKeyInEnvironmentHandler> Cannot load project")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "deleteKeyInEnvironmentHandler> Cannot load environment")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "v> Cannot start transaction")
		}
		defer tx.Rollback()
		for _, k := range env.Keys {
			if k.Name == keyName {
				if err := environment.DeleteEnvironmentKey(tx, env.ID, keyName); err != nil {
					return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot delete key %s", k.Name)
				}
				if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
					return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot update application last modified date")
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteKeyInEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

func (api *API) addKeyInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		var newKey sdk.EnvironmentKey
		if err := UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := regexp.MustCompile(sdk.NamePattern)
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInEnvironmentHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "addKeyInEnvironmentHandler> Cannot load project")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "addKeyInEnvironmentHandler> Cannot load environment")
		}

		newKey.EnvironmentID = env.ID

		switch newKey.Type {
		case sdk.KeyTypeSsh:
			pubR, privR, errGenerate := keys.GenerateSSHKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInEnvironmentHandler> Cannot generate sshKey")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return sdk.WrapError(errPub, "addKeyInApplicationHandler> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return sdk.WrapError(errPriv, "addKeyInApplicationHandler>  Unable to read private key")
			}
			newKey.Private = string(priv)
			newKey.Public = string(pub)
		case sdk.KeyTypePgp:
			kid, pubR, privR, errGenerate := keys.GeneratePGPKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInEnvironmentHandler> Cannot generate pgpKey")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return sdk.WrapError(errPub, "addKeyInEnvironmentHandler> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return sdk.WrapError(errPriv, "addKeyInEnvironmentHandler>  Unable to read private key")
			}
			newKey.Private = string(priv)
			newKey.Public = string(pub)
			newKey.KeyID = kid
		default:
			return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInEnvironmentHandler> unknown key of type: %s", newKey.Type)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := environment.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot insert application key")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addKeyInEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, newKey, http.StatusOK)
	}
}
