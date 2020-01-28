package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/bookmark"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getBookmarksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)
		data, err := bookmark.LoadAll(api.mustDB(), consumer.AuthentifiedUser)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}
