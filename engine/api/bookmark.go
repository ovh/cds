package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/bookmark"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getBookmarksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getUserConsumer(ctx)
		data, err := bookmark.LoadAll(api.mustDB(), consumer.AuthConsumerUser.AuthentifiedUser.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}
