package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getPlatformModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p, err := platform.LoadModels(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getPlatformModels> Cannot get platform models")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPlatformModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		clearPassword := FormBool(r, "clearPassword")

		p, err := platform.LoadModelByName(api.mustDB(), name, clearPassword)
		if err != nil {
			return sdk.WrapError(err, "getPlatformModelHandler> Cannot get platform model")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) postPlatformModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := new(sdk.PlatformModel)
		if err := UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> Unable to start tx")
		}

		defer tx.Rollback()

		if exist, err := platform.ModelExists(tx, m.Name); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> Unable to check if model %s exist", m.Name)
		} else if exist {
			return sdk.NewError(sdk.ErrConflict, fmt.Errorf("platform model %s already exist", m.Name))
		}

		if m.PluginName != "" {
			p, err := plugin.LoadByName(tx, m.PluginName)
			if err != nil {
				return sdk.WrapError(err, "postPlatformModelHandler")
			}
			m.PluginID = &p.ID
		}

		if err := platform.InsertModel(tx, m); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> unable to insert model %s", m.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> Unable to commit tx")
		}

		if m.Public {
			go propagatePublicPlatformModel(api.mustDB(), api.Cache, *m, getUser(ctx))
		}

		return service.WriteJSON(w, m, http.StatusCreated)
	}
}

func (api *API) putPlatformModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		if name == "" {
			return sdk.ErrNotFound
		}

		m := new(sdk.PlatformModel)
		if err := UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> Unable to start tx")
		}
		defer tx.Rollback()

		old, err := platform.LoadModelByName(tx, name, true)
		if err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> Unable to load model")
		}

		if old.IsBuiltin() {
			return sdk.WrapError(sdk.ErrForbidden, "putPlatformModelHandler> Update builtin model is forbidden")
		}

		if m.Name != old.Name {
			return sdk.ErrWrongRequest
		}

		m.ID = old.ID
		m.PluginID = old.PluginID

		if m.PluginName != "" {
			p, err := plugin.LoadByName(tx, m.PluginName)
			if err != nil {
				return sdk.WrapError(err, "putPlatformModelHandler")
			}
			m.PluginID = &p.ID
		}

		if err := platform.UpdateModel(tx, m); err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> ")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> Unable to commit tx")
		}

		if m.Public {
			go propagatePublicPlatformModel(api.mustDB(), api.Cache, *m, getUser(ctx))
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func propagatePublicPlatformModel(db *gorp.DbMap, store cache.Store, m sdk.PlatformModel, u *sdk.User) {
	if !m.Public && len(m.PublicConfigurations) > 0 {
		return
	}

	projs, err := project.LoadAll(context.Background(), db, store, nil, project.LoadOptions.WithClearPlatforms)
	if err != nil {
		log.Error("propagatePublicPlatformModel> Unable to retrieve all projects: %v", err)
		return
	}

	for _, p := range projs {
		tx, err := db.Begin()
		if err != nil {
			log.Error("propagatePublicPlatformModel> error: %v", err)
			continue
		}
		if err := propagatePublicPlatformModelOnProject(tx, store, m, p, u); err != nil {
			log.Error("propagatePublicPlatformModel> error: %v", err)
			_ = tx.Rollback()
			continue
		}
		if err := tx.Commit(); err != nil {
			log.Error("propagatePublicPlatformModel> unable to commit: %v", err)
		}
	}
}

func propagatePublicPlatformModelOnProject(db gorp.SqlExecutor, store cache.Store, m sdk.PlatformModel, p sdk.Project, u *sdk.User) error {
	if !m.Public {
		return nil
	}

	for pfName, immutableCfg := range m.PublicConfigurations {
		cfg := immutableCfg.Clone()
		oldPP, _ := platform.LoadPlatformsByName(db, p.Key, pfName, true)
		if oldPP.ID == 0 {
			pp := sdk.ProjectPlatform{
				Model:           m,
				PlatformModelID: m.ID,
				Name:            pfName,
				Config:          cfg,
				ProjectID:       p.ID,
			}
			if err := platform.InsertPlatform(db, &pp); err != nil {
				return sdk.WrapError(err, "propagatePublicPlatformModelOnProject> Unable to insert project platform %s", pp.Name)
			}
			event.PublishAddProjectPlatform(&p, pp, u)
			continue
		}

		pp := sdk.ProjectPlatform{
			ID:              oldPP.ID,
			Model:           m,
			PlatformModelID: m.ID,
			Name:            pfName,
			Config:          cfg,
			ProjectID:       p.ID,
		}
		oldPP.Config = m.DefaultConfig
		if err := platform.UpdatePlatform(db, pp); err != nil {
			return sdk.WrapError(err, "propagatePublicPlatformModelOnProject> unable to update project platform %s", pp.Name)
		}
		event.PublishUpdateProjectPlatform(&p, oldPP, pp, u)
	}
	return nil
}

func (api *API) deletePlatformModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to start tx")
		}
		defer tx.Rollback()

		old, err := platform.LoadModelByName(tx, name, false)
		if err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to load model")
		}

		if err := platform.DeleteModel(tx, old.ID); err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler>")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to commit tx")
		}

		return nil
	}
}
