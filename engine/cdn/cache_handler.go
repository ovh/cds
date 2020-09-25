package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) deleteCacheHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.WrapError(s.LogCache.Clear(), "unable to clear log cache")
	}
}

func (s *Service) getStatusCacheHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.LogCache.Status(ctx), status)
	}
}
