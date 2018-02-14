package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
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
		if t.GroupID != group.SharedInfraGroup.ID {
			return sdk.WrapError(sdk.ErrUnauthorized, "postServiceRegisterHandler> Cannot register service")
		}

		//Insert or update the service
		repo := services.NewRepository(api.mustDB, api.Cache)
		if err := repo.Begin(); err != nil {
			return sdk.WrapError(err, "postServiceRegisterHandler")
		}

		//Try to find the service, and keep; else generate a new one
		oldSrv, errOldSrv := repo.Find(srv.Name)
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

		defer repo.Rollback()

		if oldSrv != nil {
			if err := repo.Update(srv); err != nil {
				return sdk.WrapError(err, "postServiceRegisterHandler> Unable to update service %s", srv.Name)
			}
		} else {
			if err := repo.Insert(srv); err != nil {
				return sdk.WrapError(err, "postServiceRegisterHandler> Unable to insert service %s", srv.Name)
			}
		}

		if err := repo.Commit(); err != nil {
			return sdk.WrapError(err, "postServiceRegisterHandler")
		}

		return WriteJSON(w, srv, http.StatusOK)
	}
}
