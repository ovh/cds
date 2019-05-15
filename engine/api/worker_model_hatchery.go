package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) bookWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "invalid permModelID")
		}

		if _, err := worker.BookForRegister(api.Cache, workerModelID, getHatchery(ctx)); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) spawnErrorWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var spawnErrorForm sdk.SpawnErrorForm
		if err := service.UnmarshalBody(r, &spawnErrorForm); err != nil {
			return sdk.WrapError(err, "Unable to parse spawn error form")
		}

		workerModelID, err := requestVarInt(r, "permModelID")
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		model, err := worker.LoadWorkerModelByID(tx, workerModelID)
		if err != nil {
			return err
		}

		if spawnErrorForm.Error == "" && len(spawnErrorForm.Logs) == 0 {
			return nil
		}

		if err := worker.UpdateSpawnErrorWorkerModel(tx, model.ID, spawnErrorForm); err != nil {
			return sdk.WrapError(err, "cannot update spawn error on worker model")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)
		worker.UnbookForRegister(api.Cache, workerModelID)

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkerModelsEnabledHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		h := getHatchery(ctx)
		if h == nil || h.GroupID == nil || *h.GroupID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "this route can be called only by hatchery: %+v", h)
		}

		models, err := worker.LoadWorkerModelsUsableOnGroupWithClearPassword(api.mustDB(), api.Cache, *h.GroupID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker models for hatchery %d with group %d", h.ID, *h.GroupID)
		}

		return service.WriteJSON(w, models, http.StatusOK)
	}
}
