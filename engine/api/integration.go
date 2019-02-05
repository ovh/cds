package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getIntegrationModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p, err := integration.LoadModels(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "Cannot get integration models")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		clearPassword := FormBool(r, "clearPassword")

		p, err := integration.LoadModelByName(api.mustDB(), name, clearPassword)
		if err != nil {
			return sdk.WrapError(err, "Cannot get integration model")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) postIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := new(sdk.IntegrationModel)
		if err := service.UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "postIntegrationModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}

		defer tx.Rollback()

		if exist, err := integration.ModelExists(tx, m.Name); err != nil {
			return sdk.WrapError(err, "Unable to check if model %s exist", m.Name)
		} else if exist {
			return sdk.NewError(sdk.ErrConflict, fmt.Errorf("integration model %s already exist", m.Name))
		}

		if err := integration.InsertModel(tx, m); err != nil {
			return sdk.WrapError(err, "unable to insert model %s", m.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit tx")
		}

		if m.Public {
			go propagatePublicIntegrationModel(api.mustDB(), api.Cache, *m, deprecatedGetUser(ctx))
		}

		return service.WriteJSON(w, m, http.StatusCreated)
	}
}

func (api *API) putIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		if name == "" {
			return sdk.ErrNotFound
		}

		m := new(sdk.IntegrationModel)
		if err := service.UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "putIntegrationModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback()

		old, err := integration.LoadModelByName(tx, name, true)
		if err != nil {
			return sdk.WrapError(err, "Unable to load model")
		}

		if old.IsBuiltin() {
			return sdk.WrapError(sdk.ErrForbidden, "putIntegrationModelHandler> Update builtin model is forbidden")
		}

		if m.Name != old.Name {
			return sdk.ErrWrongRequest
		}

		m.ID = old.ID
		if err := integration.UpdateModel(tx, m); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit tx")
		}

		if m.Public {
			go propagatePublicIntegrationModel(api.mustDB(), api.Cache, *m, deprecatedGetUser(ctx))
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func propagatePublicIntegrationModel(db *gorp.DbMap, store cache.Store, m sdk.IntegrationModel, u *sdk.User) {
	if !m.Public && len(m.PublicConfigurations) > 0 {
		return
	}

	projs, err := project.LoadAll(context.Background(), db, store, nil, project.LoadOptions.WithClearIntegrations)
	if err != nil {
		log.Error("propagatePublicIntegrationModel> Unable to retrieve all projects: %v", err)
		return
	}

	for _, p := range projs {
		tx, err := db.Begin()
		if err != nil {
			log.Error("propagatePublicIntegrationModel> error: %v", err)
			continue
		}
		if err := propagatePublicIntegrationModelOnProject(tx, store, m, p, u); err != nil {
			log.Error("propagatePublicIntegrationModel> error: %v", err)
			_ = tx.Rollback()
			continue
		}
		if err := tx.Commit(); err != nil {
			log.Error("propagatePublicIntegrationModel> unable to commit: %v", err)
		}
	}
}

func propagatePublicIntegrationModelOnProject(db gorp.SqlExecutor, store cache.Store, m sdk.IntegrationModel, p sdk.Project, u *sdk.User) error {
	if !m.Public {
		return nil
	}

	for pfName, immutableCfg := range m.PublicConfigurations {
		cfg := immutableCfg.Clone()
		oldPP, _ := integration.LoadProjectIntegrationByName(db, p.Key, pfName, true)
		if oldPP.ID == 0 {
			pp := sdk.ProjectIntegration{
				Model:              m,
				IntegrationModelID: m.ID,
				Name:               pfName,
				Config:             cfg,
				ProjectID:          p.ID,
			}
			if err := integration.InsertIntegration(db, &pp); err != nil {
				return sdk.WrapError(err, "Unable to insert integration %s", pp.Name)
			}
			event.PublishAddProjectIntegration(&p, pp, u)
			continue
		}

		pp := sdk.ProjectIntegration{
			ID:                 oldPP.ID,
			Model:              m,
			IntegrationModelID: m.ID,
			Name:               pfName,
			Config:             cfg,
			ProjectID:          p.ID,
		}
		oldPP.Config = m.DefaultConfig
		if err := integration.UpdateIntegration(db, pp); err != nil {
			return err
		}
		event.PublishUpdateProjectIntegration(&p, oldPP, pp, u)
	}
	return nil
}

func (api *API) deleteIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback()

		old, err := integration.LoadModelByName(tx, name, false)
		if err != nil {
			return err
		}

		if err := integration.DeleteModel(tx, old.ID); err != nil {
			return sdk.WithStack(err)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit tx")
		}

		return nil
	}
}
