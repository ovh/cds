package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getRegionByIdentifier(ctx context.Context, regionIdentifier string) (*sdk.Region, error) {
	var reg *sdk.Region
	var err error
	if sdk.IsValidUUID(regionIdentifier) {
		reg, err = region.LoadRegionByID(ctx, api.mustDB(), regionIdentifier)
	} else {
		reg, err = region.LoadRegionByName(ctx, api.mustDB(), regionIdentifier)
	}
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (api *API) postRegionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.RegionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var reg sdk.Region
			if err := service.UnmarshalBody(req, &reg); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := region.Insert(ctx, tx, &reg); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (api *API) getRegionsHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			regions, err := region.LoadAllRegions(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, regions, http.StatusOK)
		}
}

func (api *API) getRegionHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			regionIdentifier := vars["regionIdentifier"]

			reg, err := api.getRegionByIdentifier(ctx, regionIdentifier)
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, reg, http.StatusOK)
		}
}

func (api *API) deleteRegionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.RegionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			regionIdentifier := vars["regionIdentifier"]

			reg, err := api.getRegionByIdentifier(ctx, regionIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := region.Delete(tx, reg.ID); err != nil {
				return err
			}
			return sdk.WithStack(tx.Commit())
		}
}
