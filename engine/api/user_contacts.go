package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserContactsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "cannot load user %s", username)
		}

		contacts, err := user.LoadContactsByUserIDs(ctx, api.mustDB(), []string{u.ID})
		if err != nil {
			return err
		}

		return service.WriteJSON(w, contacts, http.StatusOK)
	}
}

func (api *API) putAdminUserContactHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		var req sdk.UserContact
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if req.Value == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing contact value")
		}
		if req.Type == "" {
			req.Type = sdk.UserContactTypeEmail
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		u, err := user.LoadByUsername(ctx, tx, username)
		if err != nil {
			return sdk.WrapError(err, "cannot load user %s", username)
		}

		contacts, err := user.LoadContactsByUserIDs(ctx, tx, []string{u.ID})
		if err != nil {
			return err
		}

		// Find the primary contact of the requested type
		var found *sdk.UserContact
		for i := range contacts {
			if contacts[i].Type == req.Type && contacts[i].Primary {
				found = &contacts[i]
				break
			}
		}
		if found == nil {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "no primary contact of type %s for user %s", req.Type, username)
		}

		// Check uniqueness: no other user should have this contact value
		existing, err := user.LoadContactByTypeAndValue(ctx, tx, req.Type, req.Value)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existing != nil && existing.UserID != u.ID {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "contact value %q already used by another user", req.Value)
		}

		found.Value = req.Value
		if err := user.UpdateContact(ctx, tx, found); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, found, http.StatusOK)
	}
}
