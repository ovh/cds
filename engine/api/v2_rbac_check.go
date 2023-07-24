package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getCheckSessionProjectAccessHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.checkSessionPermission),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var checkRequest sdk.CheckProjectAccess
			if err := service.UnmarshalRequest(ctx, req, &checkRequest); err != nil {
				return sdk.WithStack(err)
			}

			session, err := authentication.LoadSessionByID(ctx, api.mustDBWithCtx(ctx), checkRequest.SessionID)
			if err != nil {
				return err
			}

			consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
			}
			if consumer.Disabled {
				return sdk.WrapError(sdk.ErrUnauthorized, "consumer (%s) is disabled", consumer.ID)
			}

			switch consumer.Type {
			case sdk.ConsumerHatchery:
				return sdk.WrapError(sdk.ErrUnauthorized, "hatchery consumer cannot access proejct %s", checkRequest.ProjectKey)
			default:
				userConsumer, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), session.ConsumerID)
				if err != nil {
					return err
				}
				hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), checkRequest.Role, userConsumer.AuthConsumerUser.AuthentifiedUserID, checkRequest.ProjectKey)
				if err != nil {
					return err
				}
				if !hasRole {
					return sdk.WrapError(sdk.ErrUnauthorized, "user %s doesn't have the right on project %s", userConsumer.AuthConsumerUser.AuthentifiedUser.Username, checkRequest.ProjectKey)
				}
			}
			return nil
		}
}
