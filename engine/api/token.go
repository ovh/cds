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

		g, err := group.LoadGroup(api.mustDB(ctx), groupName)
		if err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot load group '%s'", groupName)
		}

		tk, err := token.GenerateToken()
		if err != nil {
			return sdk.WrapError(err, "generateTokenHandler: cannot generate key")
		}
		now := time.Now()
		if err := token.InsertToken(api.mustDB(ctx), g.ID, tk, exp, tokenPostInfos.Description, getUser(ctx).Fullname); err != nil {
			return sdk.WrapError(err, "generateTokenHandler> cannot insert new key")
		}
		token := sdk.Token{
			GroupID:     g.ID,
			Token:       tk,
			Expiration:  exp,
			Created:     now,
			Description: tokenPostInfos.Description,
			Creator:     getUser(ctx).Fullname,
			GroupName:   groupName,
		}
		return WriteJSON(w, token, http.StatusOK)
	}
}

func (api *API) getGroupTokenListHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]

		isAdmin, errA := group.IsGroupAdmin(api.mustDB(ctx), groupName, getUser(ctx).ID)
		if errA != nil {
			return sdk.WrapError(errA, "getGroupTokenListHandler> cannot load group admin information '%s'", groupName)
		}

		if !isAdmin {
			return WriteJSON(w, nil, http.StatusForbidden)
		}

		tokens, err := group.LoadTokens(api.mustDB(ctx), groupName)
		if err != nil {
			return sdk.WrapError(err, "getGroupTokenListHandler> cannot load group '%s'", groupName)
		}

		return WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) deleteTokenHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]
		tokenID, errT := requestVarInt(r, "tokenid")
		if errT != nil {
			return sdk.WrapError(errT, "deleteTokenHandler> token id is not a number '%s'", vars["tokenid"])
		}

		isGroupAdmin, errA := group.IsGroupAdmin(api.mustDB(ctx), groupName, getUser(ctx).ID)
		if errA != nil {
			return sdk.WrapError(errT, "deleteTokenHandler> cannot load group admin for user %s", getUser(ctx).Username)
		}

		if !isGroupAdmin {
			return WriteJSON(w, nil, http.StatusForbidden)
		}

		if err := token.Delete(api.mustDB(ctx), tokenID); err != nil {
			return sdk.WrapError(err, "deleteTokenHandler> cannot load delete token id %d", tokenID)
		}

		return WriteJSON(w, nil, http.StatusOK)
	}
}
