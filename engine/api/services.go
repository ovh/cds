package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postServiceRegisterHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srv := &sdk.Service{}
		if err := UnmarshalBody(r, srv); err != nil {
			return sdk.WrapError(err, "postServiceRegisterHandler")
		}

		// Load token
		t, errL := token.LoadToken(api.mustDB(), srv.Token)
		if errL != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "postServiceRegisterHandler> Cannot register service: %v", errL)
		}

		//Service must be with a sharedinfra group token
		// except for hatchery: users can start hatchery with their group
		if t.GroupID != group.SharedInfraGroup.ID && srv.Type != services.TypeHatchery {
			return sdk.WrapError(sdk.ErrUnauthorized, "postServiceRegisterHandler> Cannot register service for group %d with service %s", t.GroupID, srv.Type)
		}

		//Insert or update the service
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "postServiceRegisterHandler")
		}
		defer tx.Rollback()

		//Try to find the service, and keep; else generate a new one
		oldSrv, errOldSrv := services.FindByName(tx, srv.Name)
		if oldSrv != nil {
			srv.Hash = oldSrv.Hash
		} else if errOldSrv == sdk.ErrNotFound {
			//Generate a hash
			hash, errsession := sessionstore.NewSessionKey()
			if errsession != nil {
				return sdk.WrapError(errsession, "postServiceRegisterHandler> Unable to create session")
			}
			srv.Hash = string(hash)
		} else {
			return sdk.WrapError(errOldSrv, "postServiceRegisterHandler")
		}

		srv.LastHeartbeat = time.Now()
		srv.Token = ""

		if oldSrv != nil {
			if err := services.Update(tx, srv); err != nil {
				return sdk.WrapError(err, "postServiceRegisterHandler> Unable to update service %s", srv.Name)
			}
		} else {
			if err := services.Insert(tx, srv); err != nil {
				return sdk.WrapError(err, "postServiceRegisterHandler> Unable to insert service %s", srv.Name)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postServiceRegisterHandler")
		}

		return WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) serviceAPIHeartbeat(c context.Context) {
	tick := time.NewTicker(30 * time.Second).C

	hash, errsession := sessionstore.NewSessionKey()
	if errsession != nil {
		log.Error("serviceAPIHeartbeat> Unable to create session:%v", errsession)
		return
	}

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

func (api *API) serviceAPIHeartbeatUpdate(c context.Context, db *gorp.DbMap, hash sessionstore.SessionKey) {
	tx, err := db.Begin()
	if err != nil {
		log.Error("serviceAPIHeartbeat> error on repo.Begin:%v", err)
		return
	}
	defer tx.Rollback() // nolint

	srv := &sdk.Service{
		Name:             event.GetCDSName(),
		MonitoringStatus: api.Status(),
		Hash:             string(hash),
		LastHeartbeat:    time.Now(),
		Type:             services.TypeAPI,
	}

	//Try to find the service, and keep; else generate a new one
	oldSrv, errOldSrv := services.FindByName(tx, srv.Name)
	if errOldSrv != nil && errOldSrv != sdk.ErrNotFound {
		log.Error("serviceAPIHeartbeat> Unable to find by name:%v", errOldSrv)
		return
	}

	if oldSrv != nil {
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
