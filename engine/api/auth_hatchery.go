package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/hatchery"
	hatch "github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) postAuthHatcherySigninHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Extract and validate signin request
			var req sdk.AuthConsumerHatcherySigninRequest
			if err := service.UnmarshalBody(r, &req); err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}
			consumerID, err := hatchery.CheckSigninRequest(req)
			if err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}

			// Check if a consumer exists
			consumer, err := authentication.LoadHatcheryConsumerByID(ctx, api.mustDB(), consumerID)
			if err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}

			h, err := hatch.LoadHatcheryByID(ctx, api.mustDB(), consumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}
			defer tx.Rollback() // nolint

			// Check the Token validity againts the IAT attribute
			if _, err := hatchery.CheckSigninConsumerTokenIssuedAt(ctx, req.Token, consumer); err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}

			// Generate a new session for consumer
			session, err := authentication.NewSession(ctx, tx, &consumer.AuthConsumer, hatchery.SessionDuration)
			if err != nil {
				return err
			}

			// Store the last authentication date on the consumer
			now := time.Now()
			consumer.LastAuthentication = &now
			if err := authentication.UpdateConsumerLastAuthentication(ctx, tx, &consumer.AuthConsumer); err != nil {
				return err
			}

			// Generate a jwt for current session
			jwt, err := authentication.NewSessionJWT(session, "")
			if err != nil {
				return err
			}

			// Set those value in ctx
			ctx = context.WithValue(ctx, contextHatcheryConsumer, consumer)
			ctx = context.WithValue(ctx, cdslog.AuthHatcheryID, consumer.AuthConsumerHatchery.HatcheryID)
			SetTracker(w, cdslog.AuthHatcheryID, consumer.AuthConsumerHatchery.HatcheryID)
			ctx = context.WithValue(ctx, cdslog.AuthConsumerID, consumer.ID)
			SetTracker(w, cdslog.AuthConsumerID, consumer.ID)

			ctx = context.WithValue(ctx, contextSession, session)
			ctx = context.WithValue(ctx, cdslog.AuthSessionID, session.ID)
			SetTracker(w, cdslog.AuthSessionID, session.ID)
			ctx = context.WithValue(ctx, cdslog.AuthSessionIAT, session.Created.Unix())
			SetTracker(w, cdslog.AuthSessionIAT, session.Created.Unix())
			ctx = context.WithValue(ctx, contextSession, session)

			var driverManifest *sdk.AuthDriverManifest
			m := hatchery.GetManifest()
			driverManifest = &m

			if driverManifest == nil {
				return sdk.WrapError(sdk.ErrUnauthorized, "consumer driver (%s) was not found", consumer.Type)
			}
			ctx = context.WithValue(ctx, contextDriverManifest, driverManifest)

			// Set a cookie with the jwt token
			api.SetCookie(w, service.JWTCookieName, jwt, session.ExpireAt, true)

			// Prepare http response
			resp := sdk.AuthConsumerHatcherySigninResponse{
				Token:    jwt,
				Hatchery: *h,
				APIURL:   api.Config.URL.API,
				Uptodate: req.Version == sdk.VERSION,
			}

			if err := api.hatcheryRegister(ctx, tx, *consumer, session.ID, h, req); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			pubKey, err := jws.ExportPublicKey(authentication.GetSigningKey())
			if err != nil {
				return sdk.WrapError(err, "Unable to export public signing key")
			}

			encodedPubKey := base64.StdEncoding.EncodeToString(pubKey)
			w.Header().Set("X-Api-Pub-Signing-Key", encodedPubKey)

			return service.WriteJSON(w, resp, http.StatusOK)
		}
}

// This has to be called by the signin handler
func (api *API) hatcheryRegister(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, consumer sdk.AuthHatcheryConsumer, sessionID string, h *sdk.Hatchery, signInRequest sdk.AuthConsumerHatcherySigninRequest) error {
	h.Name = signInRequest.Name
	h.HTTPURL = signInRequest.HTTPURL
	h.Config = signInRequest.Config
	h.PublicKey = signInRequest.PublicKey
	h.ModelType = signInRequest.HatcheryType
	h.LastHeartbeat = time.Now()

	if err := hatch.Update(ctx, tx, h); err != nil {
		return err
	}
	log.Debug(ctx, "hatcheryRegister> update existing hatchery %s(%d) registered for consumer %s", h.Name, h.ID, consumer.ID)

	hs := sdk.HatcheryStatus{
		HatcheryID: h.ID,
		SessionID:  sessionID,
	}
	if err := hatch.UpsertStatus(ctx, tx, h.ID, &hs); err != nil {
		return sdk.WithStack(err)
	}

	if len(h.PublicKey) > 0 {
		log.Debug(ctx, "hatcheryRegister> hatchery %s registered with public key: %s", h.Name, string(h.PublicKey))
	}

	// TODO
	//if err := worker.ReAttachAllToHatchery(ctx, tx, *srv); err != nil {
	//	return err
	//}
	return nil
}
