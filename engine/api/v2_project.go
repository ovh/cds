package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectV2AccessHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isCDNService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)

			projectKey := vars["projectKey"]
			itemType := vars["type"]

			if sdk.CDNItemType(itemType) == sdk.CDNTypeItemWorkerCache {
				return sdk.WrapError(sdk.ErrForbidden, "cdn is not enabled for this type %s", itemType)
			}

			sessionID := req.Header.Get(sdk.CDSSessionID)
			if sessionID == "" {
				return sdk.WrapError(sdk.ErrForbidden, "missing session id header")
			}

			session, err := authentication.LoadSessionByID(ctx, api.mustDBWithCtx(ctx), sessionID)
			if err != nil {
				return err
			}
			consumer, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), session.ConsumerID,
				authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
			}

			if consumer.Disabled {
				return sdk.WrapError(sdk.ErrUnauthorized, "consumer (%s) is disabled", consumer.ID)
			}

			maintainerOrAdmin := consumer.Maintainer() || consumer.Admin()
			canRead, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), sdk.ProjectRoleRead, consumer.AuthConsumerUser.AuthentifiedUserID, projectKey)
			if err != nil {
				return err
			}

			if maintainerOrAdmin || canRead {
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}
}
