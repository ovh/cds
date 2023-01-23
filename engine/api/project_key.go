package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getKeysInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, err := project.Load(ctx, api.mustDB(), key)
		if err != nil {
			return err
		}

		keys, err := project.LoadAllKeys(ctx, api.mustDB(), p.ID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, keys, http.StatusOK)
	}
}

func (api *API) deleteKeyInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		keyName := vars["name"]

		p, errP := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithKeys)
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
			return sdk.WithStack(err)
		}

		event.PublishDeleteProjectKey(ctx, p, deletedKey, getUserConsumer(ctx))

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

		p, errP := project.Load(ctx, api.mustDB(), key)
		if errP != nil {
			return sdk.WrapError(errP, "addKeyInProjectHandler> Cannot load project")
		}
		newKey.ProjectID = p.ID

		if !strings.HasPrefix(newKey.Name, "proj-") {
			newKey.Name = "proj-" + newKey.Name
		}

		k, err := keys.GenerateKey(newKey.Name, newKey.Type)
		if err != nil {
			return err
		}
		newKey.Private = k.Private
		newKey.Public = k.Public
		newKey.KeyID = k.KeyID

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "addKeyInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := project.InsertKey(tx, &newKey); err != nil {
			return sdk.WrapError(err, "Cannot insert project key")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishAddProjectKey(ctx, p, newKey, getUserConsumer(ctx))

		return service.WriteJSON(w, newKey, http.StatusOK)
	}
}

func (api *API) postDisableKeyInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.ErrForbidden
		}

		vars := mux.Vars(r)
		key := vars[permProjectKey]
		keyName := vars["name"]

		p, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithKeys)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // nolint

		var updateKey sdk.ProjectKey
		for _, k := range p.Keys {
			if k.Name == keyName {
				updateKey = k
				updateKey.Disabled = true
				if err := project.DisableProjectKey(tx, p.ID, keyName); err != nil {
					return err
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		event.PublishDisableProjectKey(ctx, p, updateKey, getUserConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postEnableKeyInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.ErrForbidden
		}

		vars := mux.Vars(r)
		key := vars[permProjectKey]
		keyName := vars["name"]

		p, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithKeys)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // nolint

		var updateKey sdk.ProjectKey
		for _, k := range p.Keys {
			if k.Name == keyName {
				updateKey = k
				updateKey.Disabled = false
				if err := project.EnableProjectKey(tx, p.ID, keyName); err != nil {
					return err
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishEnableProjectKey(ctx, p, updateKey, getUserConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
