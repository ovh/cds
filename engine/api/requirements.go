package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"

	"github.com/ovh/cds/sdk"
)

func (api *API) getRequirementTypesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailableRequirementsType, http.StatusOK)
	}
}

func (api *API) getRequirementTypeValuesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		reqType := vars["type"]

		switch reqType {
		case sdk.BinaryRequirement:
			req, err := action.LoadAllBinaryRequirements(api.mustDB())
			if err != nil {
				return sdk.WrapError(err, "getRequirementTypeValuesHandler> Cannot load binary requirements")
			}
			return service.WriteJSON(w, req.Values(), http.StatusOK)

		case sdk.ModelRequirement:
			models, err := worker.LoadWorkerModelsByUser(api.mustDB(), api.Cache, getUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "getRequirementTypeValuesHandler> Cannot load worker models")
			}
			modelsAsRequirements := make(sdk.RequirementList, len(models))
			for i, m := range models {
				modelsAsRequirements[i] = sdk.Requirement{
					Name:  m.Name,
					Type:  sdk.ModelRequirement,
					Value: m.Name,
				}
			}
			return service.WriteJSON(w, modelsAsRequirements.Values(), http.StatusOK)

		case sdk.OSArchRequirement:
			return service.WriteJSON(w, sdk.OSArchRequirementValues.Values(), http.StatusOK)

		default:
			return nil

		}
	}
}
