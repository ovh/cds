package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
)

// generateTokenHandler allows a user to generate a token associated to a group permission
// and used by worker to take action from API.
// User generating the token needs to be admin of given group
func (api *API) generateTokenHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]
		expiration := vars["expiration"]

		exp, err := sdk.ExpirationFromString(expiration)
		if err != nil {
			return sdk.WrapError(err, "generateTokenHandler> '%s'", expiration)
		}

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot load group '%s'", groupName)
		}

		tk, err := token.GenerateToken()
		if err != nil {
			return sdk.WrapError(err, "generateTokenHandler: cannot generate key")
		}
		now := time.Now()
		if err := token.InsertToken(api.mustDB(), g.ID, tk, exp); err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot insert new key")
		}
		token := sdk.Token{
			GroupID:    g.ID,
			Token:      tk,
			Expiration: exp,
			Created:    now,
		}
		return WriteJSON(w, r, token, http.StatusOK)
	}
}
