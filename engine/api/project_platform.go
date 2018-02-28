package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectPlatformHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		clearPassword := FormBool(r, "clearPassword")

		platform, err := project.LoadPlatformsByName(api.mustDB(), projectKey, platformName, clearPassword)
		if err != nil {
			return sdk.WrapError(err, "getProjectPlatformHandler> Cannot load platform %s/%s", projectKey, platformName)
		}
		return WriteJSON(w, platform, http.StatusOK)
	}
}

func (api *API) putProjectPlatformHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		var ppBody sdk.ProjectPlatform
		if err := UnmarshalBody(r, &ppBody); err != nil {
			return sdk.WrapError(err, "putProjectPlatformHandler> Cannot read body")
		}

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "putProjectPlatformHandler> Cannot load project")
		}

		ppDB, errP := project.LoadPlatformsByName(api.mustDB(), projectKey, platformName, true)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectPlatformHandler> Cannot load project platform")
		}

		ppBody.ID = ppDB.ID

		for kkBody := range ppBody.Config {
			c := ppBody.Config[kkBody]
			// if we received a placeholder, replace with the right value
			if c.Type == sdk.PlatformConfigTypePassword && c.Value == sdk.PasswordPlaceholder {
				for kkDB, ccDB := range ppDB.Config {
					if kkDB == kkBody {
						c.Value = ccDB.Value
						break
					}
				}
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putProjectPlatformHandler> Cannot strat transaction")
		}
		defer tx.Rollback()

		if err := project.UpdatePlatform(tx, ppBody); err != nil {
			return sdk.WrapError(err, "putProjectPlatformHandler> Cannot update project platform")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectPlatformsLastModificationType); err != nil {
			return sdk.WrapError(err, "putProjectPlatformHandler> Cannot update project last modification date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putProjectPlatformHandler> Cannot commit transaction")
		}

		return WriteJSON(w, ppBody, http.StatusOK)

	}
}

func (api *API) deleteProjectPlatformHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "deleteProjectPlatformHandler> Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteProjectPlatformHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		for _, plat := range p.Platforms {
			if plat.Name == platformName {
				if err := project.DeletePlatform(tx, plat); err != nil {
					return sdk.WrapError(err, "deleteProjectPlatformHandler> Cannot delete project platform")
				}
				break
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectPlatformsLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteProjectPlatformHandler> Cannot update project last modification date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteProjectPlatformHandler> Cannot commit transaction")
		}
		return nil
	}
}

func (api *API) getProjectPlatformsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "getProjectPlatformsHandler> Cannot load project")
		}
		return WriteJSON(w, p.Platforms, http.StatusOK)
	}
}

func (api *API) postProjectPlatformHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "postProjectPlatformHandler> Cannot load project")
		}

		var pp sdk.ProjectPlatform
		if err := UnmarshalBody(r, &pp); err != nil {
			return sdk.WrapError(err, "postProjectPlatformHandler> Cannot read body")
		}

		pp.ProjectID = p.ID
		if pp.PlatformModelID == 0 {
			pp.PlatformModelID = pp.Model.ID
		}
		for _, pprojPlat := range p.Platforms {
			if pprojPlat.Name == pp.Name {
				return sdk.WrapError(sdk.ErrWrongRequest, "postProjectPlatformHandler> project platform already exist")
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "postProjectPlatformHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := project.InsertPlatform(tx, &pp); err != nil {
			return sdk.WrapError(err, "postProjectPlatformHandler> Cannot insert project platform")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectPlatformsLastModificationType); err != nil {
			return sdk.WrapError(err, "postProjectPlatformHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postProjectPlatformHandler> Cannot commit transaction")
		}

		return WriteJSON(w, pp, http.StatusOK)
	}
}
