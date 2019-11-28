package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getKeysInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getKeysInEnvironmentHandler> Cannot load environment")
		}

		if errK := environment.LoadAllKeys(api.mustDB(), env); errK != nil {
			return sdk.WrapError(errK, "getKeysInEnvironmentHandler> Cannot load environment keys")
		}

		return service.WriteJSON(w, env.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		keyName := vars["name"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "deleteKeyInEnvironmentHandler> Cannot load environment")
		}
		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "v> Cannot start transaction")
		}
		defer tx.Rollback() // nolint
		var envKey sdk.EnvironmentKey
		for _, k := range env.Keys {
			if k.Name == keyName {
				envKey = k
				if err := environment.DeleteEnvironmentKey(tx, env.ID, keyName); err != nil {
					return sdk.WrapError(err, "Cannot delete key %s", k.Name)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishEnvironmentKeyDelete(ctx, key, *env, envKey, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) addKeyInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]

		var newKey sdk.EnvironmentKey
		if err := service.UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInEnvironmentHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "addKeyInEnvironmentHandler> Cannot load environment")
		}
		newKey.EnvironmentID = env.ID

		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if !strings.HasPrefix(newKey.Name, "env-") {
			newKey.Name = "env-" + newKey.Name
		}

		switch newKey.Type {
		case sdk.KeyTypeSSH:
			k, errK := keys.GenerateSSHKey(newKey.Name)
			if errK != nil {
				return sdk.WrapError(errK, "addKeyInEnvironmentHandler> Cannot generate ssh key")
			}
			newKey.Key = k
		case sdk.KeyTypePGP:
			k, errGenerate := keys.GeneratePGPKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInEnvironmentHandler> Cannot generate pgpKey")
			}
			newKey.Key = k
		default:
			return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInEnvironmentHandler> unknown key of type: %s", newKey.Type)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := environment.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "Cannot insert application key")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishEnvironmentKeyAdd(ctx, key, *env, newKey, getAPIConsumer(ctx))

		return service.WriteJSON(w, newKey, http.StatusOK)
	}
}
