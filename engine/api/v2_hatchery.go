package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	hatch_auth "github.com/ovh/cds/engine/api/authentication/hatchery"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postHatcheryHeartbeatHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHatchery),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			hatcheryAuthConsumer := getHatcheryConsumer(ctx)
			h, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatcheryAuthConsumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return err
			}

			var mon sdk.MonitoringStatus
			if err := service.UnmarshalBody(req, &mon); err != nil {
				return err
			}

			// Update status to warn if service version != api version
			for i := range mon.Lines {
				if mon.Lines[i].Component == "Version" {
					if sdk.VERSION != mon.Lines[i].Value {
						mon.Lines[i].Status = sdk.MonitoringStatusWarn
					} else {
						mon.Lines[i].Status = sdk.MonitoringStatusOK
					}
					break
				}
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			h.LastHeartbeat = time.Now()
			if err := hatchery.Update(ctx, tx, h); err != nil {
				return err
			}
			var sessionID string
			if a := getAuthSession(ctx); a != nil {
				sessionID = a.ID
			}
			hs := sdk.HatcheryStatus{
				HatcheryID: h.ID,
				SessionID:  sessionID,
				Status:     mon,
			}
			if err := hatchery.UpsertStatus(ctx, tx, h.ID, &hs); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return nil
		}
}

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
	return service.RBAC(api.globalHatcheryManage),
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

			c, err := authentication.NewConsumerHatchery(ctx, tx, h)
			if err != nil {
				return err
			}
			jwsToken, err := hatch_auth.NewSigninConsumerToken(c)
			if err != nil {
				return err
			}
			h.Token = jwsToken

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, h, http.StatusCreated)
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
	return service.RBAC(api.globalHatcheryManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			hatcheryIdentifier := vars["hatcheryIdentifier"]

			hatch, err := api.getHatcheryByIdentifier(ctx, hatcheryIdentifier)
			if err != nil {
				return err
			}

			hatcheryPermission, err := rbac.LoadRBACByHatcheryID(ctx, api.mustDB(), hatch.ID)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			// Remove all permissions on this hatchery
			rbacHatcheries := make([]sdk.RBACHatchery, 0)
			for _, h := range hatcheryPermission.Hatcheries {
				if h.HatcheryID != hatch.ID {
					rbacHatcheries = append(rbacHatcheries, h)
				}
			}
			hatcheryPermission.Hatcheries = rbacHatcheries

			if hatcheryPermission.IsEmpty() {
				if err := rbac.Delete(ctx, tx, *hatcheryPermission); err != nil {
					return err
				}
			} else {
				if err := rbac.Update(ctx, tx, hatcheryPermission); err != nil {
					return err
				}
			}

			if err := hatchery.Delete(tx, hatch.ID); err != nil {
				return err
			}
			return sdk.WithStack(tx.Commit())
		}
}
