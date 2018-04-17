package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getPlatformModelsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p, err := platform.LoadModels(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getPlatformModels> Cannot get platform models")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPlatformModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		p, err := platform.LoadModelByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getPlatformModelHandler> Cannot get platform model")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) postPlatformModelHandler() Handler {
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

		if err := platform.InsertModel(tx, m); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> unable to insert model %s", m.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postPlatformModelHandler> Unable to commit tx")
		}

		return WriteJSON(w, m, http.StatusCreated)
	}
}

func (api *API) putPlatformModelHandler() Handler {
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

		log.Debug("putPlatformModelHandler> %+v", m)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> Unable to start tx")
		}

		defer tx.Rollback()

		old, err := platform.LoadModelByName(tx, name)
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
		m.PlatformModelPlugin = old.PlatformModelPlugin

		if err := platform.UpdateModel(tx, m); err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> ")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putPlatformModelHandler> Unable to commit tx")
		}

		return WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) deletePlatformModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to start tx")
		}
		defer tx.Rollback()

		old, err := platform.LoadModelByName(tx, name)
		if err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to load model")
		}

		if err := platform.DeleteModel(tx, old.ID); err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler>")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deletePlatformModelHandler> Unable to commit tx")
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
