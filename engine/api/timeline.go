package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := getUser(ctx)

		filter := sdk.EventFilter{
			ProjectKeys: make([]string, 0, len(u.Permissions.ProjectsPerm)),
		}

		for k := range u.Permissions.ProjectsPerm {
			filter.ProjectKeys = append(filter.ProjectKeys, k)
		}

		events, err := event.GetEvents(api.mustDB(), api.Cache, filter)
		if err != nil {
			return sdk.WrapError(err, "getTimelineHandler> Unable to load events")
		}
		return WriteJSON(w, events, http.StatusOK)
	}
}
