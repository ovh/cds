package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getConsumersByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)

		cs, err := authentication.LoadConsumersByUserID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, cs, http.StatusOK)
	}
}

func (api *API) postConsumerByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var reqData sdk.AuthConsumer
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := reqData.IsValid(); err != nil {
			return err
		}

		consumer := getAPIConsumer(ctx)

		// Create the new built in consumer from request data
		newConsumer, token, err := builtin.NewConsumer(api.mustDB(), reqData.Name, reqData.Description,
			consumer, reqData.GroupIDs, reqData.Scopes)
		if err != nil {
			return err
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

		consumer, err := authentication.LoadConsumerByID(ctx, tx, consumerID)
		if err != nil {
			return err
		}
		if consumer.Type != sdk.ConsumerBuiltin {
			return sdk.WithStack(sdk.ErrForbidden)
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

func (api *API) getSessionsByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, nil, http.StatusNotImplemented)
	}
}

func (api *API) deleteSessionByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, nil, http.StatusNotImplemented)
	}
}
