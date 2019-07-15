package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/group"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/services"
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
		var srv sdk.Service
		if err := service.UnmarshalBody(r, &srv); err != nil {
			return sdk.WithStack(err)
		}

		//Service must be with a sharedinfra group token
		// except for hatchery: users can start hatchery with their group
		if !isGroupMember(ctx, group.SharedInfraGroup) && srv.Type != services.TypeHatchery {
			return sdk.WrapError(sdk.ErrForbidden, "Cannot register service for token %s with service %s", getAPIConsumer(ctx).ID, srv.Type)
		}

		// For hatcheries, the user who created the used token must be admin of all groups in the token
		if srv.Type == services.TypeHatchery {
			gAdmins, err := group.LoadGroupByAdmin(api.mustDB(), getAPIConsumer(ctx).AuthentifiedUser.OldUserStruct.ID)
			if err != nil {
				return err
			}
			groupsAdminIDs := sdk.Groups(gAdmins).ToIDs()
			for _, gID := range getAPIConsumer(ctx).GetGroupIDs() {
				if !sdk.IsInInt64Array(gID, groupsAdminIDs) {
					return sdk.WrapError(sdk.ErrForbidden, "Cannot register service for token %s with service %s", getAPIConsumer(ctx).ID, srv.Type)
				}
			}
		}

		srv.Uptodate = srv.Version == sdk.VERSION
		srv.ConsumerID = &getAPIConsumer(ctx).ID

		//Insert or update the service
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		//Try to find the service, and keep; else generate a new one
		oldSrv, errOldSrv := services.LoadByName(ctx, tx, srv.Name)
		if oldSrv != nil {
			srv.ID = oldSrv.ID
			if err := services.Update(tx, &srv); err != nil {
				return sdk.WithStack(err)
			}
		} else if !sdk.ErrorIs(errOldSrv, sdk.ErrNotFound) {
			log.Error("unable to find service by name %s: %v", srv.Name, errOldSrv)
			return sdk.WithStack(errOldSrv)
		} else {
			srv.Maintainer = *getAPIConsumer(ctx).AuthentifiedUser
			if err := services.Insert(tx, &srv); err != nil {
				return sdk.WithStack(err)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) postServiceHearbeatHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var mon sdk.MonitoringStatus
		if err := service.UnmarshalBody(r, &mon); err != nil {
			return sdk.WithStack(err)
		}

		for i := range mon.Lines {
			s := &mon.Lines[i]
			if s.Component == "Version" {
				if sdk.VERSION != s.Value {
					s.Status = sdk.MonitoringStatusWarn
				} else {
					s.Status = sdk.MonitoringStatusOK
				}
				break
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		srv, err := services.LoadByConsumerID(ctx, tx, getAPIConsumer(ctx).ID)
		if err != nil {
			return err

		}

		srv.LastHeartbeat = time.Now()
		srv.MonitoringStatus = mon

		if err := services.Update(tx, srv); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) serviceAPIHeartbeat(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second).C

	// first call
	api.serviceAPIHeartbeatUpdate(ctx, api.mustDB())

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error("Exiting serviceAPIHeartbeat: %v", ctx.Err())
				return
			}
		case <-tick:
			api.serviceAPIHeartbeatUpdate(ctx, api.mustDB())
		}
	}
}

func (api *API) serviceAPIHeartbeatUpdate(ctx context.Context, db *gorp.DbMap) {
	tx, err := db.Begin()
	if err != nil {
		log.Error("serviceAPIHeartbeat> error on repo.Begin:%v", err)
		return
	}
	defer tx.Rollback() // nolint

	var srvConfig sdk.ServiceConfig
	b, _ := json.Marshal(api.Config)
	json.Unmarshal(b, &srvConfig) // nolint

	srv := &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:   event.GetCDSName(),
			Type:   services.TypeAPI,
			Config: srvConfig,
		},
		MonitoringStatus: api.Status(),
		LastHeartbeat:    time.Now(),
	}

	//Try to find the service, and keep; else generate a new one
	oldSrv, errOldSrv := services.LoadByName(ctx, tx, srv.Name)
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
