package hooks

import (
	"context"
	"io/ioutil"
	"net/http"
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
			return sdk.WrapError(sdk.ErrWrongRequest, "Hook> webhookHandler> invalid uuid")
		}

		webHook := s.Dao.FindLongRunningTask(uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> webhookHandler> unknown uuid")
		}

		req, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Hook> webhookHandler> unable to read request")
		}

		if r.Method != webHook.Config["method"] {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Hook> webhookHandler> Unsupported method %s : %v", r.Method, webHook.Config)
		}

		exec := &LongRunningTaskExecution{
			RequestBody:   req,
			RequestHeader: r.Header,
			RequestURL:    r.URL.RawQuery,
			Timestamp:     time.Now().UnixNano(),
			Type:          webHook.Type,
			UUID:          webHook.UUID,
			Config:        webHook.Config,
		}

		s.Dao.SaveLongRunningTaskExecution(exec)
		s.Dao.EnqueueLongRunningTaskExecution(exec)

		return nil
	}
}
