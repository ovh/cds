package hooks

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

func (s *Service) webhookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "webhookHandler> invalid uuid")
		}

		webHook := s.Dao.FindLongRunningTask(uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "webhookHandler> unknown uuid")
		}

		return s.doWebHook(webHook, r)
	}
}
