package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserTokenListHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tokens, err := token.LoadTokens(api.mustDB(), deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load group for user %s", deprecatedGetUser(ctx).Username)
		}

		return service.WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) getUserTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		tok, err := token.LoadTokenWithGroup(api.mustDB(), vars["token"])
		if sdk.ErrorIs(err, sdk.ErrInvalidToken) {
			return sdk.ErrTokenNotFound
		}
		if err != nil {
			return sdk.WrapError(err, "Cannot load token for user %s", deprecatedGetUser(ctx).Username)
		}
		tok.Token = ""

		return service.WriteJSON(w, tok, http.StatusOK)
	}
}
