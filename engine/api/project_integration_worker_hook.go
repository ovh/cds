package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/workerhook"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectIntegrationWorkerHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
		if err != nil {
			return err
		}

		wh, err := workerhook.LoadByProjectIntegrationID(ctx, api.mustDB(), integ.ID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, wh, http.StatusOK)
	}
}

func (api *API) postProjectIntegrationWorkerHookHandler() service.Handler {
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

		var inputWh sdk.WorkerHookProjectIntegrationModel
		if err := service.UnmarshalBody(r, &inputWh); err != nil {
			return err
		}

		inputWh.ProjectIntegrationID = integ.ID

		wh, err := workerhook.LoadByProjectIntegrationID(ctx, tx, integ.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		if wh == nil {
			if err := workerhook.Insert(ctx, tx, &inputWh); err != nil {
				return err
			}
		} else {
			inputWh.ID = wh.ID
			if err := workerhook.Update(ctx, tx, &inputWh); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, inputWh, http.StatusOK)
	}
}
