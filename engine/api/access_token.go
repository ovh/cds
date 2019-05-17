package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Manage access token handlers

func (api *API) createNewAccessToken(u sdk.AuthentifiedUser, accessTokenRequest sdk.AccessTokenRequest) (token sdk.AccessToken, jwttoken string, err error) {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	// TODO: after migration from user.groups to authentifiedUser.groups
	oldUser, err := user.GetDeprecatedUser(api.mustDB(), &u)
	if err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	allGroups, err := group.LoadGroupByAdmin(tx, oldUser.ID)
	if err != nil {
		return token, jwttoken, sdk.WithStack(err)
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
			return token, jwttoken, sdk.WrapError(sdk.ErrWrongRequest, "group %d not found", groupID)
		}
	}

	if accessTokenRequest.ExpirationDelaySecond <= 0 {
		accessTokenRequest.ExpirationDelaySecond = 86400 // 1 Day
	}
	expiration := time.Now().Add(time.Duration(accessTokenRequest.ExpirationDelaySecond) * time.Second)

	// Create the token
	token, jwttoken, err = accesstoken.New(u, scopeGroup, accessTokenRequest.Scopes, accessTokenRequest.Origin, accessTokenRequest.Description, expiration)
	if err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	// Insert the token
	if err := accesstoken.Insert(tx, &token); err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	// Commit the token
	if err := tx.Commit(); err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	return token, jwttoken, nil
}

// postNewAccessTokenHandler create a new specific accesstoken with a specific scope (list of groups)
// the JWT token is send through a header X-CDS-JWT
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

		token, jwttoken, err := api.createNewAccessToken(grantedUser.OnBehalfOf, accessTokenRequest)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Set the JWT token as a header
		log.Debug("token.postNewAccessTokenHandler> X-CDS-JWT:%s", jwttoken[:12])
		w.Header().Add("X-CDS-JWT", jwttoken)

		return service.WriteJSON(w, token, http.StatusCreated)
	}
}

// putRegenAccessTokenHandler create a new specific accesstoken with a specific scope (list of groups)
// the JWT token is send through a header X-CDS-JWT
func (api *API) putRegenAccessTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		t, err := accesstoken.FindByID(tx, id)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Only the creator of the token can regen it
		if t.AuthentifiedUserID != getGrantedUser(ctx).OnBehalfOf.ID {
			return sdk.WithStack(sdk.ErrForbidden)
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
		id, err := requestVar(r, "id")
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

		// Check that the current user is member of the group
		g, err := group.LoadGroupByID(api.mustDB(), id)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := group.LoadUserGroup(api.mustDB(), g); err != nil {
			return sdk.WithStack(err)
		}

		oldUser, err := user.GetDeprecatedUser(api.mustDB(), &getGrantedUser(ctx).OnBehalfOf)
		if err != nil {
			return sdk.WithStack(err)
		}

		var isGroupMember bool
		for _, u := range g.Users {
			if u.ID == oldUser.ID {
				isGroupMember = true
				break
			}
		}

		if !isGroupMember {
			for _, u := range g.Admins {
				if u.ID == oldUser.ID {
					isGroupMember = true
					break
				}
			}
		}

		if !isGroupMember {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tokens, err := accesstoken.FindAllByGroup(api.mustDB(), id)
		if err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) deleteAccessTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		t, err := accesstoken.FindByID(api.mustDB(), id)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Only the creator of the token can delete it
		if t.AuthentifiedUserID != getGrantedUser(ctx).OnBehalfOf.ID {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if err := accesstoken.Delete(api.mustDB(), &t); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}
