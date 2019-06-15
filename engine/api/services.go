package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
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
		pubKey, err := jws.ExportPublicKey(authentication.GetSigningKey())
		if err != nil {
			return sdk.WrapError(err, "Unable to export public signing key")
		}

		srv := &sdk.Service{}
		if err := service.UnmarshalBody(r, srv); err != nil {
			return sdk.WithStack(err)
		}

		srv.ConsumerID = getAPIConsumer(ctx).ID

		//Service must be with a sharedinfra group token
		// except for hatchery: users can start hatchery with their group
		if !isGroupMember(ctx, group.SharedInfraGroup) && srv.Type != services.TypeHatchery {
			return sdk.WrapError(sdk.ErrForbidden, "Cannot register service for token %s with service %s", getAPIConsumer(ctx).ID, srv.Type)
		}
		// TODO: for hatcheries, the user who created the used token must be admin of all groups in the token
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
		oldSrv, errOldSrv := services.GetByName(ctx, tx, srv.Name)
		if oldSrv != nil {
			srv.ID = oldSrv.ID
		} else if !sdk.ErrorIs(errOldSrv, sdk.ErrNotFound) {
			return sdk.WithStack(errOldSrv)
		}

		srv.LastHeartbeat = time.Now()

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

		encodedPubKey := base64.StdEncoding.EncodeToString(pubKey)
		w.Header().Set("X-Api-Pub-Signing-Key", encodedPubKey)

		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) postServiceUnregisterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}

func (api *API) serviceAPIHeartbeat(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second).C

	var u = sdk.AuthentifiedUser{} // TODO: fake user for the API ?

	// first call
	api.serviceAPIHeartbeatUpdate(ctx, api.mustDB(), u)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error("Exiting serviceAPIHeartbeat: %v", ctx.Err())
				return
			}
		case <-tick:
			api.serviceAPIHeartbeatUpdate(ctx, api.mustDB(), u)
		}
	}
}

func (api *API) serviceAPIHeartbeatUpdate(ctx context.Context, db *gorp.DbMap, authUser sdk.AuthentifiedUser) {
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
			Name:       event.GetCDSName(),
			Type:       services.TypeAPI,
			Config:     srvConfig,
			Maintainer: authUser,
		},
		MonitoringStatus: api.Status(),
		LastHeartbeat:    time.Now(),
	}

	//Try to find the service, and keep; else generate a new one
	oldSrv, errOldSrv := services.GetByName(ctx, tx, srv.Name)
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
