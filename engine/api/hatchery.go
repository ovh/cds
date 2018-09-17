package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) hatcheryCountHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wfNodeRunID, err := requestVarInt(r, "workflowNodeRunID")
		if err != nil {
			return sdk.WrapError(err, "cannot convert workflow node run ID")
		}

		count, err := hatchery.CountHatcheries(api.mustDB(), wfNodeRunID)
		if err != nil {
			return sdk.WrapError(err, "hatcheryCountHandler> cannot get hatcheries count")
		}

		return service.WriteJSON(w, count, http.StatusOK)
	}
}
