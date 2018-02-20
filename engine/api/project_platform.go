package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectPlatformsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getProjectPlatformsHandler> Cannot load project")
		}

		projectPlatforms, err := project.LoadPlatformsByID(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "getProjectPlatformsHandler> Cannot load project platform")
		}
		return WriteJSON(w, projectPlatforms, http.StatusOK)
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
