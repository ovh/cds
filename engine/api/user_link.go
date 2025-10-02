package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (api *API) getUserLinksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		links, err := link.LoadUserLinksByUserID(ctx, api.mustDB(), u.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, links, http.StatusOK)
	}
}

func (api *API) deleteUserLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]
		consumerType := vars["consumerType"]

		// Only keep this handler for admin for the moment
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		trackSudo(ctx, w)

		// Retrieve user
		userToUpdate, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		// Load links
		existingLinks, err := link.LoadUserLinksByUserID(ctx, api.mustDB(), userToUpdate.ID)
		if err != nil {
			return err
		}

		var userLink *sdk.UserLink
		for _, l := range existingLinks {
			if l.Type == consumerType {
				userLink = &l
				break
			}
		}
		if userLink == nil {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "user link of type %s does not exists", consumerType)
		}

		// Check admin update
		if userToUpdate.Ring == sdk.UserRingAdmin {
			// Specific audit log for admin: don't change it
			log.Info(ctx, "Administrator user link has been removed (id=%s username: %q) (Link removed: %q)",
				userToUpdate.ID,
				userToUpdate.Username, consumerType,
			)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := link.Delete(ctx, tx, userLink.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit")
		}
		log.Info(ctx, "User link of type %s has been removed for user %s", consumerType, username)
		return service.WriteJSON(w, nil, http.StatusNoContent)
	}
}

func (api *API) postUserLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		// Only keep this handler for admin for the moment
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		trackSudo(ctx, w)

		var data sdk.UserLink
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}

		if data.Type == "" || data.ExternalID == "" || data.Username == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "type, external_id and username are required")
		}

		// Retrieve user id
		userToUpdate, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}
		data.AuthentifiedUserID = userToUpdate.ID

		// Load links
		existingLinks, err := link.LoadUserLinksByUserID(ctx, api.mustDB(), userToUpdate.ID)
		if err != nil {
			return err
		}

		// Check if links exists
		for _, l := range existingLinks {
			if l.Type == data.Type {
				return sdk.NewErrorFrom(sdk.ErrConflictData, "user link of type %s already exists", data.Type)
			}
		}

		// Check admin update
		if userToUpdate.Ring == sdk.UserRingAdmin {
			// Specific audit log for admin: don't change it
			log.Info(ctx, "Administrator user link has been added (id=%s username: %q) (Link added: %q)",
				userToUpdate.ID,
				userToUpdate.Username, data.Username,
			)
		}

		if !sdk.AuthConsumerType(data.Type).IsValid() {
			return sdk.WithStack(sdk.ErrInvalidData)
		}
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		data.Created = time.Now()
		if err := link.Insert(ctx, tx, &data); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit")
		}
		return nil
	}
}
