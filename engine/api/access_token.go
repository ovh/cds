package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/group"

	"github.com/ovh/cds/engine/service"
)

// Manage access token handlers

// postNewAccessTokenHandler create a new specific accesstoken with a specific scope (list of groups)
// the JWT token is send through a header X-CDS-JWTTOKEN
func (api *API) postNewAccessTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// the groupIDs are the scope of the requested token
		var accessTokenRequest sdk.AccessTokenRequest
		if err := service.UnmarshalBody(r, &accessTokenRequest); err != nil {
			return sdk.WithStack(err)
		}

		// if the scope is empty, raise an error
		if len(accessTokenRequest.GroupsIDs) == 0 {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		grantedUser := getGrantedUser(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		defer tx.Rollback() // nolint

		allGroups, err := group.LoadGroupByAdmin(tx, grantedUser.OnBehalfOf.ID)
		if err != nil {
			return sdk.WithStack(err)
		}

		// check that provided group is among the allGroups slice
		// a user can create a token associated to a group only if he is admin of this group
		var scopeGroup = make([]sdk.Group, 0, len(accessTokenRequest.GroupsIDs))
		for _, groupID := range accessTokenRequest.GroupsIDs {
			var found bool
			for _, g := range allGroups {
				if g.ID == groupID {
					found = true
					scopeGroup = append(scopeGroup, g)
					break
				}
			}
			if !found {
				return sdk.WrapError(sdk.ErrWrongRequest, "group %d not found", groupID)
			}
		}

		var expiration *time.Time
		if accessTokenRequest.ExpirationDelaySecond > 0 {
			t := time.Now().Add(time.Duration(accessTokenRequest.ExpirationDelaySecond) * time.Second)
			expiration = &t
		}

		// Create the token
		token, jwttoken, err := accesstoken.New(grantedUser.OnBehalfOf, scopeGroup, accessTokenRequest.Origin, accessTokenRequest.Description, expiration)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Insert the token
		if err := accesstoken.Insert(tx, &token); err != nil {
			return sdk.WithStack(err)
		}

		// Commit the token
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// Set the JWT token as a header
		log.Debug("token.postNewAccessTokenHandler> X-CDS-JWT:%s", jwttoken[:12])
		w.Header().Add("X-CDS-JWT", jwttoken)

		return service.WriteJSON(w, token, http.StatusCreated)
	}
}

// putRegenAccessTokenHandler create a new specific accesstoken with a specific scope (list of groups)
// the JWT token is send through a header X-CDS-JWTTOKEN
func (api *API) putRegenAccessTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		// the groupIDs are the scope of the requested token
		var accessTokenRequest sdk.AccessTokenRequest
		if err := service.UnmarshalBody(r, &accessTokenRequest); err != nil {
			return sdk.WithStack(err)
		}

		// if the scope is empty, raise an error
		if len(accessTokenRequest.GroupsIDs) == 0 {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		t, err := accesstoken.FindByID(tx, id)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Create the token
		jwttoken, err := accesstoken.Regen(&t)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Set the JWT token as a header
		w.Header().Add("X-CDS-JWT", jwttoken)

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (api *API) getAccessTokenByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.WithStack(err)
		}

		tokens, err := accesstoken.FindAllByUser(api.mustDB(), id)
		if err != nil {
			return sdk.WithStack(err)
		}
		return service.WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) getAccessTokenByGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.WithStack(err)
		}

		tokens, err := accesstoken.FindAllByGroup(api.mustDB(), id)
		if err != nil {
			return sdk.WithStack(err)
		}
		return service.WriteJSON(w, tokens, http.StatusOK)
	}
}
