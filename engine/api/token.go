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

		tokenPostInfos := struct {
			Expiration  string `json:"expiration"`
			Description string `json:"description"`
		}{}
		if err := UnmarshalBody(r, &tokenPostInfos); err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot unmarshal")
		}

		if expiration == "" {
			expiration = tokenPostInfos.Expiration
		}

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
		if err := token.InsertToken(api.mustDB(), g.ID, tk, exp, tokenPostInfos.Description, getUser(ctx).Fullname); err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot insert new key")
		}
		token := sdk.Token{
			GroupID:     g.ID,
			Token:       tk,
			Expiration:  exp,
			Created:     now,
			Description: tokenPostInfos.Description,
			Creator:     getUser(ctx).Fullname,
		}
		return WriteJSON(w, r, token, http.StatusOK)
	}
}

func (api *API) getGroupTokenListHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]

		isAdmin, errA := group.IsGroupAdmin(api.mustDB(), groupName, getUser(ctx).ID)
		if errA != nil {
			return sdk.WrapError(errA, "getGroupTokenListHandler> cannot load group admin information '%s'", groupName)
		}

		if !isAdmin {
			return WriteJSON(w, r, nil, http.StatusForbidden)
		}

		tokens, err := group.LoadTokens(api.mustDB(), groupName)
		if err != nil {
			return sdk.WrapError(err, "getGroupTokenListHandler> cannot load group '%s'", groupName)
		}

		return WriteJSON(w, r, tokens, http.StatusOK)
	}
}

func (api *API) getUserTokenListHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tokens, err := token.LoadTokens(api.mustDB(), getUser(ctx).ID)
		if err != nil {
			return sdk.WrapError(err, "getUserTokenListHandler> cannot load group for user %s", getUser(ctx).Username)
		}

		return WriteJSON(w, r, tokens, http.StatusOK)
	}
}

func (api *API) deleteTokenHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		tokenID, errT := requestVarInt(r, "tokenid")
		if errT != nil {
			return sdk.WrapError(errT, "deleteTokenHandler> token id is not a number '%s'", vars["tokenid"])
		}

		if err := token.Delete(api.mustDB(), tokenID); err != nil {
			return sdk.WrapError(err, "deleteTokenHandler> cannot load delete token id %d", tokenID)
		}

		return WriteJSON(w, r, nil, http.StatusOK)
	}
}
