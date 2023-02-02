package api

import (
	"context"
	builtin2 "github.com/ovh/cds/engine/api/driver/builtin"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getConsumersByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

		var u *sdk.AuthentifiedUser
		var err error
		if username == "me" {
			u, err = user.LoadByID(ctx, api.mustDB(), getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)
		} else {
			u, err = user.LoadByUsername(ctx, api.mustDB(), username)
		}
		if err != nil {
			return err
		}

		cs, err := authentication.LoadUserConsumersByUserID(ctx, api.mustDB(), u.ID,
			authentication.LoadUserConsumerOptions.Default)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, cs, http.StatusOK)
	}
}

func (api *API) postConsumerByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		consumer := getUserConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Only a user can create a consumer for itself, an admin can't create one for an other user
		var u *sdk.AuthentifiedUser
		if username == "me" {
			u, err = user.LoadByID(ctx, tx, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)
		} else {
			u, err = user.LoadByUsername(ctx, tx, username)
		}
		if err != nil {
			return err
		}
		if u.ID != consumer.AuthConsumerUser.AuthentifiedUserID {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "a user can't create a consumer for someone else")
		}

		// Check request data
		var reqData sdk.AuthUserConsumer
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := reqData.IsValid(api.Router.scopeDetails); err != nil {
			return err
		}

		if reqData.ValidityPeriods.Latest() == nil {
			reqData.ValidityPeriods = sdk.NewAuthConsumerValidityPeriod(time.Now(), time.Duration(api.Config.Auth.TokenDefaultDuration)*(24*time.Hour))
		}

		// Create the new built in consumer from request data
		consumerOpts := builtin.NewConsumerOptions{
			Name:                         reqData.Name,
			Description:                  reqData.Description,
			Duration:                     reqData.ValidityPeriods.Latest().Duration,
			GroupIDs:                     reqData.AuthConsumerUser.GroupIDs,
			Scopes:                       reqData.AuthConsumerUser.ScopeDetails,
			ServiceName:                  reqData.AuthConsumerUser.ServiceName,
			ServiceType:                  reqData.AuthConsumerUser.ServiceType,
			ServiceRegion:                reqData.AuthConsumerUser.ServiceRegion,
			ServiceIgnoreJobWithNoRegion: reqData.AuthConsumerUser.ServiceIgnoreJobWithNoRegion,
		}
		newConsumer, token, err := builtin.NewConsumer(ctx, tx, consumerOpts, consumer)
		if err != nil {
			return err
		}
		if err := authentication.LoadUserConsumerOptions.Default(ctx, tx, newConsumer); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, sdk.AuthConsumerCreateResponse{
			Token:    token,
			Consumer: newConsumer,
		}, http.StatusCreated)
	}
}

func (api *API) deleteConsumerByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		consumerID := vars["permConsumerID"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		consumer, err := authentication.LoadUserConsumerByID(ctx, tx, consumerID)
		if err != nil {
			return err
		}

		if consumer.Type != sdk.ConsumerBuiltin {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "can't delete a no builtin consumer")
		}

		if err := authentication.DeleteConsumerByID(tx, consumer.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postConsumerRegenByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		consumerID := vars["permConsumerID"]

		var req sdk.AuthConsumerRegenRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Load the consumer from the input
		consumer, err := authentication.LoadUserConsumerByID(ctx, tx, consumerID)
		if err != nil {
			return err
		}

		if req.OverlapDuration == "" {
			req.OverlapDuration = api.Config.Auth.TokenOverlapDefaultDuration
		}
		if req.NewDuration == 0 {
			req.NewDuration = api.Config.Auth.TokenDefaultDuration
		}
		var overlapDuration time.Duration
		if req.OverlapDuration != "" {
			overlapDuration, err = time.ParseDuration(req.OverlapDuration)
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
		}

		newDuration := time.Duration(req.NewDuration) * (24 * time.Hour)

		if overlapDuration > newDuration {
			return sdk.NewError(sdk.ErrWrongRequest, errors.New("invalid duration"))
		}

		if err := authentication.ConsumerRegen(ctx, tx, consumer,
			overlapDuration,
			newDuration,
		); err != nil {
			return err
		}

		jws, err := builtin2.NewSigninConsumerToken(consumer) // Regen a new jws (signin token)
		if err != nil {
			return err
		}

		if req.RevokeSessions {
			sessions, err := authentication.LoadSessionsByConsumerIDs(ctx, tx, []string{consumer.ID}) // Find all the sessions
			if err != nil {
				return err
			}
			for _, s := range sessions { // Now remove all current sessions for the consumer
				if err := authentication.DeleteSessionByID(tx, s.ID); err != nil {
					return err
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, sdk.AuthConsumerCreateResponse{
			Token:    jws,
			Consumer: consumer,
		}, http.StatusOK)
	}
}

func (api *API) getSessionsByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

		var u *sdk.AuthentifiedUser
		var err error
		if username == "me" {
			u, err = user.LoadByID(ctx, api.mustDB(), getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)
		} else {
			u, err = user.LoadByUsername(ctx, api.mustDB(), username)
		}
		if err != nil {
			return err
		}

		cs, err := authentication.LoadUserConsumersByUserID(ctx, api.mustDB(), u.ID)
		if err != nil {
			return err
		}

		ss, err := authentication.LoadSessionsByConsumerIDs(ctx, api.mustDB(), sdk.AuthConsumersToIDs(cs))
		if err != nil {
			return err
		}

		// Set extra data on sessions
		currentSession := getAuthSession(ctx)
		for i := range ss {
			if ss[i].ID == currentSession.ID {
				ss[i].Current = true
			}
			if ss[i].MFA {
				active, lastActivity, err := authentication.GetSessionActivity(api.Cache, ss[i].ID)
				if err != nil {
					log.Warn(ctx, "getSessionsByUserHandler> cannot get session activity for %s", ss[i].ID)
					continue
				}
				if active {
					ss[i].LastActivity = &lastActivity
				}
			}
		}

		return service.WriteJSON(w, ss, http.StatusOK)
	}
}

func (api *API) deleteSessionByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		sessionID := vars["permSessionID"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		session, err := authentication.LoadSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}

		if err := authentication.DeleteSessionByID(tx, session.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
