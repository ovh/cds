package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/authentication"
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
		return service.WriteJSON(w, nil, http.StatusNotImplemented)
	}
}

func (api *API) deleteConsumerByUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, nil, http.StatusNotImplemented)
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
