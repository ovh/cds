package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getOrganizationByIdentifier(ctx context.Context, orgaIdentifier string) (*sdk.Organization, error) {
	var orga *sdk.Organization
	var err error
	if sdk.IsValidUUID(orgaIdentifier) {
		orga, err = organization.LoadOrganizationByID(ctx, api.mustDB(), orgaIdentifier)
	} else {
		orga, err = organization.LoadOrganizationByName(ctx, api.mustDB(), orgaIdentifier)
	}
	if err != nil {
		return nil, err
	}
	return orga, nil
}

func (api *API) postOrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalOrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var org sdk.Organization
			if err := service.UnmarshalBody(req, &org); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := organization.Insert(ctx, tx, &org); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (api *API) getOrganizationsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalOrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			orgas, err := organization.LoadOrganizations(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, orgas, http.StatusOK)
		}
}

func (api *API) getOrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalOrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			orgaIdentifier := vars["organizationIdentifier"]

			orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, orga, http.StatusOK)
		}
}

func (api *API) deleteOrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalOrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			orgaIdentifier := vars["organizationIdentifier"]

			orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := organization.Delete(tx, orga.ID); err != nil {
				return err
			}
			return sdk.WithStack(tx.Commit())
		}
}
