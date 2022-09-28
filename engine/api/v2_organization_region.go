package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postAllowRegionOnOrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.OrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			orgaIdentifier := vars["organizationIdentifier"]

			orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
			if err != nil {
				return err
			}
			var reg sdk.Region
			if err := service.UnmarshalBody(req, &reg); err != nil {
				return err
			}

			regionIdentifier := reg.ID
			if reg.ID == "" {
				regionIdentifier = reg.Name
			}
			if regionIdentifier == "" {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing region identifier")
			}

			regDB, err := api.getRegionByIdentifier(ctx, regionIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			orgReg := sdk.OrganizationRegion{
				RegionID:       regDB.ID,
				OrganizationID: orga.ID,
			}

			if err := organization.InsertOrganizationRegion(ctx, tx, &orgReg); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (api *API) getListRegionAllowedOnIrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			orgaIdentifier := vars["organizationIdentifier"]

			orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
			if err != nil {
				return err
			}

			regionIDs, err := organization.LoadRegionIDs(ctx, api.mustDB(), orga.ID)
			if err != nil {
				return err
			}

			regions, err := region.LoadRegionByIDs(ctx, api.mustDB(), regionIDs)
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, regions, http.StatusOK)
		}
}

func (api *API) deleteRegionFromOrganizationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.OrganizationManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			orgaIdentifier := vars["organizationIdentifier"]
			regIdentifier := vars["regionIdentifier"]

			orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
			if err != nil {
				return err
			}

			reg, err := api.getRegionByIdentifier(ctx, regIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}

			if err := organization.DeleteOrganizationRegion(tx, orga.ID, reg.ID); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteMarshal(w, req, nil, http.StatusNoContent)
		}
}
