package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getExternalServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		typeService := vars["type"]

		for _, s := range api.Config.Services {
			if s.Type == typeService {
				return service.WriteJSON(w, s, http.StatusOK)
			}
		}
		return sdk.WrapError(sdk.ErrNotFound, "getExternalServiceHandler> Service %s not found", typeService)
	}
}

func (api *API) postServiceRegisterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srv := &sdk.Service{}
		if err := service.UnmarshalBody(r, srv); err != nil {
			return sdk.WithStack(err)
		}

		// Load token
		t, errL := token.LoadToken(api.mustDB(), srv.Token)
		if errL != nil {
			return sdk.NewError(sdk.ErrUnauthorized, sdk.WrapError(errL, "Cannot register service"))
		}

		//Service must be with a sharedinfra group token
		// except for hatchery: users can start hatchery with their group
		if t.GroupID != group.SharedInfraGroup.ID && srv.Type != services.TypeHatchery {
			return sdk.WrapError(sdk.ErrUnauthorized, "Cannot register service for group %d with service %s", t.GroupID, srv.Type)
		}

		srv.GroupID = &t.GroupID
		srv.IsSharedInfra = srv.GroupID == &group.SharedInfraGroup.ID
		srv.Uptodate = srv.Version == sdk.VERSION
		for i := range srv.MonitoringStatus.Lines {
			s := &srv.MonitoringStatus.Lines[i]
			if s.Component == "Version" {
				if sdk.VERSION != s.Value {
					s.Status = sdk.MonitoringStatusWarn
				} else {
					s.Status = sdk.MonitoringStatusOK
				}
				break
			}
		}

		//Insert or update the service
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		//Try to find the service, and keep; else generate a new one
		oldSrv, errOldSrv := services.FindByName(tx, srv.Name)
		if oldSrv != nil {
			srv.Hash = oldSrv.Hash
			srv.ID = oldSrv.ID
		} else if sdk.ErrorIs(errOldSrv, sdk.ErrNotFound) {
			srv.Hash = sdk.UUID()
		} else {
			return sdk.WithStack(errOldSrv)
		}

		srv.LastHeartbeat = time.Now()
		srv.Token = ""

		if oldSrv != nil {
			if err := services.Update(tx, srv); err != nil {
				return sdk.WrapError(err, "Unable to update service %s", srv.Name)
			}
		} else {
			if err := services.Insert(tx, srv); err != nil {
				return sdk.WrapError(err, "Unable to insert service %s", srv.Name)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) serviceAPIHeartbeat(c context.Context) {
	tick := time.NewTicker(30 * time.Second).C

	hash := sdk.UUID()
	// first call
	api.serviceAPIHeartbeatUpdate(c, api.mustDB(), hash)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting serviceAPIHeartbeat: %v", c.Err())
				return
			}
		case <-tick:
			api.serviceAPIHeartbeatUpdate(c, api.mustDB(), hash)
		}
	}
}

func (api *API) serviceAPIHeartbeatUpdate(c context.Context, db *gorp.DbMap, hash string) {
	tx, err := db.Begin()
	if err != nil {
		log.Error("serviceAPIHeartbeat> error on repo.Begin:%v", err)
		return
	}
	defer tx.Rollback() // nolint

	srv := &sdk.Service{
		Name:             event.GetCDSName(),
		MonitoringStatus: api.Status(),
		Hash:             hash,
		LastHeartbeat:    time.Now(),
		Type:             services.TypeAPI,
		Config:           api.Config,
	}
	if group.SharedInfraGroup != nil {
		srv.GroupID = &group.SharedInfraGroup.ID
	}

	//Try to find the service, and keep; else generate a new one
	oldSrv, errOldSrv := services.FindByName(tx, srv.Name)
	if errOldSrv != nil && !sdk.ErrorIs(errOldSrv, sdk.ErrNotFound) {
		log.Error("serviceAPIHeartbeat> Unable to find by name:%v", errOldSrv)
		return
	}

	if oldSrv != nil {
		srv.ID = oldSrv.ID
		if err := services.Update(tx, srv); err != nil {
			log.Error("serviceAPIHeartbeat> Unable to update service %s: %v", srv.Name, err)
			return
		}
	} else {
		if err := services.Insert(tx, srv); err != nil {
			log.Error("serviceAPIHeartbeat> Unable to insert service %s: %v", srv.Name, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("serviceAPIHeartbeat> error on repo.Commit: %v", err)
		return
	}
}
