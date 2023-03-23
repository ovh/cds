package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/invopop/jsonschema"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getJsonSchemaHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			t := vars["type"]

			u := getUserConsumer(ctx)

			var schema *jsonschema.Schema
			switch t {
			case sdk.EntityTypeWorkerModel:
				schema = sdk.GetWorkerModelJsonSchema()
			case sdk.EntityTypeAction:
				var actionNames []string
				if u != nil {
					keys, err := rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
					if err != nil {
						return err
					}
					actionFullNames, err := entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, api.mustDB(), sdk.EntityTypeAction, keys)
					if err != nil {
						return nil
					}
					for _, an := range actionFullNames {
						actionNames = append(actionNames, fmt.Sprintf("%s/%s/%s/%s@%s", an.ProjectKey, an.VCSName, an.RepoName, an.Name, an.Branch))
					}
				}

				schema = sdk.GetActionJsonSchema(actionNames)
			}
			return service.WriteJSON(w, schema, http.StatusOK)
		}
}
