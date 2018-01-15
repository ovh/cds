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
		return WriteJSON(w, r, sdk.AvailableRequirementsType, http.StatusOK)
	}
}

func (api *API) getRequirementTypeValuesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		reqType := vars["type"]

		switch reqType {
		case sdk.BinaryRequirement:
			req, err := action.LoadAllBinaryRequirements(api.mustDB())
			if err != nil {
				return sdk.WrapError(err, "getRequirementTypeValuesHandler> Cannot load binary requirements")
			}
			return WriteJSON(w, r, req.Values(), http.StatusOK)

		case sdk.ModelRequirement:
			models, err := worker.LoadWorkerModelsByUser(api.mustDB(), getUser(ctx))
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
			return WriteJSON(w, r, modelsAsRequirements.Values(), http.StatusOK)

		case sdk.OSArchRequirement:
			return WriteJSON(w, r, sdk.OSArchRequirementValues.Values(), http.StatusOK)

		default:
			return nil

		}
	}
}
