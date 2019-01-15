package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectPlatformHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		clearPassword := FormBool(r, "clearPassword")

		platform, err := platform.LoadPlatformsByName(api.mustDB(), projectKey, platformName, clearPassword)
		if err != nil {
			return sdk.WrapError(err, "Cannot load platform %s/%s", projectKey, platformName)
		}
		return service.WriteJSON(w, platform, http.StatusOK)
	}
}

func (api *API) putProjectPlatformHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		var ppBody sdk.ProjectPlatform
		if err := service.UnmarshalBody(r, &ppBody); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		ppDB, errP := platform.LoadPlatformsByName(api.mustDB(), projectKey, platformName, true)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectPlatformHandler> Cannot load platform %s for project %s", platformName, projectKey)
		}

		//If the platform model is public, it's forbidden to update the project platform
		if ppDB.Model.Public {
			return sdk.ErrForbidden
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

		if err := platform.UpdatePlatform(tx, ppBody); err != nil {
			return sdk.WrapError(err, "Cannot update project platform")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishUpdateProjectPlatform(p, ppBody, ppDB, getUser(ctx))

		return service.WriteJSON(w, ppBody, http.StatusOK)
	}
}

func (api *API) deleteProjectPlatformHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		platformName := vars["platformName"]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteProjectPlatformHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		var deletedPlatform sdk.ProjectPlatform
		for _, plat := range p.Platforms {
			if plat.Name == platformName {
				//If the platform model is public, it's forbidden to delete the project platform
				if plat.Model.Public {
					return sdk.ErrForbidden
				}

				deletedPlatform = plat
				if err := platform.DeletePlatform(tx, plat); err != nil {
					return sdk.WrapError(err, "Cannot delete project platform")
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteProjectPlatform(p, deletedPlatform, getUser(ctx))
		return nil
	}
}

func (api *API) getProjectPlatformsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "getProjectPlatformsHandler> Cannot load project")
		}
		return service.WriteJSON(w, p.Platforms, http.StatusOK)
	}
}

func (api *API) postProjectPlatformHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		var pp sdk.ProjectPlatform
		if err := service.UnmarshalBody(r, &pp); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		pp.ProjectID = p.ID
		if pp.PlatformModelID == 0 {
			pp.PlatformModelID = pp.Model.ID
		}
		if pp.PlatformModelID == 0 && pp.Model.Name != "" {
			pfs, _ := platform.LoadModels(api.mustDB())
			for _, pf := range pfs {
				if pf.Name == pp.Model.Name {
					pp.PlatformModelID = pf.ID
					break
				}
			}
		}

		if pp.PlatformModelID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "postProjectPlatformHandler> model not found")
		}

		for _, pprojPlat := range p.Platforms {
			if pprojPlat.Name == pp.Name {
				if pprojPlat.Model.Public {
					return sdk.ErrForbidden
				}
				return sdk.WrapError(sdk.ErrWrongRequest, "postProjectPlatformHandler> project platform already exist")
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "postProjectPlatformHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := platform.InsertPlatform(tx, &pp); err != nil {
			return sdk.WrapError(err, "Cannot insert project platform")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishAddProjectPlatform(p, pp, getUser(ctx))

		return service.WriteJSON(w, pp, http.StatusOK)
	}
}
