package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/workerhook"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectIntegrationWorkerHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
		if err != nil {
			return err
		}

		wh, err := workerhook.LoadAllByProjectIntegrationID(ctx, api.mustDB(), integ.ID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, wh, http.StatusOK)
	}
}

func (api *API) postProjectIntegrationWorkerHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		defer tx.Rollback() // nolint

		var inputWh []sdk.WorkerHookProjectIntegrationModel
		if err := service.UnmarshalBody(r, &inputWh); err != nil {
			return err
		}

		whs, err := workerhook.LoadAllByProjectIntegrationID(ctx, tx, integ.ID)
		if err != nil {
			return err
		}

		for i := range whs {
			if err := workerhook.DeleteByID(ctx, tx, whs[i].ID); err != nil {
				return err
			}
		}

		for i := range inputWh {
			wh := &inputWh[i]
			wh.ID = 0
			wh.ProjectIntegrationModelID = integ.ID
			if err := workerhook.Insert(ctx, tx, wh); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, inputWh, http.StatusOK)
	}
}

func (api *API) getProjectIntegrationWorkerHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		idS := vars["id"]
		id, err := strconv.ParseInt(idS, 10, 64)
		if err != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
		if err != nil {
			return err
		}

		wh, err := workerhook.LoadByID(ctx, api.mustDB(), id)
		if err != nil {
			return err
		}

		if wh.ProjectIntegrationModelID != integ.ID {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		return service.WriteJSON(w, wh, http.StatusOK)
	}
}

func (api *API) putProjectIntegrationWorkerHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		idS := vars["id"]
		id, err := strconv.ParseInt(idS, 10, 64)
		if err != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
		if err != nil {
			return err
		}

		wh, err := workerhook.LoadByID(ctx, api.mustDB(), id)
		if err != nil {
			return err
		}

		if wh.ProjectIntegrationModelID != integ.ID {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		var inputWh sdk.WorkerHookProjectIntegrationModel
		if err := service.UnmarshalBody(r, &inputWh); err != nil {
			return err
		}

		wh.Disable = inputWh.Disable
		wh.Configuration = inputWh.Configuration

		if err := workerhook.Update(ctx, api.mustDB(), wh); err != nil {
			return err
		}

		return service.WriteJSON(w, wh, http.StatusOK)
	}
}
