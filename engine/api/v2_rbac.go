package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
)

func (api *API) postImportRbacHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.PermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			force := service.FormBool(req, "force")

			var rbacRule sdk.RBAC
			if err := service.UnmarshalRequest(ctx, req, &rbacRule); err != nil {
				return err
			}

			existingRule, err := rbac.LoadRbacByName(ctx, api.mustDB(), rbacRule.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			if err := rbac.FillWithIDs(ctx, api.mustDB(), &rbacRule); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if existingRule != nil && !force {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "unable to override existing permission")
			}
			if existingRule != nil {
				if err := rbac.Delete(ctx, tx, *existingRule); err != nil {
					return err
				}
			}

			if err := rbac.Insert(ctx, tx, &rbacRule); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}
