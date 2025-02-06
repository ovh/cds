package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
)

// getUserGPGKeysHandler Get all gpgkey for the given user
func (api *API) getUserGPGKeysHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			username := vars["user"]
			u, err := user.LoadByUsername(ctx, api.mustDB(), username)
			if err != nil {
				return sdk.WrapError(err, "cannot load user %s", username)
			}

			gpgKeys, err := user.LoadGPGKeysByUserID(ctx, api.mustDB(), u.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, gpgKeys, http.StatusOK)
		}
}

// getUserGPGKeyHandler Get the given user gpg key
func (api *API) getUserGPGKeyHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			gpgKeyID := vars["gpgKeyID"]

			gpgKey, err := user.LoadGPGKeyByKeyID(ctx, api.mustDB(), gpgKeyID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, gpgKey, http.StatusOK)
		}
}

// postUserGPGGKeyHandler Get the given user gpg key
func (api *API) postUserGPGGKeyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isCurrentUser),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			username := vars["user"]

			var gpgKey sdk.UserGPGKey
			if err := service.UnmarshalBody(req, &gpgKey); err != nil {
				return err
			}

			u, err := user.LoadByUsername(ctx, api.mustDB(), username)
			if err != nil {
				return sdk.WrapError(err, "cannot load user %s", username)
			}
			gpgKey.AuthentifiedUserID = u.ID

			publicKey, err := gpg.NewPublicKeyFromPem(gpgKey.PublicKey)
			if err != nil {
				return err
			}
			gpgKey.KeyID = publicKey.KeyShortID()

			tx, err := api.mustDB().Begin()
			if err != nil {
				return err
			}
			if err := user.InsertGPGKey(ctx, tx, &gpgKey); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishUserGPGEvent(ctx, api.Cache, sdk.EventUserGPGKeyCreated, gpgKey, *u)
			return service.WriteJSON(w, gpgKey, http.StatusOK)
		}
}

// getUserGPGKeyHandler Get the given user gpg key
func (api *API) deleteUserGPGKey() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isCurrentUser),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			username := vars["user"]
			gpgKeyID := vars["gpgKeyID"]

			u, err := user.LoadByUsername(ctx, api.mustDB(), username)
			if err != nil {
				return sdk.WrapError(err, "cannot load user %s", username)
			}

			gpgKey, err := user.LoadGPGKeyByKeyID(ctx, api.mustDB(), gpgKeyID)
			if err != nil {
				return sdk.WrapError(err, "cannot load gpgkey %s", gpgKeyID)
			}

			if u.ID != gpgKey.AuthentifiedUserID {
				return sdk.NewErrorFrom(err, "key %s not found on user %s", gpgKeyID, u.ID)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return err
			}
			if err := user.DeleteGPGKey(tx, *gpgKey); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishUserGPGEvent(ctx, api.Cache, sdk.EventUserGPGKeyDeleted, *gpgKey, *u)
			return nil
		}
}
