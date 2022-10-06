package api

import (
	"context"
	"github.com/ovh/cds/engine/api/hatchery"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getHatcheryByIdentifier(ctx context.Context, hatcheryIdentifier string) (*sdk.Hatchery, error) {
	var h *sdk.Hatchery
	var err error
	if sdk.IsValidUUID(hatcheryIdentifier) {
		h, err = hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatcheryIdentifier)
	} else {
		h, err = hatchery.LoadHatcheryByName(ctx, api.mustDB(), hatcheryIdentifier)
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (api *API) postHatcheryHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.GlobalHatcheryManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var h sdk.Hatchery
			if err := service.UnmarshalBody(req, &h); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := hatchery.Insert(ctx, tx, &h); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (api *API) getHatcheriesHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			hatcheries, err := hatchery.LoadHatcheries(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, hatcheries, http.StatusOK)
		}
}

func (api *API) getHatcheryHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			hatcheryIdentifier := vars["hatcheryIdentifier"]

			reg, err := api.getHatcheryByIdentifier(ctx, hatcheryIdentifier)
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, reg, http.StatusOK)
		}
}

func (api *API) deleteHatcheryHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.GlobalHatcheryManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			hatcheryIdentifier := vars["hatcheryIdentifier"]

			reg, err := api.getHatcheryByIdentifier(ctx, hatcheryIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := hatchery.Delete(tx, reg.ID); err != nil {
				return err
			}
			return sdk.WithStack(tx.Commit())
		}
}
