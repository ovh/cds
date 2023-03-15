package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/invopop/jsonschema"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getJsonSchemaHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			t := vars["type"]

			var schema *jsonschema.Schema
			switch t {
			case sdk.EntityTypeWorkerModel:
				schema = sdk.GetWorkerModelJsonSchema()
			case sdk.EntityTypeAction:
				schema = sdk.GetActionJsonSchema()
			}
			return service.WriteJSON(w, schema, http.StatusOK)
		}
}
