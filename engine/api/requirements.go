package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/worker"

	"github.com/ovh/cds/sdk"
)

func (api *API) getRequirementTypesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, sdk.AvailableRequirementsType, http.StatusOK)
	}
}

func (api *API) getRequirementTypeValuesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		reqType := vars["type"]

		switch reqType {
		case sdk.BinaryRequirement:
			req, err := action.LoadAllBinaryRequirements(api.mustDB(ctx))
			if err != nil {
				return sdk.WrapError(err, "getRequirementTypeValuesHandler> Cannot load binary requirements")
			}
			return WriteJSON(w, req.Values(), http.StatusOK)

		case sdk.ModelRequirement:
			models, err := worker.LoadWorkerModelsByUser(api.mustDB(ctx), getUser(ctx))
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
			return WriteJSON(w, modelsAsRequirements.Values(), http.StatusOK)

		case sdk.OSArchRequirement:
			return WriteJSON(w, sdk.OSArchRequirementValues.Values(), http.StatusOK)

		default:
			return nil

		}
	}
}
