package hooks

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

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

		req, err := httputil.DumpRequest(r, true)
		if err != nil {
			return sdk.WrapError(err, "webhookHandler> unsupported request")
		}

		exec := &LongRunningTaskExecution{
			Request:   req,
			Timestamp: time.Now().UnixNano(),
			Type:      webHook.Type,
			UUID:      webHook.UUID,
		}

		s.Dao.SaveLongRunningTaskExecution(exec)
		s.Dao.EnqueueLongRunningTaskExecution(exec)

		return nil
	}
}
