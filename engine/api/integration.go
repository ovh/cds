package api

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

		p, err := integration.LoadModelByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "Cannot get integration model")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) postIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		m := new(sdk.IntegrationModel)
		if err := service.UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "postIntegrationModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}

		defer tx.Rollback() // nolint

		if exist, err := integration.ModelExists(tx, m.Name); err != nil {
			return sdk.WrapError(err, "unable to check if model %s exist", m.Name)
		} else if exist {
			return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "integration model %s already exist", m.Name)
		}

		if err := integration.InsertModel(tx, m); err != nil {
			return sdk.WrapError(err, "unable to insert model %s", m.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		event_v2.PublishIntegrationModelEvent(ctx, api.Cache, sdk.EventIntegrationModelCreated, *m, *u.AuthConsumerUser.AuthentifiedUser)

		if m.Public {
			go propagatePublicIntegrationModel(ctx, api.mustDB(), api.Cache, *m, *u)
		}

		return service.WriteJSON(w, m, http.StatusCreated)
	}
}

func (api *API) putIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if name == "" {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		m := new(sdk.IntegrationModel)
		if err := service.UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "putIntegrationModelHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback() // nolint

		old, err := integration.LoadModelByName(ctx, tx, name)
		if err != nil {
			return sdk.WrapError(err, "Unable to load model")
		}

		if old.IsBuiltin() {
			return sdk.WrapError(sdk.ErrForbidden, "putIntegrationModelHandler> Update builtin model is forbidden")
		}

		if m.Name != old.Name {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		m.ID = old.ID
		if err := integration.UpdateModel(ctx, tx, m); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit tx")
		}

		event_v2.PublishIntegrationModelEvent(ctx, api.Cache, sdk.EventIntegrationModelUpdated, *m, *u.AuthConsumerUser.AuthentifiedUser)

		if m.Public {
			api.GoRoutines.Exec(ctx, "propagatePublicIntegrationModel", func(ctx context.Context) {
				propagatePublicIntegrationModel(ctx, api.mustDB(), api.Cache, *m, *u)
			})
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func propagatePublicIntegrationModel(ctx context.Context, db *gorp.DbMap, store cache.Store, m sdk.IntegrationModel, u sdk.AuthUserConsumer) {
	if !m.Public && len(m.PublicConfigurations) > 0 {
		return
	}

	projs, err := project.LoadAll(context.Background(), db, store, nil, project.LoadOptions.WithClearIntegrations)
	if err != nil {
		log.Error(ctx, "propagatePublicIntegrationModel> Unable to retrieve all projects: %v", err)
		return
	}

	for _, p := range projs {
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "propagatePublicIntegrationModel> error: %v", err)
			continue
		}
		created, updated, err := propagatePublicIntegrationModelOnProject(ctx, tx, store, m, p, u)
		if err != nil {
			log.Error(ctx, "propagatePublicIntegrationModel> error: %v", err)
			_ = tx.Rollback()
			continue
		}
		if err := tx.Commit(); err != nil {
			log.Error(ctx, "propagatePublicIntegrationModel> unable to commit: %v", err)
		}
		for _, pp := range created {
			event_v2.PublishProjectIntegrationEvent(ctx, store, sdk.EventIntegrationCreated, p.Key, pp, *u.AuthConsumerUser.AuthentifiedUser)
		}
		for _, pp := range updated {
			event_v2.PublishProjectIntegrationEvent(ctx, store, sdk.EventIntegrationUpdated, p.Key, pp, *u.AuthConsumerUser.AuthentifiedUser)
		}
	}
}

func propagatePublicIntegrationModelOnProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, m sdk.IntegrationModel, p sdk.Project, u sdk.Identifiable) ([]sdk.ProjectIntegration, []sdk.ProjectIntegration, error) {
	if !m.Public {
		return nil, nil, nil
	}

	createdIntegration := make([]sdk.ProjectIntegration, 0)
	updatedIntegration := make([]sdk.ProjectIntegration, 0)
	for pfName, immutableCfg := range m.PublicConfigurations {
		cfg := immutableCfg.Clone()
		oldPP, _ := integration.LoadProjectIntegrationByNameWithClearPassword(ctx, db, p.Key, pfName)
		if oldPP.ID == 0 {
			pp := sdk.ProjectIntegration{
				Model:              m,
				IntegrationModelID: m.ID,
				Name:               pfName,
				Config:             cfg,
				ProjectID:          p.ID,
			}
			if err := integration.InsertIntegration(db, &pp); err != nil {
				return nil, nil, sdk.WrapError(err, "Unable to insert integration %s", pp.Name)
			}
			createdIntegration = append(createdIntegration, pp)
			event.PublishAddProjectIntegration(ctx, &p, pp, u)
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
		if err := integration.UpdateIntegration(ctx, db, pp); err != nil {
			return nil, nil, err
		}
		event.PublishUpdateProjectIntegration(ctx, &p, oldPP, pp, u)
		updatedIntegration = append(updatedIntegration, pp)
	}
	return createdIntegration, updatedIntegration, nil
}

func (api *API) deleteIntegrationModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback() // nolint

		old, err := integration.LoadModelByName(ctx, tx, name)
		if err != nil {
			return err
		}

		if err := integration.DeleteModel(ctx, tx, old.ID); err != nil {
			return sdk.WithStack(err)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit tx")
		}

		event_v2.PublishIntegrationModelEvent(ctx, api.Cache, sdk.EventIntegrationModelDeleted, old, *u.AuthConsumerUser.AuthentifiedUser)

		return nil
	}
}
