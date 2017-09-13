package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/group"
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

		return WriteJSON(w, r, srv, http.StatusOK)
	}
}
