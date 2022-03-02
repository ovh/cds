package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
)

func (api *API) rbacMiddleware(ctx context.Context, _ http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	for _, checker := range rc.RbacCheckers {
		if err := checker(ctx, api.mustDB(), mux.Vars(req)); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
