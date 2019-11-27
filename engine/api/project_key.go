package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getAllKeysProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		appName := r.FormValue("appName")

		allkeys := struct {
			ProjectKeys     []sdk.ProjectKey     `json:"project_key"`
			ApplicationKeys []sdk.ApplicationKey `json:"application_key"`
			EnvironmentKeys []sdk.EnvironmentKey `json:"environment_key"`
		}{}

		p, errP := project.Load(api.mustDB(), api.Cache, key)
		if errP != nil {
			return errP
		}
		projectKeys, errK := project.LoadAllKeysByID(api.mustDB(), p.ID)
		if errK != nil {
			return errK
		}
		allkeys.ProjectKeys = projectKeys

		if appName == "" {
			appKeys, errA := application.LoadAllApplicationKeysByProject(api.mustDB(), p.ID)
			if errA != nil {
				return errA
			}
			allkeys.ApplicationKeys = appKeys
		} else {
			app, errA := application.LoadByName(api.mustDB(), api.Cache, p.Key, appName)
			if errA != nil {
				return errA
			}
			if errK := application.LoadAllKeys(api.mustDB(), app); errK != nil {
				return errK
			}
			allkeys.ApplicationKeys = app.Keys
		}

		envKeys, errP := environment.LoadAllEnvironmentKeysByProject(api.mustDB(), p.ID)
		if errP != nil {
			return errP
		}
		allkeys.EnvironmentKeys = envKeys

		return service.WriteJSON(w, allkeys, http.StatusOK)
	}
}

func (api *API) getKeysInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, errP := project.Load(api.mustDB(), api.Cache, key)
		if errP != nil {
			return sdk.WrapError(errP, "getKeysInProjectHandler> Cannot load project")
		}

		if errK := project.LoadAllKeys(api.mustDB(), p); errK != nil {
			return sdk.WrapError(errK, "getKeysInProjectHandler> Cannot load project keys")
		}

		return service.WriteJSON(w, p.Keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		keyName := vars["name"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithKeys)
		if errP != nil {
			return sdk.WrapError(errP, "deleteKeyInProjectHandler> Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteKeyInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint
		var deletedKey sdk.ProjectKey
		for _, k := range p.Keys {
			if k.Name == keyName {
				deletedKey = k
				if err := project.DeleteProjectKey(tx, p.ID, keyName); err != nil {
					return sdk.WrapError(err, "Cannot delete key %s", k.Name)
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteProjectKey(ctx, p, deletedKey, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) addKeyInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		var newKey sdk.ProjectKey
		if err := service.UnmarshalBody(r, &newKey); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(newKey.Name) {
			return sdk.WrapError(sdk.ErrInvalidKeyPattern, "addKeyInProjectHandler: Key name %s do not respect pattern %s", newKey.Name, sdk.NamePattern)
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key)
		if errP != nil {
			return sdk.WrapError(errP, "addKeyInProjectHandler> Cannot load project")
		}
		newKey.ProjectID = p.ID

		if !strings.HasPrefix(newKey.Name, "proj-") {
			newKey.Name = "proj-" + newKey.Name
		}

		switch newKey.Type {
		case sdk.KeyTypeSSH:
			k, errK := keys.GenerateSSHKey(newKey.Name)
			if errK != nil {
				return sdk.WrapError(errK, "addKeyInProjectHandler> Cannot generate ssh key")
			}
			newKey.Key = k
		case sdk.KeyTypePGP:
			k, errGenerate := keys.GeneratePGPKeyPair(newKey.Name)
			if errGenerate != nil {
				return sdk.WrapError(errGenerate, "addKeyInProjectHandler> Cannot generate pgpKey")
			}
			newKey.Key = k
		default:
			return sdk.WrapError(sdk.ErrUnknownKeyType, "addKeyInProjectHandler> unknown key of type: %s", newKey.Type)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := project.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "Cannot insert project key")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishAddProjectKey(ctx, p, newKey, getAPIConsumer(ctx))

		return service.WriteJSON(w, newKey, http.StatusOK)
	}
}
